package models

import (
	"Moonbase/src/db"
	"database/sql"
	"fmt"
)

type User struct
{
	ID int
	Username string
}

func CreateUser(name string) error {
	_, err := FindUserByName(name)
	if err == nil {
		return fmt.Errorf("Username %s already exist", name)
	}

	if err != sql.ErrNoRows {
		return err
	}

	_, err = db.DB.Exec(
		"INSERT INTO users (username) VALUES (?)",
		name,
	)

	if err != nil {
		return err
	}

	return nil
}

func (user User) DeleteUser() error {
	_, err := db.DB.Exec(
		"DELETE FROM users WHERE id = ?",
		user.ID,
	)

	if err != nil {
		return err
	}

	return nil
}

func FindUserByName(name string) (*User, error){
	user := &User{}

	query := `
		SELECT id, username
		FROM users
		WHERE username = ?
	`

	err := db.DB.QueryRow(query,name).Scan(&user.ID,&user.Username)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func FindUserById(id int) (*User, error) {
	user:= &User{}

	query := `
		SELECT id, username
		FROM users
		WHERE id = ?
	`

	err := db.DB.QueryRow(query,id).Scan(&user.ID,&user.Username)

	if err != nil {
		return nil, err
	}

	return user, nil
}