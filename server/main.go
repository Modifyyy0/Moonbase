package main

import (
	"Moonbase/src/websocket"
	"Moonbase/src/db"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	Name string `json:"name"`
}

type Session struct {
	Username      string
	Session_token string
	CreatedAt     time.Time
}

type Conversation struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

var manager = websocket.NewConnectionManager()

func main() {
	err := db.Connect()

	if err != nil {
		log.Printf("Could not connect to the database: %v", err)
		return
	}
	defer db.Close()

	http.HandleFunc("/ws", handleWebsocket)

	http.HandleFunc("/users", handleUsers)
	http.HandleFunc("/newUser", NewUser)
	http.HandleFunc("/me", handleMe)
	http.HandleFunc("/logout", LogoutHandler)
	http.HandleFunc("/del", delUser)
	http.HandleFunc("/conversations", handleConversations)

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {

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
	
	websocket.SendUserOnline(client, manager, user.Username)
}

func generateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawStdEncoding.EncodeToString(b)
}

func handleConversations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		getConversation(w, r)
	case http.MethodPost:
		postConversation(w, r)
	}
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		getUsers(w, r)
	case http.MethodPost:
		LoginHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT username FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var u User
		if err := rows.Scan(&u.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	json.NewEncoder(w).Encode(users)
}

// func createUser(w http.ResponseWriter, r *http.Request) {
// 	var u User

// 	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
// 		http.Error(w, "Invalid request payload", http.StatusBadRequest)
// 		return
// 	}

// 	query := "INSERT INTO users (username) VALUES (?)"
// 	_, err := db.DB.Exec(query, u.Name)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(u)
// }

func NewUser(w http.ResponseWriter, r *http.Request) {
	var u User
	err := json.NewDecoder(r.Body).Decode(&u)

	if err != nil {
		http.Error(w, "The user was not created", http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec("INSERT INTO users (username) VALUES (?)", u.Name)

	if err != nil {
		http.Error(w, "The user was not stored in the database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "The user was created and stroed in the db!",
	})

}

func LoginHandler(w http.ResponseWriter, r *http.Request) {

	var u User

	err := json.NewDecoder(r.Body).Decode(&u)

	if err != nil {
		http.Error(w, "The user json was not decoded", http.StatusInternalServerError)
		return
	}

	var UserIn string

	query := "SELECT username FROM users WHERE username = ?"

	err = db.DB.QueryRow(query, u.Name).Scan(&UserIn)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "The user doest exist", http.StatusInternalServerError)
			return
		}

		http.Error(w, "There is something wrong with checking the user", http.StatusInternalServerError)
		return
	}

	sessionID := generateSessionID()

	http.SetCookie(w, &http.Cookie{

		Name:     "session_token",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	query = "INSERT INTO session (username, cookie, createdAt) values (?, ?, ?)"

	_, err = db.DB.Exec(query, UserIn, sessionID, time.Now())

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized: No session cookie found", http.StatusUnauthorized)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var username string

	query := "SELECT username FROM session WHERE cookie = ?"

	err = db.DB.QueryRow(query, cookie.Value).Scan(&username)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"username": username,
	})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	//Logging out means deleting the current session, but i guess first we have to check whos session is that at the moment
	//And then after verifying the session we delete the session and cookies.

	cookie, err := r.Cookie("session_token")

	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := "DELETE FROM session where cookie = ?"

	_, err = db.DB.Exec(query, cookie.Value)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{

		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Mogged out successfully",
	})

}

func delUser(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("session_token")

	if err != nil {
		http.Error(w, "Cookie aint exist", http.StatusUnauthorized)
		return
	}

	var usern string

	err = db.DB.QueryRow("SELECT username FROM session WHERE cookie = ?", cookie.Value).Scan(&usern)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := "DELETE from users WHERE username = ?"

	_, err = db.DB.Exec(query, usern)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.DB.Exec("DELETE FROM session where cookie = ?", cookie.Value)

	if err != nil {
		http.Error(w, "Cookie not found", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{

		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User was deleted succesfully",
	})

}

func getConversation(w http.ResponseWriter, r *http.Request) {

}

func postConversation(w http.ResponseWriter, r *http.Request) {

	var conv Conversation

	// cookie, err := r.Cookie("session_token")

	err := json.NewDecoder(r.Body).Decode(&conv)
	if err != nil {
		http.Error(w, "The new convo json was not decoded", http.StatusInternalServerError)
		return
	}

	// err = models.CreateConvo(conv.Name)

	query := "INSERT INTO conversations (name) VALUES (?)"

	_, err = db.DB.Exec(query, conv.Name)

	if err != nil {
		http.Error(w, "Something happened in the createConvo in db", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "The new convo was created",
	})

}
