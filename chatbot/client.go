package chatbot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Tool represents an OpenAI function tool
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function tool
type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ChatRequest is the request body for the chat completions API
type ChatRequest struct {
	Model      string    `json:"model"`
	Messages   []Message `json:"messages"`
	Tools      []Tool    `json:"tools,omitempty"`
	ToolChoice string    `json:"tool_choice,omitempty"` // "auto", "none", or specific
	Stream     bool      `json:"stream"`
}

// ChatResponse is the response from the chat completions API
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice represents a single completion choice
type Choice struct {
	Index        int      `json:"index"`
	Message      Message  `json:"message"`
	Delta        *Message `json:"delta,omitempty"`
	FinishReason string   `json:"finish_reason"`
}

// Usage contains token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a chunk from the streaming response
type StreamChunk struct {
	Content      string     // Text content delta
	ToolCalls    []ToolCall // Tool calls (sent in one chunk usually)
	FinishReason string     // "stop", "tool_calls", or empty
	Err          error
}

// StreamToolCall represents a tool call delta in streaming
type StreamToolCall struct {
	Index    int          `json:"index"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function,omitempty"`
}

// AIClient is a client for the OpenAI-compatible API
type AIClient struct {
	endpoint   string
	model      string
	httpClient *http.Client
}

// NewAIClient creates a new AI client
func NewAIClient(endpoint, model string) *AIClient {
	return &AIClient{
		endpoint:   endpoint,
		model:      model,
		httpClient: &http.Client{},
	}
}

// Chat makes a non-streaming chat request (for tool calls)
func (c *AIClient) Chat(ctx context.Context, messages []Message, tools []Tool) (*ChatResponse, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}
	if len(tools) > 0 {
		req.ToolChoice = "auto"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// ChatStreamWithTools makes a streaming chat request that handles both content and tool calls
func (c *AIClient) ChatStreamWithTools(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamChunk, error) {
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	}
	if len(tools) > 0 {
		req.ToolChoice = "auto"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// Accumulate tool calls across chunks
		toolCallsMap := make(map[int]*ToolCall)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var streamResp struct {
				Choices []struct {
					Index        int    `json:"index"`
					FinishReason string `json:"finish_reason"`
					Delta        struct {
						Role      string           `json:"role,omitempty"`
						Content   *string          `json:"content,omitempty"`
						ToolCalls []StreamToolCall `json:"tool_calls,omitempty"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				ch <- StreamChunk{Err: fmt.Errorf("failed to parse SSE data: %w", err)}
				return
			}

			if len(streamResp.Choices) == 0 {
				continue
			}

			choice := streamResp.Choices[0]
			chunk := StreamChunk{FinishReason: choice.FinishReason}

			// Handle content
			if choice.Delta.Content != nil && *choice.Delta.Content != "" {
				chunk.Content = *choice.Delta.Content
			}

			// Handle tool calls - accumulate them
			for _, tc := range choice.Delta.ToolCalls {
				if _, exists := toolCallsMap[tc.Index]; !exists {
					toolCallsMap[tc.Index] = &ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				} else {
					// Append arguments
					toolCallsMap[tc.Index].Function.Arguments += tc.Function.Arguments
				}
			}

			// If finish_reason is tool_calls, send the accumulated tool calls
			if choice.FinishReason == "tool_calls" {
				var toolCalls []ToolCall
				for i := 0; i < len(toolCallsMap); i++ {
					if tc, ok := toolCallsMap[i]; ok {
						toolCalls = append(toolCalls, *tc)
					}
				}
				chunk.ToolCalls = toolCalls
			}

			// Only send chunk if there's something useful
			if chunk.Content != "" || chunk.FinishReason != "" || len(chunk.ToolCalls) > 0 {
				ch <- chunk
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("failed to read stream: %w", err)}
		}
	}()

	return ch, nil
}

// ChatStream makes a streaming chat request (for final response without tools)
func (c *AIClient) ChatStream(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	return c.ChatStreamWithTools(ctx, messages, nil)
}
