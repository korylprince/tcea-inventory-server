package api

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

//Location is an allowed location
type Location string

// Scan implements the Scanner interface.
func (s *Location) Scan(value interface{}) error {
	b := value.([]byte)
	*s = Location(b)
	return nil
}

// Value implements the driver Valuer interface.
func (s Location) Value() (driver.Value, error) {
	return string(s), nil
}

//ReadLocations returns all Locations, or an error if one occurred
func ReadLocations(ctx context.Context) ([]Location, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	var locations []Location

	rows, err := tx.Query("SELECT location FROM location;")
	if err != nil {
		return nil, &Error{Description: "Could not query Locations", Type: ErrorTypeServer, Err: err}
	}

	for rows.Next() {
		var s Location
		err = rows.Scan(&s)
		if err != nil {
			return nil, &Error{Description: "Could not scan Location row", Type: ErrorTypeServer, Err: err}
		}

		locations = append(locations, s)
	}

	err = rows.Err()
	if err != nil {
		return nil, &Error{Description: "Could not scan Location rows", Type: ErrorTypeServer, Err: err}
	}

	return locations, nil
}
