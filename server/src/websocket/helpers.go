package websocket

import (
	"Moonbase/src/models"
	"encoding/json"
	"github.com/gorilla/websocket"
	"time"
)

func ReceiveMsg(conn *websocket.Conn, userID int) error {

	for {
		frameType, data, err := conn.ReadMessage()

		if err != nil {
			return err
		}

		if frameType != websocket.TextMessage {
			continue
		}

		var message struct {
			Type string `json:"type"`
			Data struct {
				ConversationID int    `json:"conversation_id"`
				Content        string `json:"content"`
			} `json:"data"`
		}

		err = json.Unmarshal(data, &message)

		if err != nil {
			return err
		}

		switch message.Type {
		case "send_message":
			savedMessage, err := models.CreateMessage(
				userID,
				message.Data.ConversationID,
				message.Data.Content,
			)

			if err != nil {
				return err
			}

			err = SendMsg(conn, savedMessage)
			if err != nil {
				return err
			}
		default:
			// unknown message type
		}
	}
}

func SendMsg(conn *websocket.Conn, message *models.Message) error {
	var newMessage struct {
		Type string `json:"type"`
		Data struct {
			ID             int       `json:"id"`
			ConversationID int       `json:"conversation_id"`
			SenderID       int       `json:"sender_id"`
			SenderUsername string    `json:"sender_username"`
			Content        string    `json:"content"`
			SentTime       time.Time `json:"sent_time"`
		} `json:"data"`
	}

	//This is another wasted lookup, cuz remember when you look up for msg
	//you already use this lookup might as well do it again
	//BUT for now ill do this, so basically double lookup
	//On optimization this is definitely one of the parts i have to work on
	user, err := models.FindUserById(message.UserID)
	if err != nil {
		return err
	}

	newMessage.Type = "new_message"

	newMessage.Data.ID = message.ID
	newMessage.Data.ConversationID = message.ConversationID
	newMessage.Data.SenderID = message.UserID
	newMessage.Data.SenderUsername = user.Username
	newMessage.Data.Content = message.Content
	newMessage.Data.SentTime = message.SentAt

	return conn.WriteJSON(newMessage)
}
