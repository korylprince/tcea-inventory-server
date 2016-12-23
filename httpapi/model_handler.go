package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/korylprince/tcea-inventory-server/api"
)

//POST /models
func handleCreateModel(w http.ResponseWriter, r *http.Request) {
	var req *ModelCreateRequest
	d := json.NewDecoder(r.Body)

	err := d.Decode(&req)
	if err != nil || req == nil || req.Model == nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
		return
	}

	id, err := api.CreateModel(r.Context(), req.Model)
	if !checkAPIError(w, r, err) {
		return
	}

	if req.Note != "" {
		_, err = api.CreateNoteEvent(r.Context(), id, api.ModelEventLocation, req.Note)
		if !checkAPIError(w, r, err) {
			return
		}
	}

	model, err := api.ReadModel(r.Context(), id)
	if !checkAPIError(w, r, err) {
		return
	}
	if model == nil {
		handleError(w, r, http.StatusInternalServerError, errors.New("Could not find model, but just created"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(model)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//GET /models/:id
func handleReadModel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
		return
	}

	model, err := api.ReadModel(r.Context(), id)
	if !checkAPIError(w, r, err) {
		return
	}
	if model == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find model"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(model)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//POST /models/:id
func handleUpdateModel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
		return
	}

	var model *api.Model
	d := json.NewDecoder(r.Body)

	err = d.Decode(&model)
	if err != nil || model == nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
		return
	}

	if model.ID != id {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("model id mismatch: URL: %d, Body: %d", id, model.ID))
		return
	}

	err = api.UpdateModel(r.Context(), model)
	if !checkAPIError(w, r, err) {
		return
	}

	model, err = api.ReadModel(r.Context(), model.ID)
	if !checkAPIError(w, r, err) {
		return
	}
	if model == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find model, but just updated"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(model)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//POST /models/:id/notes
func handleCreateModelNoteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
		return
	}

	var note *NoteRequest
	d := json.NewDecoder(r.Body)

	err = d.Decode(&note)
	if err != nil || note == nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
		return
	}

	_, err = api.CreateNoteEvent(r.Context(), id, api.ModelEventLocation, note.Note)
	if !checkAPIError(w, r, err) {
		return
	}

	model, err := api.ReadModel(r.Context(), id)
	if !checkAPIError(w, r, err) {
		return
	}
	if model == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find model, but just updated"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(model)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}
