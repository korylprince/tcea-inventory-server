package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/korylprince/tcea-inventory-server/api"
)

//POST /models
func handleCreateModel(w http.ResponseWriter, r *http.Request) *handlerResponse {
	var req *CreateModelRequest
	d := json.NewDecoder(r.Body)

	err := d.Decode(&req)
	if err != nil || req == nil || req.Model == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
	}

	id, err := api.CreateModel(r.Context(), req.Model)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	if req.Note != "" {
		_, err = api.CreateNoteEvent(r.Context(), id, api.ModelEventLocation, req.Note)
		if resp := checkAPIError(err); resp != nil {
			return resp
		}
	}

	model, err := api.ReadModel(r.Context(), id, true)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if model == nil {
		return handleError(http.StatusInternalServerError, errors.New("Could not find model, but just created"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: model}
}

//GET /models/:id
func handleReadModel(w http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	includeEvents := false
	if v := r.URL.Query().Get("events"); v == eventsTrue {
		includeEvents = true
	}

	model, err := api.ReadModel(r.Context(), id, includeEvents)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if model == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find model"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: model}
}

//POST /models/:id
func handleUpdateModel(w http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	var model *api.Model
	d := json.NewDecoder(r.Body)

	err = d.Decode(&model)
	if err != nil || model == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
	}

	if model.ID != id {
		return handleError(http.StatusBadRequest, fmt.Errorf("model id mismatch: URL: %d, Body: %d", id, model.ID))
	}

	err = api.UpdateModel(r.Context(), model)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	model, err = api.ReadModel(r.Context(), model.ID, true)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if model == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find model, but just updated"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: model}
}

//POST /models/:id/notes
func handleCreateModelNoteEvent(w http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	var note *NoteRequest
	d := json.NewDecoder(r.Body)

	err = d.Decode(&note)
	if err != nil || note == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
	}

	_, err = api.CreateNoteEvent(r.Context(), id, api.ModelEventLocation, note.Note)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	model, err := api.ReadModel(r.Context(), id, true)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if model == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find model, but just updated"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: model}

}

//GET /models/
func handleQueryModel(w http.ResponseWriter, r *http.Request) *handlerResponse {
	models, err := api.QueryModel(r.Context(),
		r.URL.Query().Get("manufacturer"),
		r.URL.Query().Get("model"),
	)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	return &handlerResponse{Code: http.StatusOK, Body: &QueryModelResponse{Models: models}}
}
