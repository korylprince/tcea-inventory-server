package api

import "fmt"

//ErrorType are APIError types
type ErrorType int

//ErrorTypes
const (
	ErrorTypeUser ErrorType = iota
	ErrorTypeServer
	ErrorTypeDuplicate
)

//Error wraps errors in the API
type Error struct {
	Description string
	Type        ErrorType
	Err         error
	DuplicateID int64
}

func (e *Error) Error() string {
	if e.Type == ErrorTypeUser {
		return fmt.Sprintf("User Error: %s: %v", e.Description, e.Err)
	} else if e.Type == ErrorTypeServer {
		return fmt.Sprintf("Server Error: %s: %v", e.Description, e.Err)
	}
	return fmt.Sprintf("Duplicate Error (ID: %d): %s: %v", e.DuplicateID, e.Description, e.Err)
}
