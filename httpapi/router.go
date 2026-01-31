package httpapi

import (
	"database/sql"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/korylprince/tcea-inventory-server/chatbot"
)

// ChatConfig holds configuration for the chat handler
type ChatConfig struct {
	AIEndpoint    string
	AIModel       string
	CacheMaxBytes int
}

//NewRouter returns an HTTP router for the HTTP API
func NewRouter(w io.Writer, s SessionStore, db *sql.DB, chatCfg *ChatConfig) http.Handler {

	//construct middleware
	var m = func(h returnHandler) http.Handler {
		return logMiddleware(jsonMiddleware(txMiddleware(authMiddleware(h, s), db)), w)
	}

	r := mux.NewRouter()

	r.Path("/statuses/").Methods("GET").Handler(m(handleReadStatuses))
	r.Path("/locations/").Methods("GET").Handler(m(handleReadLocations))

	r.Path("/models/").Methods("POST").Handler(m(handleCreateModel))
	r.Path("/models/").Methods("GET").Handler(m(handleQueryModel))
	r.Path("/models/{id:[0-9]+}").Methods("GET").Handler(m(handleReadModel))
	r.Path("/models/{id:[0-9]+}").Methods("POST").Handler(m(handleUpdateModel))

	r.Path("/devices/").Methods("POST").Handler(m(handleCreateDevice))
	r.Path("/devices/").Methods("GET").Handler(m(handleQueryDevice))
	r.Path("/devices/{id:[0-9]+}").Methods("GET").Handler(m(handleReadDevice))
	r.Path("/devices/{id:[0-9]+}").Methods("POST").Handler(m(handleUpdateDevice))
	r.Path("/devices/{id:[0-9]+}/notes/").Methods("POST").Handler(m(handleCreateDeviceNoteEvent))

	r.Path("/users/").Methods("POST").Handler(m(handleCreateUserWithCredentials))
	r.Path("/users/{id:[0-9]+}").Methods("GET").Handler(m(handleReadUser))
	r.Path("/users/{id:[0-9]+}").Methods("POST").Handler(m(handleUpdateUser))
	r.Path("/users/{id:[0-9]+}/password").Methods("POST").Handler(m(handleChangeUserPassword))

	r.Path("/stats/").Methods("GET").Handler(m(handleReadStats))

	r.Path("/auth").Methods("POST").Handler(logMiddleware(jsonMiddleware(txMiddleware(handleAuthenticate(s), db)), w))

	// Chat WebSocket endpoint (auth via header, no JSON middleware)
	if chatCfg != nil {
		store := chatbot.NewLRUStore(chatCfg.CacheMaxBytes)
		client := chatbot.NewAIClient(chatCfg.AIEndpoint, chatCfg.AIModel)
		chatHandler := chatbot.NewHandler(store, client, db)
		r.Path("/chat").Handler(wsAuthMiddleware(chatHandler, s, db, w))
	}

	r.NotFoundHandler = m(notFoundHandler)

	return http.StripPrefix("/api/1.0", r)
}
