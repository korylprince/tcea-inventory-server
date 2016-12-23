package httpapi

import "github.com/korylprince/tcea-inventory-server/api"

//ModelCreateRequest is a Model Create and Note Create request combined
type ModelCreateRequest struct {
	Model *api.Model `json:"model"`
	Note  string     `json:"note"`
}

//DeviceCreateRequest is a Device Create and Note Create request combined
type DeviceCreateRequest struct {
	Device *api.Device `json:"device"`
	Note   string      `json:"note"`
}

//UserCreateRequest is a request to create a new User
type UserCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

//ChangeUserPasswordRequest is a request to change a User's password
type ChangeUserPasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

//NoteRequest is a note encapsulated in a JSON object
type NoteRequest struct {
	Note string `json:"note"`
}

//AuthenticateRequest is an email/password authentication request
type AuthenticateRequest struct {
	Email    string
	Password string
}
