package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/korylprince/tcea-inventory-server/api"
)

func authMiddleware(next http.Handler, s SessionStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Session-Key")
		if key == "" {
			handleError(w, r, http.StatusForbidden, errors.New("X-Session-Key header empty"))
			return
		}

		sess, err := s.Check(key)
		if err != nil {
			handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not check session key: %v", err))
			return
		}
		if sess == nil {
			handleError(w, r, http.StatusForbidden, errors.New("Could not find session"))
			return
		}

		user, err := api.ReadUser(r.Context(), sess.UserID)
		if !checkAPIError(w, r, err) {
			return
		}

		ctx := context.WithValue(r.Context(), api.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Header.Get("Content-Type") != "application/json" {
			handleError(w, r, http.StatusBadRequest, errors.New("Content-Type not application/json"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func txMiddleware(next http.Handler, db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx, err := db.Begin()
		if err != nil {
			handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not begin transaction: %v", err))
			return
		}
		ctx := context.WithValue(r.Context(), api.TransactionKey, tx)
		next.ServeHTTP(w, r.WithContext(ctx))

		err = tx.Rollback()
		if err != sql.ErrTxDone {
			log.Println("Transaction rolled back")
		}

	})
}
