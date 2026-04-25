package ai

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

// OllamaClient implements the Client interface for Ollama
type OllamaClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type ollamaRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ollamaResponse struct {
	Model   string  `json:"model"`
	Message Message `json:"message"`
	Done    bool    `json:"done"`
	Error   string  `json:"error,omitempty"`
	// Stats
	PromptEvalCount int `json:"prompt_eval_count,omitempty"`
	EvalCount       int `json:"eval_count,omitempty"`
}

// Generate generates a non-streaming response
func (c *OllamaClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	apiURL := fmt.Sprintf("%s/api/chat", c.BaseURL)

	ollamaReq := ollamaRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
	}

	if req.Temperature > 0 || req.MaxTokens > 0 {
		ollamaReq.Options = make(map[string]interface{})
		if req.Temperature > 0 {
			ollamaReq.Options["temperature"] = req.Temperature
		}
		if req.MaxTokens > 0 {
			ollamaReq.Options["num_predict"] = req.MaxTokens
		}
	}

	body, err := sonic.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status: %d", resp.StatusCode)
	}

	var ollamaResp ollamaResponse
	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if ollamaResp.Error != "" {
		return nil, fmt.Errorf("ollama error: %s", ollamaResp.Error)
	}

	return &GenerateResponse{
		Content: ollamaResp.Message.Content,
		Usage: Usage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}, nil
}

// Stream generates a streaming response
func (c *OllamaClient) Stream(ctx context.Context, req GenerateRequest, ch chan<- StreamChunk) {
	defer close(ch)

	apiURL := fmt.Sprintf("%s/api/chat", c.BaseURL)

	ollamaReq := ollamaRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
	}

	if req.Temperature > 0 || req.MaxTokens > 0 {
		ollamaReq.Options = make(map[string]interface{})
		if req.Temperature > 0 {
			ollamaReq.Options["temperature"] = req.Temperature
		}
		if req.MaxTokens > 0 {
			ollamaReq.Options["num_predict"] = req.MaxTokens
		}
	}

	body, err := sonic.Marshal(ollamaReq)
	if err != nil {
		ch <- StreamChunk{Error: fmt.Errorf("failed to marshal request: %w", err)}
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		ch <- StreamChunk{Error: err}
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		ch <- StreamChunk{Error: err}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- StreamChunk{Error: fmt.Errorf("ollama returned status: %d", resp.StatusCode)}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk ollamaResponse
		if err := sonic.Unmarshal(line, &chunk); err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to decode chunk: %w", err)}
			return
		}

		if chunk.Error != "" {
			ch <- StreamChunk{Error: fmt.Errorf("ollama error: %s", chunk.Error)}
			return
		}

		ch <- StreamChunk{
			Content: chunk.Message.Content,
			Done:    chunk.Done,
		}

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil && err != context.Canceled {
		ch <- StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
	}
}
