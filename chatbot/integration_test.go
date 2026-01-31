package chatbot_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/korylprince/tcea-inventory-server/api"
	"github.com/korylprince/tcea-inventory-server/chatbot"
)

// MockDB implements a minimal mock for database operations
type MockDB struct {
	devices   []*api.Device
	models    []*api.Model
	statuses  []api.Status
	locations []api.Location
}

func NewMockDB() *MockDB {
	return &MockDB{
		devices: []*api.Device{
			{ID: 1, SerialNumber: "SN001", ModelID: 1, Status: "Available", Location: "Storage",
				Model: &api.Model{ID: 1, Manufacturer: "Dell", Model: "Latitude 5520"}},
			{ID: 2, SerialNumber: "SN002", ModelID: 1, Status: "In Use", Location: "Room 101",
				Model: &api.Model{ID: 1, Manufacturer: "Dell", Model: "Latitude 5520"}},
			{ID: 3, SerialNumber: "SN003", ModelID: 2, Status: "Available", Location: "Storage",
				Model: &api.Model{ID: 2, Manufacturer: "HP", Model: "EliteBook 840"}},
		},
		models: []*api.Model{
			{ID: 1, Manufacturer: "Dell", Model: "Latitude 5520"},
			{ID: 2, Manufacturer: "HP", Model: "EliteBook 840"},
			{ID: 3, Manufacturer: "Lenovo", Model: "ThinkPad T14"},
		},
		statuses:  []api.Status{"Available", "In Use", "Broken", "Storage"},
		locations: []api.Location{"Storage", "Room 101", "Room 102", "IT Office"},
	}
}

// MockTx implements a mock sql.Tx that provides test data
type MockTx struct {
	db *MockDB
}

func (m *MockTx) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *MockTx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockTx) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

func (m *MockTx) Commit() error   { return nil }
func (m *MockTx) Rollback() error { return nil }

// MockToolExecutor overrides tool execution with mock data
type MockToolExecutor struct {
	db *MockDB
}

func (e *MockToolExecutor) Execute(ctx context.Context, name string, arguments string) (string, error) {
	var args map[string]interface{}
	if arguments != "" {
		json.Unmarshal([]byte(arguments), &args)
	}

	var result interface{}

	switch name {
	case "query_devices":
		// Filter devices based on args
		filtered := []*api.Device{}
		for _, d := range e.db.devices {
			match := true
			if status, ok := args["status"].(string); ok && status != "" {
				if !strings.Contains(strings.ToLower(string(d.Status)), strings.ToLower(status)) {
					match = false
				}
			}
			if location, ok := args["location"].(string); ok && location != "" {
				if !strings.Contains(strings.ToLower(string(d.Location)), strings.ToLower(location)) {
					match = false
				}
			}
			if search, ok := args["search"].(string); ok && search != "" {
				searchLower := strings.ToLower(search)
				found := strings.Contains(strings.ToLower(d.SerialNumber), searchLower) ||
					strings.Contains(strings.ToLower(d.Model.Manufacturer), searchLower) ||
					strings.Contains(strings.ToLower(d.Model.Model), searchLower)
				if !found {
					match = false
				}
			}
			if match {
				filtered = append(filtered, d)
			}
		}
		result = filtered

	case "get_device":
		id := int64(args["id"].(float64))
		for _, d := range e.db.devices {
			if d.ID == id {
				result = d
				break
			}
		}
		if result == nil {
			result = map[string]string{"error": "device not found"}
		}

	case "query_models":
		result = e.db.models

	case "get_model":
		id := int64(args["id"].(float64))
		for _, m := range e.db.models {
			if m.ID == id {
				result = m
				break
			}
		}
		if result == nil {
			result = map[string]string{"error": "model not found"}
		}

	case "get_statuses":
		result = e.db.statuses

	case "get_locations":
		result = e.db.locations

	case "get_stats":
		result = map[string]interface{}{
			"device_count":   len(e.db.devices),
			"model_count":    len(e.db.models),
			"location_count": len(e.db.locations),
			"statuses": []map[string]interface{}{
				{"status": "Available", "count": 2},
				{"status": "In Use", "count": 1},
			},
		}

	case "create_device":
		result = map[string]interface{}{"id": 100, "message": "device created successfully"}

	case "update_device":
		result = map[string]string{"message": "device updated successfully"}

	case "add_device_note":
		result = map[string]interface{}{"event_id": 1, "message": "note added successfully"}

	case "create_model":
		result = map[string]interface{}{"id": 100, "message": "model created successfully"}

	case "update_model":
		result = map[string]string{"message": "model updated successfully"}

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	data, _ := json.Marshal(result)
	return string(data), nil
}

// TestHandler wraps the real handler but uses mock tool executor
type TestHandler struct {
	store    chatbot.ConversationStore
	client   *chatbot.AIClient
	executor *MockToolExecutor
	mockDB   *MockDB
}

func NewTestHandler(aiEndpoint, aiModel string) *TestHandler {
	mockDB := NewMockDB()
	return &TestHandler{
		store:    chatbot.NewLRUStore(10 * 1024 * 1024),
		client:   chatbot.NewAIClient(aiEndpoint, aiModel),
		executor: &MockToolExecutor{db: mockDB},
		mockDB:   mockDB,
	}
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create mock user context
	user := &api.User{ID: 1, Email: "test@example.com", Name: "Test User"}
	ctx := context.WithValue(r.Context(), api.UserKey, user)
	r = r.WithContext(ctx)

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Get or create conversation
	conversationID := r.URL.Query().Get("conversation_id")
	var conv *chatbot.Conversation

	if conversationID != "" {
		conv, _ = h.store.Get(conversationID)
	}
	if conv == nil {
		conv, _ = h.store.Create()
	}

	// Read user message
	var clientMsg chatbot.ClientMessage
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
	tools := chatbot.GetTools()

	// Track all messages to save
	var newMessages []chatbot.Message
	content := clientMsg.Message
	newMessages = append(newMessages, chatbot.Message{Role: "user", Content: &content})

	// Tool call loop
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		resp, err := h.client.Chat(r.Context(), messages, tools)
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

		if len(assistantMsg.ToolCalls) == 0 {
			// No tool calls - send final response
			if assistantMsg.Content != nil && *assistantMsg.Content != "" {
				conn.WriteJSON(chatbot.ServerMessage{
					Type:    chatbot.MessageTypeText,
					Content: *assistantMsg.Content,
				})
				newMessages = append(newMessages, assistantMsg)
			}
			break
		}

		// Add assistant message with tool calls
		messages = append(messages, assistantMsg)
		newMessages = append(newMessages, assistantMsg)

		// Execute tool calls using mock executor
		for _, tc := range assistantMsg.ToolCalls {
			result, err := h.executor.Execute(r.Context(), tc.Function.Name, tc.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf(`{"error": "%s"}`, err.Error())
			}

			toolMsg := chatbot.Message{
				Role:       "tool",
				Content:    &result,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			}
			messages = append(messages, toolMsg)
			newMessages = append(newMessages, toolMsg)
		}
	}

	h.store.AddMessages(conv.ID, newMessages)

	conn.WriteJSON(chatbot.ServerMessage{
		Type:           chatbot.MessageTypeDone,
		ConversationID: conv.ID,
	})
}

