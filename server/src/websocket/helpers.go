package websocket

import (
    "github.com/gorilla/websocket"
	"encoding/json"
	"fmt"
)

func ReceiveMsg(conn *websocket.Conn) error {

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

        fmt.Println("Type:", message.Type)
        fmt.Println("Conversation ID:", message.Data.ConversationID)
        fmt.Println("Content:", message.Data.Content)
    }
}