package httpapi

import (
	"net/http"

	"github.com/korylprince/tcea-inventory-server/api"
)

//GET /locations/
func handleReadLocations(w http.ResponseWriter, r *http.Request) *handlerResponse {
	locations, err := api.ReadLocations(r.Context())
	if err := checkAPIError(err); err != nil {
		return err
	}

	return &handlerResponse{Code: http.StatusOK, Body: &ReadLocationsResponse{Locations: locations}}
}
