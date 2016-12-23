package httpapi

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
)

//NewRouter returns an HTTP router for the HTTP API
func NewRouter(ctx context.Context, s SessionStore, db *sql.DB) http.Handler {
	r := mux.NewRouter()

	//catch-all
	//	r.PathPrefix("/").HandlerFunc(notFoundHandler)

	r.Path("/models/").Methods("POST").HandlerFunc(handleCreateModel)
	r.Path("/models/{id:[0-9]+}").Methods("GET").HandlerFunc(handleReadModel)
	r.Path("/models/{id:[0-9]+}").Methods("POST").HandlerFunc(handleUpdateModel)
	r.Path("/models/{id:[0-9]+}/notes").Methods("POST").HandlerFunc(handleCreateModelNoteEvent)

	r.Path("/devices/").Methods("POST").HandlerFunc(handleCreateDevice)
	r.Path("/devices/{id:[0-9]+}").Methods("GET").HandlerFunc(handleReadDevice)
	r.Path("/devices/{id:[0-9]+}").Methods("POST").HandlerFunc(handleUpdateDevice)
	r.Path("/devices/{id:[0-9]+}/notes").Methods("POST").HandlerFunc(handleCreateDeviceNoteEvent)

	r.Path("/users/").Methods("POST").HandlerFunc(handleCreateUserWithCredentials)
	r.Path("/users/{id:[0-9]+}").Methods("GET").HandlerFunc(handleReadUser)
	r.Path("/users/{id:[0-9]+}").Methods("POST").HandlerFunc(handleUpdateUser)
	r.Path("/users/{id:[0-9]+}/password").Methods("POST").HandlerFunc(handleChangeUserPassword)
	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	auth := mux.NewRouter()
	auth.Path("/auth").Methods("POST").HandlerFunc(handleAuthenticate(s))

	mux := http.NewServeMux()

	mux.Handle("/auth", auth)
	mux.Handle("/", authMiddleware(r, s))

	return http.StripPrefix("/api/1.0", jsonMiddleware(txMiddleware(mux, db)))
}
