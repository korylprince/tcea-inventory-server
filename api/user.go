package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"

	"github.com/go-sql-driver/mysql"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

//User represents an authencatable user
type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Hash  []byte `json:"-"`
	Name  string `json:"name"`
}

//Validate validates the given User
func (u *User) Validate() error {
	if e, err := mail.ParseAddress(fmt.Sprintf("User <%s>", u.Email)); err != nil || e.Address != u.Email {
		if err != nil {
			return fmt.Errorf("email (%s) must be a valid email: %v", u.Email, err)
		}
		return fmt.Errorf("email (%s) must be a valid email", u.Email)
	}
	return ValidateString("name", u.Name, 255)
}

//Authenticate authenticates against the database with the given credentials and returns nil if success or error on failure
func (u *User) Authenticate(ctx context.Context, password string) error {
	return bcrypt.CompareHashAndPassword(u.Hash, []byte(password))
}

//ChangePassword updates the password hash to the given password
func (u *User) ChangePassword(ctx context.Context, oldPassword, newPassword string) error {
	if err := u.Authenticate(ctx, oldPassword); err != nil {
		return &Error{Description: "Could not authenticate password", Type: ErrorTypeUser, Err: errors.New("invalid password")}
	}

	if newPassword == "" {
		return &Error{Description: "Could not validate password", Type: ErrorTypeUser, Err: errors.New("password cannot be empty")}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return &Error{Description: "Could not bcrypt encrypt password", Type: ErrorTypeServer, Err: err}
	}

	u.Hash = hash

	return UpdateUser(ctx, u)
}

//CreateUserWithCredentials creates a new User with the given information and returns it, or an error if one occurred
func CreateUserWithCredentials(ctx context.Context, email, password, name string) (id int64, err error) {
	if password == "" {
		return 0, &Error{Description: "Could not validate password", Type: ErrorTypeUser, Err: errors.New("password cannot be empty")}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return 0, &Error{Description: "Could not bcrypt encrypt password", Type: ErrorTypeServer, Err: err}
	}

	return CreateUser(ctx, &User{Email: email, Hash: hash, Name: name})
}

//CreateUser creates a new User with the given fields (ID is ignored and created) and returns its ID, or an error if one occurred
func CreateUser(ctx context.Context, user *User) (id int64, err error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	if err = user.Validate(); err != nil {
		return 0, &Error{Description: "Could not validate User", Type: ErrorTypeUser, Err: err}
	}

	res, err := tx.Exec("INSERT INTO user(email, hash, name) VALUES(?, ?, ?);", user.Email, user.Hash, user.Name)
	if err != nil {
		if e, ok := err.(*mysql.MySQLError); ok && e.Number == 1062 {
			dup, newErr := ReadUserByEmail(ctx, user.Email)
			if newErr != nil {
				return 0, newErr
			}
			return 0, &Error{Description: "Could not insert User", Type: ErrorTypeDuplicate, Err: err, DuplicateID: dup.ID}
		}
		return 0, &Error{Description: "Could not insert User", Type: ErrorTypeServer, Err: err}
	}

	id, err = res.LastInsertId()
	if err != nil {
		return 0, &Error{Description: "Could not fetch User id", Type: ErrorTypeServer, Err: err}
	}

	return id, nil
}

//ReadUser returns the User with the given id, or an error if one occurred
func ReadUser(ctx context.Context, id int64) (*User, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	user := &User{ID: id}

	row := tx.QueryRow("SELECT email, hash, name FROM user WHERE id=?", id)
	err := row.Scan(&(user.Email), &(user.Hash), &(user.Name))

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, &Error{Description: fmt.Sprintf("Could not query User(%d)", id), Type: ErrorTypeServer, Err: err}
	}

	return user, nil
}

//ReadUserByEmail returns the User with the given email, or an error if one occurred
func ReadUserByEmail(ctx context.Context, email string) (*User, error) {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	user := &User{Email: email}

	row := tx.QueryRow("SELECT id, hash, name FROM user WHERE email=?", email)
	err := row.Scan(&(user.ID), &(user.Hash), &(user.Name))

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, &Error{Description: fmt.Sprintf("Could not query UserByEmail(%s)", email), Type: ErrorTypeServer, Err: err}
	}

	return user, nil
}

//UpdateUser updates the fields for the given User (using the ID field), or returns an error if one occurred
func UpdateUser(ctx context.Context, user *User) error {
	tx := ctx.Value(TransactionKey).(*sql.Tx)

	if err := user.Validate(); err != nil {
		return &Error{Description: "Could not validate User", Type: ErrorTypeUser, Err: err}
	}

	_, err := tx.Exec("UPDATE user SET email=?, hash=?, name=? WHERE id=?;", user.Email, user.Hash, user.Name, user.ID)
	if err != nil {
		if e, ok := err.(*mysql.MySQLError); ok && e.Number == 1062 {
			dup, newErr := ReadUserByEmail(ctx, user.Email)
			if newErr != nil {
				return newErr
			}
			return &Error{Description: fmt.Sprintf("Could not update User(%d)", user.ID), Type: ErrorTypeDuplicate, Err: err, DuplicateID: dup.ID}
		}
		return &Error{Description: fmt.Sprintf("Could not update User(%d)", user.ID), Type: ErrorTypeServer, Err: err}
	}

	return nil
}
