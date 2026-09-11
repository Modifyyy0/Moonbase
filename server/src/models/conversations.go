package models

import (
	"Moonbase/src/db"
	"time"
)

type Conversation struct {
	ID        int
	Name      string
	CreatedAt time.Time
}

type Member struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type ConversationDetails struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Members []Member `json:"members"`
}

func CreateConvo(name string) error {
	_, err := db.DB.Exec(
		"INSERT INTO conversations (name) VALUES (?)",
		name,
	)

	return err
}

func (convo Conversation) DeleteConvo() error {
	_, err := db.DB.Exec(
		"DELETE FROM conversations WHERE id = ?",
		convo.ID,
	)

	return err
}

func FindConvoById(id int) (*Conversation, error) {
	convo := &Conversation{}

	query := `
		SELECT id, name, created_at
		FROM conversations
		WHERE id = ?
	`

	err := db.DB.QueryRow(query, id).Scan(&convo.ID, &convo.Name, &convo.CreatedAt)

	if err != nil {
		return nil, err
	}

	return convo, nil
}

func FindConvoByName(name string) ([]Conversation, error) {
	query := `
		SELECT id, name, created_at
		FROM conversations
		WHERE name = ?
	`

	rows, err := db.DB.Query(query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convos []Conversation

	for rows.Next() {
		var convo Conversation

		err := rows.Scan(
			&convo.ID,
			&convo.Name,
			&convo.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		convos = append(convos, convo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return convos, nil
}
