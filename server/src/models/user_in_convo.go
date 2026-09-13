package models

import (
	"Moonbase/src/db"
)

type UserInConversation struct {
	UserID         int
	ConversationID int
}

func AddUserToConvo(userID int, conversationID int) error {
	_, err := db.DB.Exec(
		"INSERT INTO user_in_conversation (user_id, conversation_id) VALUES (?, ?)",
		userID,
		conversationID,
	)

	return err
}

func RemoveUserFromConversation(userID, conversationID int) error {
	_, err := db.DB.Exec(
		`DELETE FROM user_in_conversation
         WHERE user_id = ?
         AND conversation_id = ?`,
		userID,
		conversationID,
	)

	return err
}

func FindAllConvoFromUserByID(userID int) ([]Conversation, error) {
	query := `
		SELECT c.id, c.name, c.created_at
		FROM conversations c
		JOIN user_in_conversation uic
			ON c.id = uic.conversation_id
		WHERE uic.user_id = ?
	`

	rows, err := db.DB.Query(query, userID)
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

func FindAllUserFromConvoByID(conversationID int) ([]User, error) {
	query := `
		SELECT u.id, u.username
		FROM users u
		JOIN user_in_conversation uic
			ON u.id = uic.user_id
		WHERE uic.conversation_id = ?
	`

	rows, err := db.DB.Query(query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User

		err := rows.Scan(
			&user.ID,
			&user.Name,
		)

		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
