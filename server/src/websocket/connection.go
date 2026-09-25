package websocket

import (
    "log"
    "net/http"
    "github.com/gorilla/websocket"
	"Moonbase/src/models"
)

var upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        allowed := []string{
            "http://localhost:5500",
            "http://127.0.0.1:5500",
            "http://localhost:3000",
            "http://localhost:5173",
        }

        for _, o := range allowed {
            if origin == o {
                return true
            }
        }

        return true
    },
}

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
