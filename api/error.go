package api

import "fmt"

//ErrorType are APIError types
type ErrorType int

//ErrorTypes
const (
	ErrorTypeUser ErrorType = iota
	ErrorTypeServer
)

//Error wraps errors in the API
type Error struct {
	Description string
	Type        ErrorType
	Err         error
}

func (e *Error) Error() string {
	if e.Type == ErrorTypeUser {
		return fmt.Sprintf("User Error: %s: %v", e.Description, e.Err)
	}
	return fmt.Sprintf("Server Error: %s: %v", e.Description, e.Err)
}
