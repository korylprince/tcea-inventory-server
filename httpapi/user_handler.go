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

//POST /users
func handleCreateUserWithCredentials(w http.ResponseWriter, r *http.Request) {
	var req *UserCreateRequest
	d := json.NewDecoder(r.Body)

	err := d.Decode(&req)
	if err != nil || req == nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode json: %v", err))
		return
	}

	id, err := api.CreateUserWithCredentials(r.Context(), req.Email, req.Password, req.Name)
	if !checkAPIError(w, r, err) {
		return
	}

	user, err := api.ReadUser(r.Context(), id)
	if !checkAPIError(w, r, err) {
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	if user == nil {
		handleError(w, r, http.StatusInternalServerError, errors.New("Could not find user, but just created"))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(user)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//GET /users/:id
func handleReadUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
		return
	}

	user, err := api.ReadUser(r.Context(), id)
	if !checkAPIError(w, r, err) {
		return
	}
	if user == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find user"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(user)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//POST /users/:id
func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
		return
	}

	var user *api.User
	d := json.NewDecoder(r.Body)

	err = d.Decode(&user)
	if err != nil || user == nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode json: %v", err))
		return
	}

	authUser := r.Context().Value(api.UserKey).(*api.User)

	if authUser.ID != id {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("user id mismatch: URL: %d, Authenticated: %d", id, user.ID))
		return
	}

	if authUser.ID != user.ID {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("user id mismatch: Body: %d, Authenticated: %d", user.ID, user.ID))
		return
	}

	//use authenticated user hash since it is not sent in request
	user.Hash = authUser.Hash

	err = api.UpdateUser(r.Context(), user)
	if !checkAPIError(w, r, err) {
		return
	}

	user, err = api.ReadUser(r.Context(), user.ID)
	if !checkAPIError(w, r, err) {
		return
	}
	if user == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find user, but just updated"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(user)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//POST /users/:id/password
func handleChangeUserPassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
		return
	}

	var req *ChangeUserPasswordRequest
	d := json.NewDecoder(r.Body)

	err = d.Decode(&req)
	if err != nil || req == nil {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode json: %v", err))
		return
	}

	user := r.Context().Value(api.UserKey).(*api.User)

	if user.ID != id {
		handleError(w, r, http.StatusBadRequest, fmt.Errorf("user id mismatch: URL: %d, Authenticated: %d", id, user.ID))
		return
	}

	err = user.ChangePassword(r.Context(), req.OldPassword, req.NewPassword)
	if !checkAPIError(w, r, err) {
		return
	}

	user, err = api.ReadUser(r.Context(), user.ID)
	if !checkAPIError(w, r, err) {
		return
	}
	if user == nil {
		handleError(w, r, http.StatusNotFound, errors.New("Could not find user, but just updated"))
		return
	}

	tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
	err = tx.Commit()
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
		return
	}

	e := json.NewEncoder(w)
	err = e.Encode(user)
	if err != nil {
		handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
	}
}

//POST /auth
func handleAuthenticate(s SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req *AuthenticateRequest
		d := json.NewDecoder(r.Body)

		err := d.Decode(&req)
		if err != nil || req == nil {
			handleError(w, r, http.StatusBadRequest, fmt.Errorf("Could not decode json: %v", err))
			return
		}

		user, err := api.ReadUserByEmail(r.Context(), req.Email)
		if !checkAPIError(w, r, err) {
			return
		}
		if user == nil {
			handleError(w, r, http.StatusNotFound, errors.New("Could not find user"))
			return
		}

		err = user.Authenticate(r.Context(), req.Password)
		if err != nil {
			handleError(w, r, http.StatusForbidden, fmt.Errorf("Could not authenticate user: %v", err))
			return
		}

		tx := r.Context().Value(api.TransactionKey).(*sql.Tx)
		err = tx.Commit()
		if err != nil {
			handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not commit transaction: %v", err))
			return
		}

		key, err := s.Create(user.ID)
		if err != nil {
			handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could not create session: %v", err))
			return
		}

		e := json.NewEncoder(w)
		err = e.Encode(&AuthenticateResponse{SessionKey: key, User: user})
		if err != nil {
			handleError(w, r, http.StatusInternalServerError, fmt.Errorf("Could encode json: %v", err))
		}
	}
}
