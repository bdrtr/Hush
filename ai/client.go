package ai

import (
	"context"
)

// Role represents the role of a message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a single chat message
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// GenerateRequest represents a request to generate a completion
type GenerateRequest struct {
	Model       string
	Messages    []Message
	Temperature float32
	MaxTokens   int
	Stream      bool
}

// GenerateResponse represents a non-streaming response
type GenerateResponse struct {
	Content string
	Usage   Usage
}

// StreamChunk represents a chunk of a streaming response
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Client represents a generic LLM client interface
type Client interface {
	// Generate generates a complete response (blocking)
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	
	// Stream generates a streaming response, sending chunks to the provided channel
	Stream(ctx context.Context, req GenerateRequest, ch chan<- StreamChunk)
}
