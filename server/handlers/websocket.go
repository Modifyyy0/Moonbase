package handlers

import (
	"net/http"
	"Moonbase/src/websocket"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var manager = websocket.NewConnectionManager()

func HandleWebsocket(w http.ResponseWriter, r *http.Request) {

	conn, user, err := websocket.CreateConnection(w, r)
	if err != nil {
		log.Println(err)
		return
	}

	client := &websocket.Client{
		UserID:  user.ID,
		Conn:    conn,
		Message: make(chan websocket.OutgoingMessage, 16),
		Done:    make(chan struct{}),
	}

	manager.AddClient(client)

	go websocket.WritePump(client)
	go websocket.ReadPump(client, manager)
	
	websocket.SendUserOnline(client, manager, user.Name)
}