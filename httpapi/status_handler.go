package httpapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/korylprince/tcea-inventory-server/api"
)

//GET /statuses/
func handleReadStatuses(w http.ResponseWriter, r *http.Request) {
	statuses, err := api.ReadStatuses(r.Context())
	if !checkAPIError(w, r, err) {
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(&ReadStatusesResponse{Statuses: statuses})
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}
