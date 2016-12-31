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

//POST /users
func handleCreateUserWithCredentials(w http.ResponseWriter, r *http.Request) *handlerResponse {
	var req *CreateUserRequest
	d := json.NewDecoder(r.Body)

	err := d.Decode(&req)
	if err != nil || req == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode json: %v", err))
	}

	id, err := api.CreateUserWithCredentials(r.Context(), req.Email, req.Password, req.Name)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	user, err := api.ReadUser(r.Context(), id)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	if user == nil {
		return handleError(http.StatusInternalServerError, errors.New("Could not find user, but just created"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: user}
}

//GET /users/:id
func handleReadUser(w http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	user, err := api.ReadUser(r.Context(), id)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if user == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find user"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: user}
}

//POST /users/:id
func handleUpdateUser(w http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	var user *api.User
	d := json.NewDecoder(r.Body)

	err = d.Decode(&user)
	if err != nil || user == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode json: %v", err))
	}

	authUser := r.Context().Value(api.UserKey).(*api.User)

	if authUser.ID != id {
		return handleError(http.StatusBadRequest, fmt.Errorf("user id mismatch: URL: %d, Authenticated: %d", id, user.ID))
	}

	if authUser.ID != user.ID {
		return handleError(http.StatusBadRequest, fmt.Errorf("user id mismatch: Body: %d, Authenticated: %d", user.ID, user.ID))
	}

	//use authenticated user hash since it is not sent in request
	user.Hash = authUser.Hash

	err = api.UpdateUser(r.Context(), user)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	user, err = api.ReadUser(r.Context(), user.ID)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if user == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find user, but just updated"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: user}
}

//POST /users/:id/password
func handleChangeUserPassword(w http.ResponseWriter, r *http.Request) *handlerResponse {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode id: %v", err))
	}

	var req *ChangeUserPasswordRequest
	d := json.NewDecoder(r.Body)

	err = d.Decode(&req)
	if err != nil || req == nil {
		return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode json: %v", err))
	}

	user := r.Context().Value(api.UserKey).(*api.User)

	if user.ID != id {
		return handleError(http.StatusBadRequest, fmt.Errorf("user id mismatch: URL: %d, Authenticated: %d", id, user.ID))
	}

	err = user.ChangePassword(r.Context(), req.OldPassword, req.NewPassword)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}

	user, err = api.ReadUser(r.Context(), user.ID)
	if resp := checkAPIError(err); resp != nil {
		return resp
	}
	if user == nil {
		return handleError(http.StatusNotFound, errors.New("Could not find user, but just updated"))
	}

	return &handlerResponse{Code: http.StatusOK, Body: user}
}

//POST /auth
func handleAuthenticate(s SessionStore) returnHandler {
	return func(w http.ResponseWriter, r *http.Request) *handlerResponse {
		var req *AuthenticateRequest
		d := json.NewDecoder(r.Body)

		err := d.Decode(&req)
		if err != nil || req == nil {
			return handleError(http.StatusBadRequest, fmt.Errorf("Could not decode json: %v", err))
		}

		if req.Email == "" || req.Password == "" {
			return handleError(http.StatusBadRequest, errors.New("email or password empty"))
		}

		user, err := api.ReadUserByEmail(r.Context(), req.Email)
		if resp := checkAPIError(err); resp != nil {
			return resp
		}
		if user == nil {
			return handleError(http.StatusUnauthorized, errors.New("Could not find user"))
		}

		err = user.Authenticate(r.Context(), req.Password)
		if err != nil {
			return handleError(http.StatusUnauthorized, fmt.Errorf("Could not authenticate user %d:%s: %v", user.ID, user.Email, err))
		}

		key, err := s.Create(user.ID)
		if err != nil {
			return handleError(http.StatusInternalServerError, fmt.Errorf("Could not create session: %v", err))
		}

		return &handlerResponse{Code: http.StatusOK, Body: &AuthenticateResponse{SessionKey: key, User: user}}
	}
}
