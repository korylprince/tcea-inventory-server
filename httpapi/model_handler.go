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

// POST /models
func handleCreateModel(_ http.ResponseWriter, r *http.Request) *handlerResponse {
	var model *api.Model
	d := json.NewDecoder(r.Body)

	err := d.Decode(&model)
	if model == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
	}

	id, err := api.CreateModel(r.Context(), model)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	model, err = api.ReadModel(r.Context(), id)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if model == nil {
		return handleError(http.StatusInternalServerError, errors.New("Could not find model, but just created"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: model}
}

// GET /models/:id
func handleReadModel(_ http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	model, err := api.ReadModel(r.Context(), id)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if model == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find model"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: model}
}

// POST /models/:id
func handleUpdateModel(_ http.ResponseWriter, r *http.Request) *handlerResponse {
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

	model, err = api.ReadModel(r.Context(), model.ID)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if model == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find model, but just updated"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: model}
}

// GET /models/
func handleQueryModel(_ http.ResponseWriter, r *http.Request) *handlerResponse {
	models, err := api.QueryModel(r.Context(),
		r.URL.Query().Get("manufacturer"),
		r.URL.Query().Get("model"),
	)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	return &handlerResponse{Code: http.StatusOK, Body: &QueryModelResponse{Models: models}}
}
