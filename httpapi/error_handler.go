package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/korylprince/tcea-inventory-server/api"
)

//ErrorResponse represents an HTTP error
type ErrorResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

//DuplicateErrorResponse represents an HTTP error
type DuplicateErrorResponse struct {
	Code        int    `json:"code"`
	Error       string `json:"error"`
	DuplicateID int64  `json:"duplicate_id"`
}

//handleError returns a json response for the given code and logs the error
func handleError(w http.ResponseWriter, r *http.Request, code int, err error) {
	log.Printf("Error at path %s: %v\n", r.URL.String(), err)

	w.WriteHeader(code)

	e := json.NewEncoder(w)

	encErr := e.Encode(ErrorResponse{Code: code, Error: http.StatusText(code)})
	if encErr != nil {
		panic(encErr)
	}
}

//handleDuplicateError returns a json response for the given code and logs the error
func handleDuplicateError(w http.ResponseWriter, r *http.Request, err *api.Error) {
	log.Printf("Error at path %s: %v\n", r.URL.String(), err)

	w.WriteHeader(http.StatusConflict)

	e := json.NewEncoder(w)

	encErr := e.Encode(DuplicateErrorResponse{
		Code:        http.StatusConflict,
		Error:       http.StatusText(http.StatusConflict),
		DuplicateID: err.DuplicateID,
	})
	if encErr != nil {
		panic(encErr)
	}
}

//notFoundHandler returns a json 401 response
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	handleError(w, r, http.StatusNotFound, errors.New("Could not find handler"))
}

//checkAPIError checks api.Error's and writes JSON responses for them, returning true if there is no error
func checkAPIError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return true
	}
	e := err.(*api.Error)
	if e != nil {
		if e.Type == api.ErrorTypeServer {
			handleError(w, r, http.StatusInternalServerError, err)
		} else if e.Type == api.ErrorTypeUser {
			handleError(w, r, http.StatusBadRequest, err)
		} else {
			handleDuplicateError(w, r, err.(*api.Error))
		}
		return false
	}
	return true
}
