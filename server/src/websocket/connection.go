package websocket

import (
    "log"
    "net/http"
    "github.com/gorilla/websocket"
	"Moonbase/src/models"
)

var upgrader = websocket.Upgrader{}

func CreateConnection(w http.ResponseWriter, r *http.Request) (*websocket.Conn, *models.User, error) {

    cookie, err := r.Cookie("session_token")
    if err != nil {
        return nil, nil, err
    }

    session, err := models.FindSessionByToken(cookie.Value)
    if err != nil {
        return nil, nil, err
    }

    user, err := models.FindUserByName(session.Username)
    if err != nil {
        return nil, nil, err
    }

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return nil, nil, err
    }

    log.Println("WebSocket connection established")

    return conn, user, nil
}
