package httpapi

import (
	"net/http"

	"github.com/korylprince/tcea-inventory-server/api"
)

//GET /statuses/
func handleReadStatuses(w http.ResponseWriter, r *http.Request) *handlerResponse {
	statuses, err := api.ReadStatuses(r.Context())
	if err := checkAPIError(err); err != nil {
		return err
	}

	return &handlerResponse{Code: http.StatusOK, Body: &ReadStatusesResponse{Statuses: statuses}}
}
