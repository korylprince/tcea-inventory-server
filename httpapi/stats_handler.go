package httpapi

import (
	"net/http"

	"github.com/korylprince/tcea-inventory-server/api"
)

//GET /stats/
func handleReadStats(w http.ResponseWriter, r *http.Request) *handlerResponse {
	stats, err := api.ReadStats(r.Context())
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	return &handlerResponse{Code: http.StatusOK, Body: stats}
}