func (h *TestHandler) buildMessages(conv *chatbot.Conversation, userMessage string) []chatbot.Message {
	systemPrompt := chatbot.SystemPrompt()
	messages := []chatbot.Message{
		{Role: "system", Content: &systemPrompt},
	}
	messages = append(messages, conv.Messages...)
	messages = append(messages, chatbot.Message{Role: "user", Content: &userMessage})
	return messages
}

func (h *TestHandler) sendError(conn *websocket.Conn, msg string) {
	conn.WriteJSON(chatbot.ServerMessage{
		Type:  chatbot.MessageTypeError,
		Error: msg,
	})
}

// Integration tests

func TestWebSocketConnection(t *testing.T) {
	endpoint := os.Getenv("AI_ENDPOINT")
	model := os.Getenv("AI_MODEL")
	if endpoint == "" || model == "" {
		t.Skip("AI_ENDPOINT or AI_MODEL not set; skipping integration test")
	}
	handler := NewTestHandler(endpoint, model)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send a simple message
	err = conn.WriteJSON(chatbot.ClientMessage{Message: "Hello"})
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Read responses with timeout
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	var textReceived bool
	var doneReceived bool
	var conversationID string

	for {
		var msg chatbot.ServerMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		t.Logf("Received message type=%s content=%q error=%q", msg.Type, msg.Content, msg.Error)

		switch msg.Type {
		case chatbot.MessageTypeText:
			textReceived = true
		case chatbot.MessageTypeDone:
			doneReceived = true
			conversationID = msg.ConversationID
		case chatbot.MessageTypeError:
			t.Fatalf("Received error: %s", msg.Error)
		}

		if doneReceived {
			break
		}
	}

	if !textReceived {
		t.Error("Did not receive any text messages")
	}
	if !doneReceived {
		t.Error("Did not receive done message")
	}
	if conversationID == "" {
		t.Error("Conversation ID is empty")
	}

	t.Logf("Test passed! Conversation ID: %s", conversationID)
}

