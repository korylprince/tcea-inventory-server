package api

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

//ModelEventLocation is the EventLocation for the Model type
var ModelEventLocation = EventLocation{
	Type:    "Model",
	Table:   "model_log",
	IDField: "model_id",
}

//Model represents a device model
type Model struct {
	ID           int64    `json:"id"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
	Events       []*Event `json:"events,omitempty"`
}

//Validate cleans and validates the given Model
func (m *Model) Validate() error {
	m.Manufacturer = strings.TrimSpace(m.Manufacturer)
	m.Model = strings.TrimSpace(m.Model)

	if err := ValidateString("manufacturer", m.Manufacturer, 255); err != nil {
		return err
	}
	if err := ValidateString("model", m.Model, 255); err != nil {
		return err
	}
	return nil
}

//CreateModel creates a new Model with the given fields (ID and Events are ignored and created) and returns its ID, or an error if one occurred
func CreateModel(ctx context.Context, model *Model) (id int64, err error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	if err = model.Validate(); err != nil {
		return 0, &Error{Description: "Could not validate Model", Type: ErrorTypeUser, Err: err}
	}

	res, err := tx.Exec("INSERT INTO model(manufacturer, model) VALUES(?, ?);",
		model.Manufacturer,
		model.Model,
	)
	if err != nil {
		if e, ok := err.(*mysql.MySQLError); ok && e.Number == 1062 {
			dup, newErr := ReadModelByManufacturerAndModel(ctx, model.Manufacturer, model.Model)
			if newErr != nil {
				return 0, newErr
			}
			return 0, &Error{Description: "Could not insert Model", Type: ErrorTypeDuplicate, Err: err, DuplicateID: dup.ID}
		}
		return 0, &Error{Description: "Could not insert Model", Type: ErrorTypeServer, Err: err}
	}

	id, err = res.LastInsertId()
	if err != nil {
		return 0, &Error{Description: "Could not fetch Model id", Type: ErrorTypeServer, Err: err}
	}

	if _, err := CreateCreatedEvent(ctx, id, ModelEventLocation); err != nil {
		return 0, &Error{Description: "Could not add Created Event", Type: ErrorTypeServer, Err: err}
	}

	return id, nil
}

//ReadModel returns the Model with the given id, or an error if one occurred
func ReadModel(ctx context.Context, id int64) (*Model, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	model := &Model{ID: id}

	row := tx.QueryRow("SELECT manufacturer, model FROM model WHERE id=?", id)
	err := row.Scan(&(model.Manufacturer), &(model.Model))

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, &Error{Description: fmt.Sprintf("Could not query Model(%d)", id), Type: ErrorTypeServer, Err: err}
	}

	events, err := ReadEvents(ctx, id, ModelEventLocation)
	if err != nil {
		return nil, err
	}

	model.Events = events

	return model, nil
}

//ReadModelByManufacturerAndModel returns the Model with the given Manufacturer and Model, or an error if one occurred
func ReadModelByManufacturerAndModel(ctx context.Context, manufacturer, model string) (*Model, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	newModel := &Model{Manufacturer: manufacturer, Model: model}

	row := tx.QueryRow("SELECT id FROM model WHERE manufacturer=? AND model=?", manufacturer, model)
	err := row.Scan(&(newModel.ID))

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, &Error{Description: fmt.Sprintf("Could not query ModelByManufacturerAndModel(%s %s)", manufacturer, model), Type: ErrorTypeServer, Err: err}
	}

	events, err := ReadEvents(ctx, newModel.ID, ModelEventLocation)
	if err != nil {
		return nil, err
	}

	newModel.Events = events

	return newModel, nil
}

//UpdateModel updates the fields for the given Model (using the ID field, Events are ignored), or returns an error if one occurred
func UpdateModel(ctx context.Context, model *Model) error {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	if err := model.Validate(); err != nil {
		return &Error{Description: "Could not validate Model", Type: ErrorTypeUser, Err: err}
	}

	oldModel, err := ReadModel(ctx, model.ID)
	if err != nil {
		return &Error{Description: fmt.Sprintf("Could not read old Model(%d)", model.ID), Type: ErrorTypeServer, Err: err}
	}

	_, err = tx.Exec("UPDATE model SET manufacturer=?, model=? WHERE id=?;",
		model.Manufacturer,
		model.Model,
		model.ID,
	)
	if err != nil {
		if e, ok := err.(*mysql.MySQLError); ok && e.Number == 1062 {
			dup, newErr := ReadModelByManufacturerAndModel(ctx, model.Manufacturer, model.Model)
			if newErr != nil {
				return newErr
			}
			return &Error{Description: fmt.Sprintf("Could not update Model(%d)", model.ID), Type: ErrorTypeDuplicate, Err: err, DuplicateID: dup.ID}
		}
		return &Error{Description: fmt.Sprintf("Could not update Model(%d)", model.ID), Type: ErrorTypeServer, Err: err}
	}

	if oldModel.Manufacturer != model.Manufacturer {
		_, err := CreateModifiedEvent(ctx, model.ID, ModelEventLocation, "manufacturer", oldModel.Manufacturer, model.Manufacturer)
		if err != nil {
			return &Error{Description: fmt.Sprintf("Could not created Modified Event for Model(%d).Manufacturer", model.ID), Type: ErrorTypeServer, Err: err}
		}
	}

	if oldModel.Model != model.Model {
		_, err := CreateModifiedEvent(ctx, model.ID, ModelEventLocation, "model", oldModel.Model, model.Model)
		if err != nil {
			return &Error{Description: fmt.Sprintf("Could not created Modified Event for Model(%d).Model", model.ID), Type: ErrorTypeServer, Err: err}
		}
	}

	return nil
}

//ReadModels returns all Models (without Events), or an error if one occurred
func ReadModels(ctx context.Context, includeEvents bool) ([]*Model, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	var models []*Model

	rows, err := tx.Query("SELECT id, manufacturer, model FROM model ORDER BY manufacturer, model;")
	if err != nil {
		return nil, &Error{Description: "Could not query Models", Type: ErrorTypeServer, Err: err}
	}
	defer rows.Close()

	for rows.Next() {
		m := new(Model)
		err = rows.Scan(&(m.ID), &(m.Manufacturer), &(m.Model))
		if err != nil {
			return nil, &Error{Description: "Could not scan Model row", Type: ErrorTypeServer, Err: err}
		}

		models = append(models, m)

	}

	if includeEvents {
		for _, m := range models {
			events, err := ReadEvents(ctx, m.ID, ModelEventLocation)
			if err != nil {
				return nil, err
			}
			m.Events = events
		}
	}

	err = rows.Err()
	if err != nil {
		return nil, &Error{Description: "Could not scan Model rows", Type: ErrorTypeServer, Err: err}
	}

	return models, nil
}
