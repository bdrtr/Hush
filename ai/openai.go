package ai

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

// OpenAIClient implements the Client interface for OpenAI API compatible services
type OpenAIClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// WithBaseURL overrides the default OpenAI base URL (useful for Groq, DeepSeek, etc.)
func (c *OpenAIClient) WithBaseURL(url string) *OpenAIClient {
	c.BaseURL = url
	return c
}

type openAIRequest struct {
	Model       string                 `json:"model"`
	Messages    []Message              `json:"messages"`
	Stream      bool                   `json:"stream"`
	Temperature float32                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
}

type openAIResponse struct {
	Choices []struct {
		Message Message `json:"message"`
		Delta   Message `json:"delta"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Generate generates a non-streaming response
func (c *OpenAIClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	apiURL := fmt.Sprintf("%s/chat/completions", c.BaseURL)

	oaiReq := openAIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      false,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	body, err := sonic.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned status: %d", resp.StatusCode)
	}

	var oaiResp openAIResponse
	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if oaiResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", oaiResp.Error.Message)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return &GenerateResponse{
		Content: oaiResp.Choices[0].Message.Content,
		Usage: Usage{
			PromptTokens:     oaiResp.Usage.PromptTokens,
			CompletionTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:      oaiResp.Usage.TotalTokens,
		},
	}, nil
}

// Stream generates a streaming response
func (c *OpenAIClient) Stream(ctx context.Context, req GenerateRequest, ch chan<- StreamChunk) {
	defer close(ch)

	apiURL := fmt.Sprintf("%s/chat/completions", c.BaseURL)

	oaiReq := openAIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      true,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	body, err := sonic.Marshal(oaiReq)
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
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		ch <- StreamChunk{Error: err}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- StreamChunk{Error: fmt.Errorf("openai returned status: %d", resp.StatusCode)}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		
		data := strings.TrimPrefix(line, "data: ")
		
		if data == "[DONE]" {
			ch <- StreamChunk{Done: true}
			break
		}

		var chunk openAIResponse
		if err := sonic.UnmarshalString(data, &chunk); err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to decode chunk: %w", err)}
			return
		}

		if chunk.Error != nil {
			ch <- StreamChunk{Error: fmt.Errorf("openai error: %s", chunk.Error.Message)}
			return
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				ch <- StreamChunk{
					Content: content,
					Done:    false,
				}
			}
		}
	}

	if err := scanner.Err(); err != nil && err != context.Canceled {
		ch <- StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
	}
}
