package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

//CreatedField represents a field for a CreatedContent. If Name is model_id, Model will be populated.
type CreatedField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
	Model *Model      `json:"_model,omitempty"`
}

//CreatedContent represents content for a created event
type CreatedContent struct {
	Fields []*CreatedField `json:"fields"`
}

//NoteContent represents content for a note event
type NoteContent struct {
	Note string `json:"note"`
}

//ModifiedField represents a field for a ModifiedContent. If Name is model_id, OldModel and NewModel will be populated.
type ModifiedField struct {
	Name     string      `json:"name"`
	OldValue interface{} `json:"old_value"`
	OldModel *Model      `json:"_old_model,omitempty"`
	NewValue interface{} `json:"new_value"`
	NewModel *Model      `json:"_new_model,omitempty"`
}

//ModifiedContent represents content for a modified event
type ModifiedContent struct {
	Fields []*ModifiedField `json:"fields"`
}

//Event represents an event that has happened.
//UserID should be used when creating and Event and User is used when reading and Event.
type Event struct {
	ID      int64       `json:"-"`
	Date    time.Time   `json:"date"`
	UserID  int64       `json:"user_id"`
	User    *User       `json:"_user,omitempty"`
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

//EventLocation contains information needed to add events for the given type
type EventLocation struct {
	Type    string
	Table   string
	IDField string
}

//CreateEvent creates a new Event for the given type and id with the given fields (ID is ignored and created) and returns its ID or an error if one occurred
func CreateEvent(ctx context.Context, id int64, el EventLocation, event *Event) (eventID int64, err error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	content, err := json.Marshal(event.Content)
	if err != nil {
		return 0, &Error{Description: "Could not marshal content json", Type: ErrorTypeServer, Err: err}
	}

	res, err := tx.Exec(fmt.Sprintf("INSERT INTO %s(%s, user_id, date, type, content) VALUES(?, ?, ?, ?, ?);", el.Table, el.IDField),
		id,
		event.UserID,
		event.Date,
		event.Type,
		content,
	)
	if err != nil {
		return 0, &Error{Description: "Could not insert event", Type: ErrorTypeServer, Err: err}
	}

	id, err = res.LastInsertId()
	if err != nil {
		return 0, &Error{Description: "Could not fetch event id", Type: ErrorTypeServer, Err: err}
	}

	return id, nil
}

//CreateCreatedEvent creates a new Created Event for the given type, id, and content
func CreateCreatedEvent(ctx context.Context, id int64, el EventLocation, c *CreatedContent) (eventID int64, err error) {
	user := ctx.Value(UserKey).(*User)

	return CreateEvent(ctx, id, el, &Event{
		Date:    time.Now(),
		UserID:  user.ID,
		Type:    "created",
		Content: c,
	})
}

//CreateNoteEvent creates a new Note Event for the given type and id with the given note text
func CreateNoteEvent(ctx context.Context, id int64, el EventLocation, note string) (eventID int64, err error) {
	if note == "" {
		return 0, &Error{Description: "Could not validate note", Type: ErrorTypeUser, Err: errors.New("note cannot be empty")}
	}
	c := &NoteContent{Note: note}

	user := ctx.Value(UserKey).(*User)

	return CreateEvent(ctx, id, el, &Event{
		Date:    time.Now(),
		UserID:  user.ID,
		Type:    "note",
		Content: c,
	})
}

//CreateModifiedEvent creates a new Modified Event for the given type, id, and content
func CreateModifiedEvent(ctx context.Context, id int64, el EventLocation, c *ModifiedContent) (eventID int64, err error) {
	user := ctx.Value(UserKey).(*User)

	return CreateEvent(ctx, id, el, &Event{
		Date:    time.Now(),
		UserID:  user.ID,
		Type:    "modified",
		Content: c,
	})
}

//ReadEvents returns the events for the given type and id, or an error if one occurred
func ReadEvents(ctx context.Context, id int64, el EventLocation) ([]*Event, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	var events []*Event

	rows, err := tx.Query(fmt.Sprintf("SELECT id, user_id, date, type, content FROM %s WHERE %s=? ORDER BY date;", el.Table, el.IDField), id)
	if err != nil {
		return nil, &Error{Description: fmt.Sprintf("Could not query events for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
	}
	defer rows.Close()

	for rows.Next() {
		e := new(Event)
		var content []byte

		if err := rows.Scan(&(e.ID), &(e.UserID), &(e.Date), &(e.Type), &content); err != nil {
			return nil, &Error{Description: fmt.Sprintf("Could not scan event row for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
		}

		if e.Type == "created" {
			var created *CreatedContent
			if err := json.Unmarshal(content, &created); err != nil {
				return nil, &Error{Description: fmt.Sprintf("Could not unmarshal created content json for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
			}
			e.Content = created

		} else if e.Type == "note" {
			var note *NoteContent
			if err := json.Unmarshal(content, &note); err != nil {
				return nil, &Error{Description: fmt.Sprintf("Could not unmarshal note content json for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
			}
			e.Content = note

		} else if e.Type == "modified" {
			var mod *ModifiedContent
			if err := json.Unmarshal(content, &mod); err != nil {
				return nil, &Error{Description: fmt.Sprintf("Could not unmarshal modified content json for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
			}
			e.Content = mod
		}

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, &Error{Description: fmt.Sprintf("Could not scan event rows for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
	}

	userCache := make(map[int64]*User)
	modelCache := make(map[int64]*Model)

	//populate models for created and modified events
	for _, e := range events {
		if user, ok := userCache[e.UserID]; ok {
			e.User = user
		} else {
			user, err := ReadUser(ctx, e.UserID)
			if err != nil {
				return nil, &Error{Description: fmt.Sprintf("Could not read event user for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
			}
			e.User = user
			userCache[e.UserID] = user
		}

		if e.Type == "created" {
			content := e.Content.(*CreatedContent)
			for _, f := range content.Fields {
				if f.Name == "model_id" {
					newID := int64(f.Value.(float64))

					if model, ok := modelCache[newID]; ok {
						f.Model = model
					} else {
						model, err := ReadModel(ctx, newID, false)
						if err != nil {
							return nil, &Error{Description: fmt.Sprintf("Could not read created event model for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
						}

						f.Model = model
						modelCache[newID] = model
					}
					break
				}
			}
		} else if e.Type == "modified" {
			content := e.Content.(*ModifiedContent)
			for _, f := range content.Fields {
				if f.Name == "model_id" {
					oldID := int64(f.OldValue.(float64))
					newID := int64(f.NewValue.(float64))

					if oldModel, ok := modelCache[oldID]; ok {
						f.OldModel = oldModel
					} else {
						oldModel, err := ReadModel(ctx, oldID, false)
						if err != nil {
							return nil, &Error{Description: fmt.Sprintf("Could not read modified event oldModel for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
						}

						f.OldModel = oldModel
						modelCache[oldID] = oldModel
					}

					if newModel, ok := modelCache[newID]; ok {
						f.NewModel = newModel
					} else {
						newModel, err := ReadModel(ctx, newID, false)
						if err != nil {
							return nil, &Error{Description: fmt.Sprintf("Could not read modified event newModel for %s(%d)", el.Type, id), Type: ErrorTypeServer, Err: err}
						}

						f.NewModel = newModel
						modelCache[newID] = newModel
					}

					break
				}
			}
		}
	}

	return events, nil
}