func TestToolCallExecution(t *testing.T) {
	endpoint := os.Getenv("AI_ENDPOINT")
	model := os.Getenv("AI_MODEL")
	if endpoint == "" || model == "" {
		t.Skip("AI_ENDPOINT or AI_MODEL not set; skipping integration test")
	}
	handler := NewTestHandler(endpoint, model)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Ask about devices - should trigger tool call
	err = conn.WriteJSON(chatbot.ClientMessage{Message: "How many devices are available?"})
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(90 * time.Second))

	var fullResponse string
	var doneReceived bool

	for {
		var msg chatbot.ServerMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}

		t.Logf("Received: type=%s content=%q", msg.Type, truncate(msg.Content, 100))

		switch msg.Type {
		case chatbot.MessageTypeText:
			fullResponse += msg.Content
		case chatbot.MessageTypeDone:
			doneReceived = true
		case chatbot.MessageTypeError:
			t.Fatalf("Received error: %s", msg.Error)
		}

		if doneReceived {
			break
		}
	}

	t.Logf("Full response: %s", fullResponse)

	// The response should mention something about devices
	if !strings.Contains(strings.ToLower(fullResponse), "device") &&
		!strings.Contains(strings.ToLower(fullResponse), "available") {
		t.Logf("Warning: Response may not contain expected content about devices")
	}
}

func TestConversationContinuity(t *testing.T) {
	endpoint := os.Getenv("AI_ENDPOINT")
	model := os.Getenv("AI_MODEL")
	if endpoint == "" || model == "" {
		t.Skip("AI_ENDPOINT or AI_MODEL not set; skipping integration test")
	}
	handler := NewTestHandler(endpoint, model)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// First message
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	conn1.WriteJSON(chatbot.ClientMessage{Message: "What locations are available?"})
	conn1.SetReadDeadline(time.Now().Add(60 * time.Second))

	var conversationID string
	for {
		var msg chatbot.ServerMessage
		if err := conn1.ReadJSON(&msg); err != nil {
			t.Fatalf("Failed to read: %v", err)
		}
		if msg.Type == chatbot.MessageTypeError {
			t.Fatalf("Error: %s", msg.Error)
		}
		if msg.Type == chatbot.MessageTypeDone {
			conversationID = msg.ConversationID
			break
		}
	}
	conn1.Close()

	t.Logf("First conversation ID: %s", conversationID)

	// Continue conversation
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL+"?conversation_id="+conversationID, nil)
	if err != nil {
		t.Fatalf("Failed to connect for continuation: %v", err)
	}
	defer conn2.Close()

	conn2.WriteJSON(chatbot.ClientMessage{Message: "Which one has the most devices?"})
	conn2.SetReadDeadline(time.Now().Add(60 * time.Second))

	var secondConvID string
	for {
		var msg chatbot.ServerMessage
		if err := conn2.ReadJSON(&msg); err != nil {
			t.Fatalf("Failed to read: %v", err)
		}
		if msg.Type == chatbot.MessageTypeError {
			t.Fatalf("Error: %s", msg.Error)
		}
		if msg.Type == chatbot.MessageTypeDone {
			secondConvID = msg.ConversationID
			break
		}
	}

	if secondConvID != conversationID {
		t.Errorf("Conversation ID changed: %s -> %s", conversationID, secondConvID)
	}

	t.Logf("Conversation continuity test passed!")
}

func TestStatsQuery(t *testing.T) {
	endpoint := os.Getenv("AI_ENDPOINT")
	model := os.Getenv("AI_MODEL")
	if endpoint == "" || model == "" {
		t.Skip("AI_ENDPOINT or AI_MODEL not set; skipping integration test")
	}
	handler := NewTestHandler(endpoint, model)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Ask for stats - should trigger get_stats tool
	err = conn.WriteJSON(chatbot.ClientMessage{Message: "Give me inventory statistics"})
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(90 * time.Second))

	var fullResponse string
	for {
		var msg chatbot.ServerMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("Failed to read: %v", err)
		}

		t.Logf("Received: type=%s", msg.Type)

		switch msg.Type {
		case chatbot.MessageTypeText:
			fullResponse += msg.Content
		case chatbot.MessageTypeDone:
			t.Logf("Stats response: %s", truncate(fullResponse, 500))
			return
		case chatbot.MessageTypeError:
			t.Fatalf("Error: %s", msg.Error)
		}
	}
}

func TestEmptyMessage(t *testing.T) {
	endpoint := os.Getenv("AI_ENDPOINT")
	model := os.Getenv("AI_MODEL")
	if endpoint == "" || model == "" {
		t.Skip("AI_ENDPOINT or AI_MODEL not set; skipping integration test")
	}
	handler := NewTestHandler(endpoint, model)

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send empty message
	err = conn.WriteJSON(chatbot.ClientMessage{Message: ""})
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	var msg chatbot.ServerMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if msg.Type != chatbot.MessageTypeError {
		t.Errorf("Expected error message, got type=%s", msg.Type)
	}

	if !strings.Contains(msg.Error, "empty") {
		t.Errorf("Expected error about empty message, got: %s", msg.Error)
	}

	t.Logf("Empty message test passed: %s", msg.Error)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
