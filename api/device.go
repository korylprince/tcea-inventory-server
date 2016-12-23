package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

//DeviceEventLocation is the EventLocation for the Device type
var DeviceEventLocation = EventLocation{
	Type:    "Device",
	Table:   "device_log",
	IDField: "device_id",
}

//Device represents an inventoried device
type Device struct {
	ID           int64    `json:"id"`
	SerialNumber string   `json:"serial_number"`
	ModelID      int64    `json:"model_id"`
	Status       string   `json:"status"`
	Location     string   `json:"location"`
	Events       []*Event `json:"events"`
}

//Model resolves the ModelID field to a Model
func (d *Device) Model(ctx context.Context) (*Model, error) {
	return ReadModel(ctx, d.ModelID)
}

//Validate validates the given Device
func (d *Device) Validate(ctx context.Context) error {
	if err := ValidateString("serial_number", d.SerialNumber, 255); err != nil {
		return err
	}
	if !(d.Status == "Checked Out" || d.Status == "Storage" || d.Status == "Damaged" || d.Status == "Missing") {
		return fmt.Errorf("status (%s) must be a valid status", d.Status)
	}
	if err := ValidateString("location", d.Location, 255); err != nil {
		return err
	}

	if model, err := d.Model(ctx); model == nil || err != nil {
		return fmt.Errorf("model (%d) must be a valid model", d.ModelID)
	}

	return nil
}

//CreateDevice creates a new Device with the given fields (ID and Events are ignored and created) and returns its ID, or an error if one occurred
func CreateDevice(ctx context.Context, device *Device) (id int64, err error) {

	tx := ctx.Value(TransactionKey).(*sql.Tx)

	if err = device.Validate(ctx); err != nil {
		return 0, &Error{Description: "Could not validate Device", Type: ErrorTypeUser, Err: err}
	}

	res, err := tx.Exec("INSERT INTO device(serial_number, model_id, status, location) VALUES(?, ?, ?, ?);",
		device.SerialNumber,
		device.ModelID,
		device.Status,
		device.Location,
	)
	if err != nil {
		log.Printf("%#v\n", err)
		return 0, &Error{Description: "Could not insert Device", Type: ErrorTypeServer, Err: err}
	}

	id, err = res.LastInsertId()
	if err != nil {
		return 0, &Error{Description: "Could not fetch Device", Type: ErrorTypeServer, Err: err}
	}

	if _, err := CreateCreatedEvent(ctx, id, DeviceEventLocation); err != nil {
		return 0, &Error{Description: "Could not add Created Event", Type: ErrorTypeServer, Err: err}
	}

	return id, nil

}

//ReadDevice returns the Device with the given id, or an error if one occurred
func ReadDevice(ctx context.Context, id int64) (*Device, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	device := &Device{ID: id}

	row := tx.QueryRow("SELECT serial_number, model_id, status, location FROM device WHERE id=?", id)
	err := row.Scan(&(device.SerialNumber), &(device.ModelID), &(device.Status), &(device.Location))

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, &Error{Description: fmt.Sprintf("Could not query Device(%d)", id), Type: ErrorTypeServer, Err: err}
	}

	events, err := ReadEvents(ctx, id, DeviceEventLocation)
	if err != nil {
		return nil, err
	}

	device.Events = events

	return device, nil
}

//UpdateDevice updates the fields for the given Device (using the ID field, Events are ignored), or returns an error if one occurred
func UpdateDevice(ctx context.Context, device *Device) error {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	if err := device.Validate(ctx); err != nil {
		return &Error{Description: "Could not validate Device", Type: ErrorTypeUser, Err: err}
	}

	oldDevice, err := ReadDevice(ctx, device.ID)
	if err != nil {
		return &Error{Description: fmt.Sprintf("Could not read old Device(%d)", device.ID), Type: ErrorTypeServer, Err: err}
	}

	_, err = tx.Exec("UPDATE device SET serial_number=?, model_id=?, status=?, location=? WHERE id=?;",
		device.SerialNumber,
		device.ModelID,
		device.Status,
		device.Location,
		device.ID,
	)
	if err != nil {
		return &Error{Description: fmt.Sprintf("Could not update Device(%d)", device.ID), Type: ErrorTypeServer, Err: err}
	}

	if oldDevice.SerialNumber != device.SerialNumber {
		_, err := CreateModifiedEvent(ctx, device.ID, DeviceEventLocation, "serial_number", oldDevice.SerialNumber, device.SerialNumber)
		if err != nil {
			return &Error{Description: fmt.Sprintf("Could not created Modified Event Device(%d).SerialNumber", device.ID), Type: ErrorTypeServer, Err: err}
		}
	}

	if oldDevice.ModelID != device.ModelID {
		_, err := CreateModifiedEvent(ctx, device.ID, DeviceEventLocation, "model", oldDevice.ModelID, device.ModelID)
		if err != nil {
			return &Error{Description: fmt.Sprintf("Could not created Modified Event Device(%d).ModelID", device.ID), Type: ErrorTypeServer, Err: err}
		}
	}

	if oldDevice.Status != device.Status {
		_, err := CreateModifiedEvent(ctx, device.ID, DeviceEventLocation, "status", oldDevice.Status, device.Status)
		if err != nil {
			return &Error{Description: fmt.Sprintf("Could not created Modified Event Device(%d).Status", device.ID), Type: ErrorTypeServer, Err: err}
		}
	}

	if oldDevice.Location != device.Location {
		_, err := CreateModifiedEvent(ctx, device.ID, DeviceEventLocation, "location", oldDevice.Location, device.Location)
		if err != nil {
			return &Error{Description: fmt.Sprintf("Could not created Modified Event Device(%d).Location", device.ID), Type: ErrorTypeServer, Err: err}
		}
	}

	return nil
}
