package chatbot

import (
	"container/list"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"sync"
	"time"
)

var randChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
var randMax = big.NewInt(int64(len(randChars)))

func randKey(length int) string {
	str := make([]byte, length)
	for i := range str {
		k, err := rand.Int(rand.Reader, randMax)
		if err != nil {
			str[i] = randChars[0]
		} else {
			str[i] = randChars[k.Int64()]
		}
	}
	return string(str)
}

// Message represents a chat message in OpenAI format
type Message struct {
	Role       string     `json:"role"`                   // system, user, assistant, tool
	Content    *string    `json:"content"`                // text content (nil for tool_calls-only messages)
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // for assistant tool call requests
	ToolCallID string     `json:"tool_call_id,omitempty"` // for tool response messages
	Name       string     `json:"name,omitempty"`         // tool name (in tool responses)
}

// MarshalJSON customizes JSON marshaling to send null for empty content strings
// This is required because the API treats empty string content as "prefill" which
// is incompatible with thinking mode
func (m Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	aux := struct {
		Alias
		Content *string `json:"content"`
	}{
		Alias: Alias(m),
	}
	// Convert empty string to nil
	if m.Content != nil && *m.Content != "" {
		aux.Content = m.Content
	}
	return json.Marshal(aux)
}

// ToolCall represents a tool call request from the assistant
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // always "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and arguments
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Conversation represents a chat conversation
type Conversation struct {
	ID        string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConversationStore defines the interface for conversation storage
type ConversationStore interface {
	Get(id string) (*Conversation, error)
	Create() (*Conversation, error)
	AddMessages(id string, msgs []Message) error
}

// LRUStore implements ConversationStore with an LRU cache
type LRUStore struct {
	mu       sync.Mutex
	maxBytes int
	curBytes int
	cache    map[string]*list.Element
	lru      *list.List
}

type cacheEntry struct {
	id    string
	conv  *Conversation
	bytes int
}

// NewLRUStore creates a new LRU conversation store
func NewLRUStore(maxBytes int) *LRUStore {
	return &LRUStore{
		maxBytes: maxBytes,
		cache:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

func (s *LRUStore) estimateBytes(conv *Conversation) int {
	data, _ := json.Marshal(conv)
	return len(data)
}

// Get retrieves a conversation by ID
func (s *LRUStore) Get(id string) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elem, ok := s.cache[id]; ok {
		s.lru.MoveToFront(elem)
		return elem.Value.(*cacheEntry).conv, nil
	}
	return nil, nil
}

// Create creates a new conversation
func (s *LRUStore) Create() (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	conv := &Conversation{
		ID:        randKey(32),
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	bytes := s.estimateBytes(conv)
	s.evictIfNeeded(bytes)

	entry := &cacheEntry{id: conv.ID, conv: conv, bytes: bytes}
	elem := s.lru.PushFront(entry)
	s.cache[conv.ID] = elem
	s.curBytes += bytes

	return conv, nil
}

// AddMessages adds messages to a conversation
func (s *LRUStore) AddMessages(id string, msgs []Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	elem, ok := s.cache[id]
	if !ok {
		return nil
	}

	entry := elem.Value.(*cacheEntry)
	oldBytes := entry.bytes

	entry.conv.Messages = append(entry.conv.Messages, msgs...)
	entry.conv.UpdatedAt = time.Now()

	newBytes := s.estimateBytes(entry.conv)
	entry.bytes = newBytes
	s.curBytes += (newBytes - oldBytes)

	s.lru.MoveToFront(elem)
	s.evictIfNeeded(0)

	return nil
}

func (s *LRUStore) evictIfNeeded(additionalBytes int) {
	for s.curBytes+additionalBytes > s.maxBytes && s.lru.Len() > 0 {
		oldest := s.lru.Back()
		if oldest == nil {
			break
		}
		entry := oldest.Value.(*cacheEntry)
		s.lru.Remove(oldest)
		delete(s.cache, entry.id)
		s.curBytes -= entry.bytes
	}
}
