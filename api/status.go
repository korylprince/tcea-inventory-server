package api

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

//Status is an allowed status
type Status string

// Scan implements the Scanner interface.
func (s *Status) Scan(value interface{}) error {
	b := value.([]byte)
	*s = Status(b)
	return nil
}

// Value implements the driver Valuer interface.
func (s Status) Value() (driver.Value, error) {
	return string(s), nil
}

//ReadStatuses returns all Statuses, or an error if one occurred
func ReadStatuses(ctx context.Context) ([]Status, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	var statuses []Status

	rows, err := tx.Query("SELECT status FROM status;")
	if err != nil {
		return nil, &Error{Description: "Could not query Statuses", Type: ErrorTypeServer, Err: err}
	}

	for rows.Next() {
		var s Status
		err = rows.Scan(&s)
		if err != nil {
			return nil, &Error{Description: "Could not scan Status row", Type: ErrorTypeServer, Err: err}
		}

		statuses = append(statuses, s)
	}

	err = rows.Err()
	if err != nil {
		return nil, &Error{Description: "Could not scan Status rows", Type: ErrorTypeServer, Err: err}
	}

	return statuses, nil
}
