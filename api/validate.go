package api

import "fmt"

//ValidateString returns an error if the given value is not within the parameters
func ValidateString(field, value string, max int) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", field)
	} else if len(value) > max {
		return fmt.Errorf("%s length (%d) was more than maximum allowed (%d)", field, len(value), max)
	}
	return nil
}
