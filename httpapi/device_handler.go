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

//POST /devices
func handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var req *DeviceCreateRequest
	d := json.NewDecoder(r.Body)

	err := d.Decode(&req)
	if err != nil || req == nil || req.Device == nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
		return
	}

	id, err := api.CreateDevice(r.Context(), req.Device)
	if !checkAPIError(w, r, err) {
		return
	}

	if req.Note != "" {
		_, err = api.CreateNoteEvent(r.Context(), id, api.DeviceEventLocation, req.Note)
		if !checkAPIError(w, r, err) {
			return
		}
	}

	device, err := api.ReadDevice(r.Context(), id)
	if !checkAPIError(w, r, err) {
		return
	}
	if device == nil {
		handleError(w, r, http.StatusInternalServerError, errors.New("Could not find device, but just created"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(device)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//GET /devices/:id
func handleReadDevice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
		return
	}

	device, err := api.ReadDevice(r.Context(), id)
	if !checkAPIError(w, r, err) {
		return
	}
	if device == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find device"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(device)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//POST /devices/:id
func handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
		return
	}

	var device *api.Device
	d := json.NewDecoder(r.Body)

	err = d.Decode(&device)
	if err != nil || device == nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode JSON: %v", err))
		return
	}

	if device.ID != id {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("device id mismatch: URL: %d, Body: %d", id, device.ID))
		return
	}

	err = api.UpdateDevice(r.Context(), device)
	if !checkAPIError(w, r, err) {
		return
	}

	device, err = api.ReadDevice(r.Context(), device.ID)
	if !checkAPIError(w, r, err) {
		return
	}
	if device == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find device, but just updated"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(device)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//POST /devices/:id/notes
func handleCreateDeviceNoteEvent(w http.ResponseWriter, r *http.Request) {
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

	_, err = api.CreateNoteEvent(r.Context(), id, api.DeviceEventLocation, note.Note)
	if !checkAPIError(w, r, err) {
		return
	}

	device, err := api.ReadDevice(r.Context(), id)
	if !checkAPIError(w, r, err) {
		return
	}
	if device == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find device, but just updated"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(device)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}
