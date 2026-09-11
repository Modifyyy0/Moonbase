package websocket

import (
    "log"
    "net/http"
    "github.com/gorilla/websocket"
	"Moonbase/src/models"
)

var upgrader = websocket.Upgrader{}

func CreateConnection(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {

	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil, err
	}

	_, err = models.FindSessionByToken(cookie.Value)
	if err != nil {
        return nil, err
    }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
        return nil, err
    }
	
	log.Println("WebSocket connection established")
	return conn, nil
}
