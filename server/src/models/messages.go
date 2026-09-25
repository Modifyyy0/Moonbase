package models

import (
	"Moonbase/src/db"
	"time"
)

type Message struct {
	ID             int
	UserID         int
	Username       string
	ConversationID int
	Content        string
	SentAt         time.Time
	DeliveryStatus string `json:"delivery_status,omitempty"`
}

type Receipt struct {
	MessageID      int
	ConversationID int
	UserID         int
	Status         string
}

func CreateMessage(userID int, conversationID int, content string) (*Message, error) {
	result, err := db.DB.Exec(
		"INSERT INTO messages (user_id, conversation_id, content) VALUES (?, ?, ?)",
		userID,
		conversationID,
		content,
	)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return FindMessageByID(int(id))
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

		err = db.QueryRow(`SELECT username FROM users WHERE id = ?`, message.UserID).Scan(&message.Username)

		if err != nil {
			return nil, err
		}

		if message.UserID != 0 {
			_ = db.QueryRow(`
				SELECT CASE
					WHEN COUNT(read_at) > 0 THEN 'read'
					WHEN COUNT(delivered_at) > 0 THEN 'delivered'
					ELSE 'sent'
				END
				FROM message_receipts
				WHERE message_id = ?
			`, message.ID).Scan(&message.DeliveryStatus)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func IsConversationMember(userID, conversationID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM user_in_conversation
		WHERE user_id = ? AND conversation_id = ?
	`, userID, conversationID).Scan(&count)
	return count > 0, err
}

func CreateMessageReceipts(messageID, conversationID, senderID int) error {
	_, err := db.DB.Exec(`
		INSERT INTO message_receipts (message_id, user_id)
		SELECT ?, user_id FROM user_in_conversation
		WHERE conversation_id = ? AND user_id <> ?
	`, messageID, conversationID, senderID)
	return err
}

func MarkMessageDelivered(messageID, userID int) (*Receipt, error) {
	_, err := db.DB.Exec(`
		INSERT INTO message_receipts (message_id, user_id, delivered_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE delivered_at = COALESCE(delivered_at, CURRENT_TIMESTAMP)
	`, messageID, userID)
	if err != nil {
		return nil, err
	}
	return findReceipt(messageID, userID, "delivered")
}

func MarkMessageRead(messageID, userID int) (*Receipt, error) {
	_, err := db.DB.Exec(`
		INSERT INTO message_receipts (message_id, user_id, delivered_at, read_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE delivered_at = COALESCE(delivered_at, CURRENT_TIMESTAMP), read_at = COALESCE(read_at, CURRENT_TIMESTAMP)
	`, messageID, userID)
	if err != nil {
		return nil, err
	}
	return findReceipt(messageID, userID, "read")
}

func findReceipt(messageID, userID int, status string) (*Receipt, error) {
	var conversationID int
	err := db.DB.QueryRow(`SELECT conversation_id FROM messages WHERE id = ?`, messageID).Scan(&conversationID)
	if err != nil {
		return nil, err
	}
	return &Receipt{MessageID: messageID, ConversationID: conversationID, UserID: userID, Status: status}, nil
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
