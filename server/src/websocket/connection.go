package websocket

import (
    "log"
    "net/http"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func CreateConnection(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		return nil, err
	}

	log.Println("WebSocket connection established")

	return conn, nil
}
