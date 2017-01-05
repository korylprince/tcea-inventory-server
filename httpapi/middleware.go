package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/korylprince/tcea-inventory-server/api"
)

type handlerResponse struct {
	Code int
	Body interface{}
	User *api.User
	Err  error
}

type returnHandler func(http.ResponseWriter, *http.Request) *handlerResponse

const logTemplate = "{{.Date}} {{.Method}} {{.Path}}{{if .Query}}?{{.Query}}{{end}} {{.Code}} ({{.Status}}) {{if .User}}, User: {{.User.ID}}:{{.User.Email}}{{end}}{{if .Err}}, Error: {{.Err}}{{end}}\n"

type logData struct {
	Date   string
	User   *api.User
	Status string
	Code   int
	Method string
	Path   string
	Query  string
	Err    error
}

func logMiddleware(next returnHandler, writer io.Writer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := next(w, r)

		err := template.Must(template.New("log").Parse(logTemplate)).Execute(writer, &logData{
			Date:   time.Now().Format("2006-01-02:15:04:05 -0700"),
			User:   resp.User,
			Status: http.StatusText(resp.Code),
			Code:   resp.Code,
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.RawQuery,
			Err:    resp.Err,
		})

		if err != nil {
			panic(err)
		}
	})
}

func jsonMiddleware(next returnHandler) returnHandler {
	return func(w http.ResponseWriter, r *http.Request) *handlerResponse {
		var resp *handlerResponse

		if r.Method != "GET" {
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				resp = handleError(http.StatusBadRequest, errors.New("Could not parse Content-Type"))
				goto serve
			}
			if mediaType != "application/json" {
				resp = handleError(http.StatusBadRequest, errors.New("Content-Type not application/json"))
				goto serve
			}
		}

		w.Header().Set("Content-Type", "application/json")
		resp = next(w, r)

	serve:
		w.WriteHeader(resp.Code)
		e := json.NewEncoder(w)
		err := e.Encode(resp.Body)
		if err != nil {
			return handleError(http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
		}
		return resp
	}
}

func authMiddleware(next returnHandler, s SessionStore) returnHandler {
	return func(w http.ResponseWriter, r *http.Request) *handlerResponse {
		key := r.Header.Get("X-Session-Key")
		if key == "" {
			return handleError(http.StatusUnauthorized, errors.New("X-Session-Key header empty"))
		}

		sess, err := s.Check(key)
		if err != nil {
			return handleError(http.StatusInternalServerError, fmt.Errorf("Could not check session key: %v", err))
		}
		if sess == nil {
			return handleError(http.StatusUnauthorized, errors.New("Could not find session"))
		}

		user, err := api.ReadUser(r.Context(), sess.UserID)
		if resp := checkAPIError(err); resp != nil {
			return resp
		}

		ctx := context.WithValue(r.Context(), api.UserKey, user)
		resp := next(w, r.WithContext(ctx))
		resp.User = user

		return resp
	}
}

func txMiddleware(next returnHandler, db *sql.DB) returnHandler {
	return func(w http.ResponseWriter, r *http.Request) *handlerResponse {
		tx, err := db.Begin()
		if err != nil {
			return handleError(http.StatusInternalServerError, fmt.Errorf("Could not begin transaction: %v", err))
		}

		ctx := context.WithValue(r.Context(), api.TransactionKey, tx)
		resp := next(w, r.WithContext(ctx))

		if err = tx.Commit(); err != nil {
			if rErr := tx.Rollback(); rErr != nil && rErr != sql.ErrTxDone {
				return handleError(http.StatusInternalServerError, fmt.Errorf("Could not rollback transaction: %v", rErr))
			}
			return handleError(http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		}

		return resp
	}
}
