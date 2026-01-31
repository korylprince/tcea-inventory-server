package chatbot

// ClientMessage is the message format from client to server
type ClientMessage struct {
	Message string `json:"message"`
}

// ServerMessage is the message format from server to client
type ServerMessage struct {
	Type           string `json:"type"`                      // "text", "done", or "error"
	Content        string `json:"content,omitempty"`         // partial response text
	ConversationID string `json:"conversation_id,omitempty"` // sent with "done"
	Error          string `json:"error,omitempty"`           // sent with "error"
}

// Message types
const (
	MessageTypeText  = "text"
	MessageTypeDone  = "done"
	MessageTypeError = "error"
)
