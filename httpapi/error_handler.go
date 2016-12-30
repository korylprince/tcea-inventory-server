package httpapi

import (
	"errors"
	"net/http"

	"github.com/korylprince/tcea-inventory-server/api"
)

//ErrorResponse represents an HTTP error. If the error is 409 Conflict, the DuplicateID field will be populated.
type ErrorResponse struct {
	Code        int    `json:"code"`
	Error       string `json:"error"`
	DuplicateID int64  `json:"duplicate_id,omitempty"`
}

//handleError returns a handlerResponse response for the given code
func handleError(code int, err error) *handlerResponse {
	return &handlerResponse{Code: code, Body: &ErrorResponse{Code: code, Error: http.StatusText(code)}, Err: err}
}

//notFoundHandler returns a 401 handlerResponse
func notFoundHandler(w http.ResponseWriter, r *http.Request) *handlerResponse {
	return handleError(http.StatusNotFound, errors.New("Could not find handler"))
}

//checkAPIError checks an api.Error and returns a handlerResponse for it, or nil if there was no error
func checkAPIError(err error) *handlerResponse {
	if err == nil {
		return nil
	}

	e := err.(*api.Error)
	if e.Type == api.ErrorTypeServer {
		return handleError(http.StatusInternalServerError, err)
	} else if e.Type == api.ErrorTypeUser {
		return handleError(http.StatusBadRequest, err)
	} else {
		return &handlerResponse{Code: http.StatusConflict, Body: &ErrorResponse{
			Code:        http.StatusConflict,
			Error:       http.StatusText(http.StatusConflict),
			DuplicateID: e.DuplicateID,
		}, Err: err}
	}
}
