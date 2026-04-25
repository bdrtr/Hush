package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaClient_Generate(t *testing.T) {
	// Mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected path '/api/chat', got %s", r.URL.Path)
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"model": "llama3",
			"message": {
				"role": "assistant",
				"content": "Hello, world!"
			},
			"done": true,
			"prompt_eval_count": 10,
			"eval_count": 5
		}`))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL)
	req := GenerateRequest{
		Model: "llama3",
		Messages: []Message{
			{Role: RoleUser, Content: "Say hello"},
		},
	}

	resp, err := client.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", resp.Content)
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestOpenAIClient_Generate(t *testing.T) {
	// Mock OpenAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path '/chat/completions', got %s", r.URL.Path)
		}
		
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected auth header 'Bearer test-api-key', got '%s'", auth)
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "Hi there!"
				}
			}],
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 5,
				"total_tokens": 15
			}
		}`))
	}))
	defer server.Close()

	client := NewOpenAIClient("test-api-key").WithBaseURL(server.URL)
	req := GenerateRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "Say hi"},
		},
	}

	resp, err := client.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp.Content != "Hi there!" {
		t.Errorf("Expected 'Hi there!', got '%s'", resp.Content)
	}
}
