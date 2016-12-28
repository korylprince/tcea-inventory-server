package httpapi

import (
	"sync"
	"time"
)

//SessionStore is an interface to an arbitrary session backend.
type SessionStore interface {
	//Create returns a new sessionID with the given User id. If the backend malfunctions,
	//sessionID will be an empty string and err will be non-nil.
	Create(userID int64) (sessionID string, err error)

	//Check returns whether or not sessionID is a valid session.
	//If sessionID is not valid, session will be nil.
	//If the backend malfunctions, session will be nil and err will be non-nil.
	Check(sessionID string) (session *Session, err error)
}

//Session represents a login session
type Session struct {
	UserID  int64
	Expires time.Time
}

//MemorySessionStore represents a SessionStore that uses an in-memory map
type MemorySessionStore struct {
	store    map[string]*Session
	duration time.Duration
	mu       *sync.Mutex
}

//scavenge removes stale records every hour
func scavenge(m *MemorySessionStore) {
	for {
		time.Sleep(time.Hour)
		now := time.Now()
		m.mu.Lock()
		for id, t := range m.store {
			if t.Expires.Before(now) {
				delete(m.store, id)
			}
		}
		m.mu.Unlock()
	}
}

//NewMemorySessionStore returns a new MemorySessionStore with the given expiration duration.
func NewMemorySessionStore(duration time.Duration) *MemorySessionStore {
	m := &MemorySessionStore{
		store:    make(map[string]*Session),
		duration: duration,
		mu:       new(sync.Mutex),
	}
	go scavenge(m)
	return m
}

//Create returns a new sessionID with the given User id. err will always be nil.
func (m *MemorySessionStore) Create(userID int64) (sessionID string, err error) {
	id := randString(128)
	m.mu.Lock()
	m.store[id] = &Session{
		UserID:  userID,
		Expires: time.Now().Add(m.duration),
	}
	m.mu.Unlock()
	return id, nil
}

//Check returns whether or not sessionID is a valid session. If sessionID is not valid, session will be nil.
//err will always be nil.
func (m *MemorySessionStore) Check(sessionID string) (session *Session, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.store[sessionID]; ok {
		if s.Expires.After(time.Now()) {
			s.Expires = time.Now().Add(m.duration)
			return s, nil
		}
		delete(m.store, sessionID)
	}
	return nil, nil
}
