package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

//Content is a generic interface representing event content
type Content interface {
	Validate() error
}

//NoteContent represents content for a note event
type NoteContent struct {
	Note string `json:"note"`
}

//Validate cleans and validates the given NoteContent
func (c *NoteContent) Validate() error {
	c.Note = strings.TrimSpace(c.Note)

	if c.Note == "" {
		return errors.New("note must not be empty")
	}
	return nil
}

//ModifiedContent represents content for a modified event
type ModifiedContent struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

//Validate satisfies the Content interface. Since a ModifiedContent will only ever be created by the server, there is no need to validate it; this function always returns nil
func (c *ModifiedContent) Validate() error {
	return nil
}

//Event represents an event that has happened
type Event struct {
	ID      int64     `json:"-"`
	Date    time.Time `json:"date"`
	UserID  int64     `json:"user_id"`
	Type    string    `json:"type"`
	Content Content   `json:"content"`
}

//User resolves the UserID field to a User
func (e *Event) User(ctx context.Context) (*User, error) {
	return ReadUser(ctx, e.UserID)
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

//CreateCreatedEvent creates a new Created Event for the given type and id
func CreateCreatedEvent(ctx context.Context, id int64, el EventLocation) (eventID int64, err error) {
	user := ctx.Value(UserKey).(*User)

	return CreateEvent(ctx, id, el, &Event{
		Date:   time.Now(),
		UserID: user.ID,
		Type:   "created",
	})
}

//CreateNoteEvent creates a new Note Event for the given type and id with the given note text
func CreateNoteEvent(ctx context.Context, id int64, el EventLocation, note string) (eventID int64, err error) {
	c := &NoteContent{Note: note}

	if err := c.Validate(); err != nil {
		return 0, &Error{Description: "Could not validate note", Type: ErrorTypeUser, Err: err}
	}

	user := ctx.Value(UserKey).(*User)

	return CreateEvent(ctx, id, el, &Event{
		Date:    time.Now(),
		UserID:  user.ID,
		Type:    "note",
		Content: c,
	})
}

//CreateModifiedEvent creates a new Modified Event for the given type and id with the given field and values
func CreateModifiedEvent(ctx context.Context, id int64, el EventLocation, field string, oldValue, newValue interface{}) (eventID int64, err error) {
	user := ctx.Value(UserKey).(*User)

	return CreateEvent(ctx, id, el, &Event{
		Date:   time.Now(),
		UserID: user.ID,
		Type:   "modified",
		Content: &ModifiedContent{
			Field:    field,
			OldValue: oldValue,
			NewValue: newValue,
		},
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

		if e.Type == "note" {
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

	return events, nil
}
