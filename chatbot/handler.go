package chatbot

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/korylprince/tcea-inventory-server/api"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler handles WebSocket chat connections
type Handler struct {
	store    ConversationStore
	client   *AIClient
	executor *ToolExecutor
	db       *sql.DB
}

// NewHandler creates a new chat handler
func NewHandler(store ConversationStore, client *AIClient, db *sql.DB) *Handler {
	return &Handler{
		store:    store,
		client:   client,
		executor: NewToolExecutor(),
		db:       db,
	}
}

// ServeHTTP handles the WebSocket upgrade and chat flow
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	user := r.Context().Value(api.UserKey).(*api.User)

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Get or create conversation
	conversationID := r.URL.Query().Get("conversation_id")
	var conv *Conversation

	if conversationID != "" {
		conv, err = h.store.Get(conversationID)
		if err != nil {
			h.sendError(conn, "Failed to load conversation")
			return
		}
	}

	if conv == nil {
		conv, err = h.store.Create()
		if err != nil {
			h.sendError(conn, "Failed to create conversation")
			return
		}
	}

	// Read user message
	var clientMsg ClientMessage
	if err := conn.ReadJSON(&clientMsg); err != nil {
		h.sendError(conn, "Failed to read message")
		return
	}

	if clientMsg.Message == "" {
		h.sendError(conn, "Message cannot be empty")
		return
	}

	// Build messages for AI
	messages := h.buildMessages(conv, clientMsg.Message)
	tools := GetTools()

	// Create a new transaction for tool execution
	tx, err := h.db.Begin()
	if err != nil {
		h.sendError(conn, "Database error")
		return
	}
	defer tx.Rollback()

	// Create context with transaction and user
	ctx := context.WithValue(r.Context(), api.TransactionKey, tx)
	ctx = context.WithValue(ctx, api.UserKey, user)

	// Track all messages to save
	var newMessages []Message
	content := clientMsg.Message
	newMessages = append(newMessages, Message{Role: "user", Content: &content})

	// Streaming loop with tool support
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		streamCh, err := h.client.ChatStreamWithTools(ctx, messages, tools)
		if err != nil {
			h.sendError(conn, "AI request failed: "+err.Error())
			return
		}

		// Accumulate the full response
		var fullContent string
		var toolCalls []ToolCall
		var finishReason string

		for chunk := range streamCh {
			if chunk.Err != nil {
				h.sendError(conn, "Stream error: "+chunk.Err.Error())
				return
			}

			// Stream content to client immediately
			if chunk.Content != "" {
				fullContent += chunk.Content
				if err := conn.WriteJSON(ServerMessage{
					Type:    MessageTypeText,
					Content: chunk.Content,
				}); err != nil {
					log.Printf("Failed to write chunk: %v", err)
					return
				}
			}

			// Collect tool calls
			if len(chunk.ToolCalls) > 0 {
				toolCalls = chunk.ToolCalls
			}

			if chunk.FinishReason != "" {
				finishReason = chunk.FinishReason
			}
		}

		// Build the assistant message from the streamed response
		assistantMsg := Message{Role: "assistant"}
		if fullContent != "" {
			assistantMsg.Content = &fullContent
		}
		if len(toolCalls) > 0 {
			assistantMsg.ToolCalls = toolCalls
		}

		// Add to history
		messages = append(messages, assistantMsg)
		newMessages = append(newMessages, assistantMsg)

		// Check if we're done (no tool calls)
		if finishReason == "stop" || len(toolCalls) == 0 {
			break
		}

		// Execute tool calls sequentially (parallel execution causes MySQL connection issues)
		toolResults := h.executeToolsSequential(ctx, toolCalls)

		// Add tool results to messages
		for _, tr := range toolResults {
			toolMsg := Message{
				Role:       "tool",
				Content:    &tr.content,
				ToolCallID: tr.id,
				Name:       tr.name,
			}
			messages = append(messages, toolMsg)
			newMessages = append(newMessages, toolMsg)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		h.sendError(conn, "Failed to save changes")
		return
	}

	// Save conversation
	if err := h.store.AddMessages(conv.ID, newMessages); err != nil {
		log.Printf("Failed to save conversation: %v", err)
	}

	// Send done message
	conn.WriteJSON(ServerMessage{
		Type:           MessageTypeDone,
		ConversationID: conv.ID,
	})
}

type toolResult struct {
	id      string
	name    string
	content string
}

func (h *Handler) executeToolsSequential(ctx context.Context, calls []ToolCall) []toolResult {
	results := make([]toolResult, len(calls))

	for i, call := range calls {
		content, err := h.executor.Execute(ctx, call.Function.Name, call.Function.Arguments)
		if err != nil {
			content = `{"error": "` + err.Error() + `"}`
		}
		results[i] = toolResult{
			id:      call.ID,
			name:    call.Function.Name,
			content: content,
		}
	}

	return results
}

func (h *Handler) buildMessages(conv *Conversation, userMessage string) []Message {
	messages := []Message{
		{Role: "system", Content: strPtr(SystemPrompt())},
	}

	// Add conversation history
	messages = append(messages, conv.Messages...)

	// Add new user message
	messages = append(messages, Message{Role: "user", Content: &userMessage})

	return messages
}

func (h *Handler) sendError(conn *websocket.Conn, msg string) {
	conn.WriteJSON(ServerMessage{
		Type:  MessageTypeError,
		Error: msg,
	})
}

func strPtr(s string) *string {
	return &s
}
