package models

import (
	"time"
	"Moonbase/src/db"
)

type Message struct
{
	ID int
	UserID int
	ConversationID int
	Content string
	SentAt time.Time
}

func CreateMessage(userID int, conversationID int,content string) error {
	_, err := db.DB.Exec(
		"INSERT INTO messages (user_id, conversation_id, content) VALUES (?, ?, ?)",
		userID,
		conversationID,
		content,
	)
	
	return err
}

func (message Message) DeleteMessage() error {
	_, err := db.DB.Exec(
		"DELETE FROM messages WHERE id = ?",
		message.ID,
	)

	return err
}

func FindMessageByID(id int) (*Message, error) {
	message := &Message{}

	query := `
		SELECT id, user_id, conversation_id, content, sent_at
		FROM messages
		WHERE id = ?
	`

	err := db.DB.QueryRow(query, id).Scan(
		&message.ID,
		&message.UserID,
		&message.ConversationID,
		&message.Content,
		&message.SentAt,
	)

	if err != nil {
		return nil, err
	}

	return message, nil
}

func FindMessagesByConversationID(conversationID int) ([]Message, error) {
	query := `
		SELECT id, user_id, conversation_id, content, sent_at
		FROM messages
		WHERE conversation_id = ?
		ORDER BY sent_at ASC
	`

	rows, err := db.DB.Query(query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message

	for rows.Next() {
		var message Message

		err := rows.Scan(
			&message.ID,
			&message.UserID,
			&message.ConversationID,
			&message.Content,
			&message.SentAt,
		)

		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func FindMessagesByUserID(userID int) ([]Message, error) {
	query := `
		SELECT id, user_id, conversation_id, content, sent_at
		FROM messages
		WHERE user_id = ?
		ORDER BY sent_at ASC
	`

	rows, err := db.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message

	for rows.Next() {
		var message Message

		err := rows.Scan(
			&message.ID,
			&message.UserID,
			&message.ConversationID,
			&message.Content,
			&message.SentAt,
		)

		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (message Message) UpdateContent(content string) error {
	_, err := db.DB.Exec(
		"UPDATE messages SET content = ? WHERE id = ?",
		content,
		message.ID,
	)

	return err
}