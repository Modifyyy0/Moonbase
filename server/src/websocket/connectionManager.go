package websocket

import (
	"Moonbase/src/models"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type OutgoingType int

const (
	NewMessage OutgoingType = iota
	UserOnline
	UserOffline
)

type OutgoingMessage struct {
	Type OutgoingType
	Data any
}

type Client struct {
	UserID  int
	Conn    *websocket.Conn
	Message chan OutgoingMessage
	Done    chan struct{}
}

type ConnectionManager struct {
	clients map[int]*Client
	mutex   sync.RWMutex
}

type IncomingMessage struct {
	Type string `json:"type"`
	Data struct {
		ConversationID int    `json:"conversation_id"`
		Content        string `json:"content"`
	} `json:"data"`
}

type PresenceData struct {
	ConversationID int    `json:"conversation_id"`
	UserID         int    `json:"user_id"`
	Username       string `json:"username"`
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		clients: make(map[int]*Client),
	}
}

func (manager *ConnectionManager) AddClient(client *Client) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.clients[client.UserID] = client
}

func (manager *ConnectionManager) RemoveClient(userID int) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	delete(manager.clients, userID)
}

func (manager *ConnectionManager) GetClient(userID int) (*Client, bool) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	client, ok := manager.clients[userID]
	return client, ok
}

func (manager *ConnectionManager) GetClients() []*Client {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	clients := make([]*Client, 0, len(manager.clients))

	for _, client := range manager.clients {
		clients = append(clients, client)
	}

	return clients
}

func WritePump(client *Client) {
	for {
		select {
		case message, ok := <-client.Message:

			if !ok {
				return
			}

			fmt.Println("WritePump received event:", message.Type)

			switch message.Type {

			case NewMessage:

				newMessageData, ok := message.Data.(*models.Message)
				if !ok {
					log.Println("invalid data for NewMessage")
					continue
				}

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

				// Another wasted lookup for now.
				user, err := models.FindUserById(newMessageData.UserID)
				if err != nil {
					log.Println("FindUserById error:", err)
					return
				}

				newMessage.Type = "new_message"
				newMessage.Data.ID = newMessageData.ID
				newMessage.Data.ConversationID = newMessageData.ConversationID
				newMessage.Data.SenderID = newMessageData.UserID
				newMessage.Data.SenderUsername = user.Username
				newMessage.Data.Content = newMessageData.Content
				newMessage.Data.SentTime = newMessageData.SentAt

				if err := client.Conn.WriteJSON(newMessage); err != nil {
					log.Println("WriteJSON error:", err)
					return
				}

			case UserOnline:

				presenceData, ok := message.Data.(PresenceData)
				if !ok {
					log.Println("invalid data for UserOnline")
					continue
				}

				userOnline := struct {
					Type string       `json:"type"`
					Data PresenceData `json:"data"`
				}{
					Type: "user_online",
					Data: presenceData,
				}

				if err := client.Conn.WriteJSON(userOnline); err != nil {
					log.Println("WriteJSON error:", err)
					return
				}

			case UserOffline:

				presenceData, ok := message.Data.(PresenceData)
				if !ok {
					log.Println("invalid data for UserOffline")
					continue
				}

				userOffline := struct {
					Type string       `json:"type"`
					Data PresenceData `json:"data"`
				}{
					Type: "user_offline",
					Data: presenceData,
				}

				if err := client.Conn.WriteJSON(userOffline); err != nil {
					log.Println("WriteJSON error:", err)
					return
				}
			}

		case <-client.Done:
			return
		}
	}
}

func ReadPump(client *Client, manager *ConnectionManager) {
	defer func() {
		SendUserOffline(client, manager)
		manager.RemoveClient(client.UserID)
		close(client.Done)
		client.Conn.Close()
	}()

	for {
		frameType, data, err := client.Conn.ReadMessage()
		if err != nil {
			log.Println("ProcessMessage error:", err)
			return
		}

		if frameType != websocket.TextMessage {
			continue
		}

		var message IncomingMessage

		if err := json.Unmarshal(data, &message); err != nil {
			log.Println("ProcessMessage error:", err)
			continue
		}

		if err := ProcessMessage(client, message, manager); err != nil {
			log.Println("ProcessMessage error:", err)
			return
		}
	}
}

func ProcessMessage(client *Client, message IncomingMessage, manager *ConnectionManager) error {
	switch message.Type {
	case "send_message":
		newMessage, err := models.CreateMessage(
			client.UserID,
			message.Data.ConversationID,
			message.Data.Content,
		)
		if err != nil {
			log.Println("ProcessMessage error:", err)
			return err
		}

		users, err := models.FindAllUserFromConvoByID(message.Data.ConversationID)
		if err != nil {
			log.Println("ProcessMessage error:", err)
			return err
		}

		for _, user := range users {
			targetClient, ok := manager.GetClient(user.ID)

			if !ok {
				continue
			}

			outgoingMessage := OutgoingMessage{
				Type: NewMessage,
				Data: newMessage,
			}

			fmt.Println(
				"Sending MSG",
			)

			select {
			case targetClient.Message <- outgoingMessage:
			case <-targetClient.Done:
				continue
			}
		}
	}

	return nil
}

func SendUserOnline(client *Client, manager *ConnectionManager, username string) {
	 fmt.Println("=== SendUserOnline START ===")

	conversations, err := models.FindAllConvoFromUserByID(client.UserID)
	if err != nil {
		log.Println("Find conversations error:", err)
		return
	}

	fmt.Println("Conversations found:", len(conversations))

	for _, conversation := range conversations {
		fmt.Println("Checking conversation:", conversation.ID)

		users, err := models.FindAllUserFromConvoByID(conversation.ID)
		if err != nil {
			log.Println("Find conversation users error:", err)
			continue
		}
		fmt.Println("Users in conversation:", len(users))

		for _, user := range users {

			targetClient, ok := manager.GetClient(user.ID)
			if !ok {
				continue
			}

			outgoingMessage := OutgoingMessage{
				Type: UserOnline,
				Data: PresenceData{
					ConversationID: conversation.ID,
					UserID:         client.UserID,
					Username:       username,
				},
			}

			fmt.Println(
				"Sending user_online to:",
				user.ID,
				"for conversation:",
				conversation.ID,
			)
			select {
			case targetClient.Message <- outgoingMessage:
			case <-targetClient.Done:
				continue
			}
		}
	}
}

func SendUserOffline(client *Client, manager *ConnectionManager) {
	user, err := models.FindUserById(client.UserID)
	if err != nil {
		log.Println("Find user error:", err)
		return
	}

	conversations, err := models.FindAllConvoFromUserByID(client.UserID)
	if err != nil {
		log.Println("Find conversations error:", err)
		return
	}

	for _, conversation := range conversations {

		users, err := models.FindAllUserFromConvoByID(conversation.ID)
		if err != nil {
			log.Println("Find conversation users error:", err)
			continue
		}

		for _, member := range users {

			targetClient, ok := manager.GetClient(member.ID)
			if !ok {
				continue
			}

			outgoingMessage := OutgoingMessage{
				Type: UserOffline,
				Data: PresenceData{
					ConversationID: conversation.ID,
					UserID:         client.UserID,
					Username:       user.Username,
				},
			}

			fmt.Print("User is offline")
			select {
			case targetClient.Message <- outgoingMessage:
			case <-targetClient.Done:
				continue
			}
		}
	}
}
