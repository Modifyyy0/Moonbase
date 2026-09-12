package websocket

import (
    "github.com/gorilla/websocket"
	"Moonbase/src/models"
	"sync"
	"time"
	"encoding/json"
	"log"
)

type Client struct {
    UserID int
    Conn   *websocket.Conn
    Message   chan *models.Message
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

			if !ok{
				return
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

			//This is another wasted lookup, cuz remember when you look up for msg
			//you already use this lookup might as well do it again
			//BUT for now ill do this, so basically double lookup
			//On optimization this is definitely one of the parts i have to work on
            user, err := models.FindUserById(message.UserID)
            if err != nil {
				log.Println("ProcessMessage error:", err)
                return
            }

            newMessage.Type = "new_message"
            newMessage.Data.ID = message.ID
            newMessage.Data.ConversationID = message.ConversationID
            newMessage.Data.SenderID = message.UserID
            newMessage.Data.SenderUsername = user.Username
            newMessage.Data.Content = message.Content
            newMessage.Data.SentTime = message.SentAt

            if err := client.Conn.WriteJSON(newMessage); err != nil {
				log.Println("ProcessMessage error:", err)
                return
            }

        case <-client.Done:
            return
        }
    }
}

func ReadPump(client *Client, manager *ConnectionManager) {
	defer func() {
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

			select {
			case targetClient.Message <- newMessage:
			case <-targetClient.Done:
				continue
			}
		}
	}

	return nil
}