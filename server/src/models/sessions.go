package models

import (
	"Moonbase/src/db"
	"time"
)

type Session struct {
	Username       string
	sessionToken   string
	CreatedAt      time.Time
}

func CreateSession(username string, sessionToken  string) error {
	_, err := db.DB.Exec(
		"INSERT INTO sessions (username, session_token) VALUES (?, ?)",
		username,
		sessionToken ,
	)

	return err
}

func FindSessionByToken(sessionToken string) (*Session, error) {
	session := &Session{}

	query := `
		SELECT username, session_token, createdAt
		FROM sessions
		WHERE session_token = ?
	`

	err := db.DB.QueryRow(query, sessionToken).Scan(
		&session.Username,
		&session.sessionToken,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

func DeleteSession(sessionToken string) error {
	_, err := db.DB.Exec(
		"DELETE FROM sessions WHERE session_token = ?",
		sessionToken,
	)

	return err
}

func FindSessionsByUsername(username string) ([]Session, error) {
	query := `
		SELECT username, session_token, createdAt
		FROM sessions
		WHERE username = ?
	`

	rows, err := db.DB.Query(query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session

	for rows.Next() {
		var session Session

		err := rows.Scan(
			&session.Username,
			&session.sessionToken,
			&session.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}