package postgres

import (
	"auth_service/app"
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
)

func (db *DBPostgres) CreateUser(user *app.User) error {
	var userID uint
	row := db.QueryRow("INSERT INTO users (login, password, admin) VALUES($1,$2,$3) RETURNING id",
		user.Login, user.PasswordHash, user.Admin)
	if err := row.Scan(&userID); err != nil {
		return errors.Wrap(err, "query err")
	}

	user.ID = userID
	return nil
}

func (db *DBPostgres) GetUserByLogin(login string) (*app.User, error) {
	var user app.User
	row := db.QueryRow("SELECT id, login, password, admin FROM users WHERE login = $1", login)
	err := row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.Admin)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no user with such login")
		}
		return nil, errors.Wrap(err, "query err")
	}
	return &user, nil
}

func (db *DBPostgres) GetUserByID(id uint) (*app.User, error) {
	var user app.User
	row := db.QueryRow("SELECT id, login, password, admin FROM users WHERE id = $1", id)
	err := row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.Admin)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no user with such id")
		}
		return nil, errors.Wrap(err, "query err")
	}

	return &user, nil
}
