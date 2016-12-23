package httpapi

import "github.com/korylprince/tcea-inventory-server/api"

//AuthenticateResponse is a successful authentication response including the session key and User
type AuthenticateResponse struct {
	SessionKey string    `json:"session_key"`
	User       *api.User `json:"user"`
}
