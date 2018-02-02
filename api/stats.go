package api

import (
	"context"
	"database/sql"
)

//StatsLocation represents Location Stats
type StatsLocation struct {
	Location string `json:"location"`
	Count    int    `json:"count"`
}

//StatsModel represents Model Stats
type StatsModel struct {
	ID           int64  `json:"id"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Count        int    `json:"count"`
}

//StatsStatus represents Status Stats
type StatsStatus struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

//Stats represents device statistics (top 10, etc)
type Stats struct {
	Locations     []*StatsLocation `json:"locations"`
	Models        []*StatsModel    `json:"models"`
	Statuses      []*StatsStatus   `json:"statuses"`
	DeviceCount   int              `json:"device_count"`
	ModelCount    int              `json:"model_count"`
	LocationCount int              `json:"location_count"`
	Devices       []*Device        `json:"devices"`
}

//ReadStats returns Stats, or an error if one occurred.
func ReadStats(ctx context.Context) (*Stats, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	s := new(Stats)

	//DeviceCount
	row := tx.QueryRow("SELECT COUNT(id) FROM device;")
	err := row.Scan(&(s.DeviceCount))

	switch {
	case err == sql.ErrNoRows:
		return nil, &Error{Description: "Could not query Stats.DeviceCount: ErrNoRows", Type: ErrorTypeServer, Err: err}
	case err != nil:
		return nil, &Error{Description: "Could not query Stats.DeviceCount", Type: ErrorTypeServer, Err: err}
	}

	//ModelCount
	row = tx.QueryRow("SELECT COUNT(id) FROM model;")
	err = row.Scan(&(s.ModelCount))

	switch {
	case err == sql.ErrNoRows:
		return nil, &Error{Description: "Could not query Stats.ModelCount: ErrNoRows", Type: ErrorTypeServer, Err: err}
	case err != nil:
		return nil, &Error{Description: "Could not query Stats.ModelCount", Type: ErrorTypeServer, Err: err}
	}

	//LocationCount
	row = tx.QueryRow("SELECT COUNT(location) FROM location;")
	err = row.Scan(&(s.LocationCount))

	switch {
	case err == sql.ErrNoRows:
		return nil, &Error{Description: "Could not query Stats.LocationCount: ErrNoRows", Type: ErrorTypeServer, Err: err}
	case err != nil:
		return nil, &Error{Description: "Could not query Stats.LocationCount", Type: ErrorTypeServer, Err: err}
	}

	//Locations
	rows, err := tx.Query("SELECT location, COUNT(id) as c FROM device GROUP BY location ORDER BY c DESC LIMIT 10;")
	if err != nil {
		return nil, &Error{Description: "Could not query Stats.Locations", Type: ErrorTypeServer, Err: err}
	}
	defer rows.Close()

	for rows.Next() {
		l := new(StatsLocation)

		sErr := rows.Scan(&(l.Location), &(l.Count))
		if sErr != nil {
			return nil, &Error{Description: "Could not scan Stats.Locations row", Type: ErrorTypeServer, Err: sErr}
		}

		s.Locations = append(s.Locations, l)
	}

	err = rows.Err()
	if err != nil {
		return nil, &Error{Description: "Could not scan Stats.Locations rows", Type: ErrorTypeServer, Err: err}
	}

	//Models
	rows, err = tx.Query("SELECT d.model_id, m.manufacturer, m.model, COUNT(d.id) as c FROM device AS d JOIN model AS m ON d.model_id = m.id GROUP BY d.model_id ORDER BY c DESC LIMIT 10;")
	if err != nil {
		return nil, &Error{Description: "Could not query Stats.Models", Type: ErrorTypeServer, Err: err}
	}
	defer rows.Close()

	for rows.Next() {
		m := new(StatsModel)

		sErr := rows.Scan(&(m.ID), &(m.Manufacturer), &(m.Model), &(m.Count))
		if sErr != nil {
			return nil, &Error{Description: "Could not scan Stats.Models row", Type: ErrorTypeServer, Err: sErr}
		}

		s.Models = append(s.Models, m)
	}

	err = rows.Err()
	if err != nil {
		return nil, &Error{Description: "Could not scan Stats.Models rows", Type: ErrorTypeServer, Err: err}
	}

	//Statuses
	rows, err = tx.Query("SELECT status, COUNT(id) as c FROM device GROUP BY status ORDER BY c DESC LIMIT 10;")
	if err != nil {
		return nil, &Error{Description: "Could not query Stats.Statuses", Type: ErrorTypeServer, Err: err}
	}
	defer rows.Close()

	for rows.Next() {
		st := new(StatsStatus)

		sErr := rows.Scan(&(st.Status), &(st.Count))
		if sErr != nil {
			return nil, &Error{Description: "Could not scan Stats.Statuses row", Type: ErrorTypeServer, Err: sErr}
		}

		s.Statuses = append(s.Statuses, st)
	}

	err = rows.Err()
	if err != nil {
		return nil, &Error{Description: "Could not scan Stats.Statuses rows", Type: ErrorTypeServer, Err: err}
	}

	//Devices
	rows, err = tx.Query("SELECT d.id, d.serial_number, m.id, m.manufacturer, m.model, d.status, d.location FROM device AS d JOIN model AS m ON d.model_id = m.id ORDER BY d.id DESC LIMIT 10;")
	if err != nil {
		return nil, &Error{Description: "Could not query Stats.Devices", Type: ErrorTypeServer, Err: err}
	}
	defer rows.Close()

	for rows.Next() {
		d := &Device{Model: new(Model)}
		sErr := rows.Scan(&(d.ID), &(d.SerialNumber), &(d.Model.ID), &(d.Model.Manufacturer), &(d.Model.Model), &(d.Status), &(d.Location))
		if sErr != nil {
			return nil, &Error{Description: "Could not scan Stats.Devices row", Type: ErrorTypeServer, Err: sErr}
		}

		s.Devices = append(s.Devices, d)
	}

	err = rows.Err()
	if err != nil {
		return nil, &Error{Description: "Could not scan Stats.Device rows", Type: ErrorTypeServer, Err: err}
	}

	return s, nil
}
