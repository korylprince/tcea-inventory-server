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

// POST /devices
func handleCreateDevice(_ http.ResponseWriter, r *http.Request) *handlerResponse {
	var req *CreateDeviceRequest
	d := json.NewDecoder(r.Body)

	err := d.Decode(&req)
	if err != nil || req == nil || req.Device == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
	}

	id, err := api.CreateDevice(r.Context(), req.Device)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	if req.Note != "" {
		_, err = api.CreateNoteEvent(r.Context(), id, api.DeviceEventLocation, req.Note)
		if resp := checkAPIError(err); resp != nil {
			return resp
		}
	}

	device, err := api.ReadDevice(r.Context(), id, true)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if device == nil {
		return handleError(http.StatusInternalServerError, errors.New("Could not find device, but just created"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: device}
}

// GET /devices/:id
func handleReadDevice(_ http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	includeEvents := false
	if v := r.URL.Query().Get("events"); v == eventsTrue {
		includeEvents = true
	}

	device, err := api.ReadDevice(r.Context(), id, includeEvents)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if device == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find device"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: device}
}

// POST /devices/:id
func handleUpdateDevice(_ http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	var device *api.Device
	d := json.NewDecoder(r.Body)

	err = d.Decode(&device)
	if err != nil || device == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
	}

	if device.ID != id {
		return handleError(http.StatusBadRequest, fmt.Errorf("device id mismatch: URL: %d, Body: %d", id, device.ID))
	}

	err = api.UpdateDevice(r.Context(), device)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	device, err = api.ReadDevice(r.Context(), device.ID, true)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if device == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find device, but just updated"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: device}
}

// POST /devices/:id/notes/
func handleCreateDeviceNoteEvent(_ http.ResponseWriter, r *http.Request) *handlerResponse {
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

	_, err = api.CreateNoteEvent(r.Context(), id, api.DeviceEventLocation, note.Note)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	device, err := api.ReadDevice(r.Context(), id, true)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if device == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find device, but just updated"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: device}
}

// GET /devices/
func handleQueryDevice(w http.ResponseWriter, r *http.Request) *handlerResponse {
	if r.URL.Query().Get("search") != "" {
		return handleSimpleQueryDevice(w, r)
	}

	devices, err := api.QueryDevice(r.Context(),
		r.URL.Query().Get("serial_number"),
		r.URL.Query().Get("manufacturer"),
		r.URL.Query().Get("model"),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("location"),
	)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	return &handlerResponse{Code: http.StatusOK, Body: &QueryDeviceResponse{Devices: devices}}
}

// GET /devices/
func handleSimpleQueryDevice(_ http.ResponseWriter, r *http.Request) *handlerResponse {
	devices, err := api.SimpleQueryDevice(r.Context(), r.URL.Query().Get("search"))
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	return &handlerResponse{Code: http.StatusOK, Body: &QueryDeviceResponse{Devices: devices}}
}
