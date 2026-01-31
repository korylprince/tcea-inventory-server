package chatbot

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"sync"

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

	// Tool call loop
	for {
		resp, err := h.client.Chat(ctx, messages, tools)
		if err != nil {
			h.sendError(conn, "AI request failed: "+err.Error())
			return
		}

		if len(resp.Choices) == 0 {
			h.sendError(conn, "No response from AI")
			return
		}

		choice := resp.Choices[0]
		assistantMsg := choice.Message

		// Add assistant message to history
		messages = append(messages, assistantMsg)
		newMessages = append(newMessages, assistantMsg)

		// Check if we have tool calls
		if len(assistantMsg.ToolCalls) == 0 {
			break // No more tool calls, proceed to final response
		}

		// Execute all tool calls in parallel
		toolResults := h.executeToolsParallel(ctx, assistantMsg.ToolCalls)

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

	// Stream final response (without tools)
	streamCh, err := h.client.ChatStream(ctx, messages)
	if err != nil {
		h.sendError(conn, "Streaming failed: "+err.Error())
		return
	}

	var fullResponse string
	for chunk := range streamCh {
		if chunk.Err != nil {
			h.sendError(conn, "Stream error: "+chunk.Err.Error())
			return
		}
		if chunk.Content != "" {
			fullResponse += chunk.Content
			if err := conn.WriteJSON(ServerMessage{
				Type:    MessageTypeText,
				Content: chunk.Content,
			}); err != nil {
				log.Printf("Failed to write chunk: %v", err)
				return
			}
		}
	}

	// Add final assistant response to messages
	if fullResponse != "" {
		newMessages = append(newMessages, Message{Role: "assistant", Content: &fullResponse})
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

func (h *Handler) executeToolsParallel(ctx context.Context, calls []ToolCall) []toolResult {
	results := make([]toolResult, len(calls))
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, tc ToolCall) {
			defer wg.Done()
			content, err := h.executor.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				content = `{"error": "` + err.Error() + `"}`
			}
			results[idx] = toolResult{
				id:      tc.ID,
				name:    tc.Function.Name,
				content: content,
			}
		}(i, call)
	}

	wg.Wait()
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
