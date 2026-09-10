package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"strconv"
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

type Member struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Online   bool   `json:"online"`
}

type ConversationDetails struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Members []Member `json:"members"`
}

var db *sql.DB

func main() {
	var err error

	dsn := "root:12345678@tcp(localhost:3306)/messaging_app"

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/newUser", NewUser)
	http.HandleFunc("/me", handleMe)
	http.HandleFunc("/logout", LogoutHandler)
	http.HandleFunc("/del", delUser)
	http.HandleFunc("/conversations", handleConversations)
	http.HandleFunc("/convoInfo/{convoID}", convoInfo)
	http.HandleFunc("/joinConvo/{convoID}", JoinConvo)
	http.HandleFunc("/conversations/{convoID}/members/leave", leaveConvo)

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
		getUserConversation(w, r)
	case http.MethodPost:
		CreateConversation(w, r)
	}
}

// func handleUsers(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")

// 	switch r.Method {
// 	case http.MethodGet:
// 		getUsers(w, r)
// 	case http.MethodPost:
// 		LoginHandler(w, r)
// 	default:
// 		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
// 	}
// }

func getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT username FROM users")
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
// 	_, err := db.Exec(query, u.Name)
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

	_, err = db.Exec("INSERT INTO users (username) VALUES (?)", u.Name)

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

	err = db.QueryRow(query, u.Name).Scan(&UserIn)

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

	query = "INSERT INTO sessions (username, session_token, createdAt) values (?, ?, ?)"

	_, err = db.Exec(query, UserIn, sessionID, time.Now())

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

	err = db.QueryRow("SELECT username FROM sessions WHERE session_token = ?", cookie.Value).Scan(&username)

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

	query := "DELETE FROM sessions where session_token = ?"

	_, err = db.Exec(query, cookie.Value)

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

	err = db.QueryRow("SELECT username FROM sessions WHERE session_token = ?", cookie.Value).Scan(&usern)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := "DELETE from users WHERE username = ?"

	_, err = db.Exec(query, usern)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM sessions where session_token = ?", cookie.Value)

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

func getUserConversation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var userId int

	cookie, err := r.Cookie("session_token")

	if err != nil {
		http.Error(w, "Unauthorzed bitrh, get away from my screen", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(`SELECT users.id FROM sessions JOIN users ON users.username = sessions.username WHERE sessions.session_token = ?`, cookie.Value).Scan(&userId)

	if err != nil {
		http.Error(w, "the user id was not obtained from the cookies in the db", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(`SELECT conversations.name FROM user_in_conversation JOIN conversations ON user_in_conversation.conversation_id = conversations.id WHERE user_in_conversation.user_id = ?`, userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var conversations []string

	for rows.Next() {
		var name string

		err := rows.Scan(&name)
		if err != nil {
			http.Error(w, "aint scan the conversation", http.StatusInternalServerError)
			return

		}

		conversations = append(conversations, name)
	}

	// w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)

}

func CreateConversation(w http.ResponseWriter, r *http.Request) {

	var conv Conversation

	cookie, err := r.Cookie("session_token")

	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Bitch no cookie to you", http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&conv)
	if err != nil {
		http.Error(w, "The new convo json was not decoded", http.StatusInternalServerError)
		return
	}

	// err = models.CreateConvo(conv.Name)

	result, err := db.Exec("INSERT INTO conversations (name) VALUES (?)", conv.Name)

	if err != nil {
		http.Error(w, "Something happened while inserting into 'conversations' table in db", http.StatusInternalServerError)
		return
	}

	conversationID, err := result.LastInsertId()

	if err != nil {
		http.Error(w, "Problem occured when getting the id of the convo", http.StatusInternalServerError)
		return
	}

	var userId int

	err = db.QueryRow(`SELECT users.id FROM sessions JOIN users ON sessions.username = users.username WHERE sessions.session_token = ?`, cookie.Value).Scan(&userId)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User doesnt exist", http.StatusInternalServerError)
			return
		}

		http.Error(w, "Problem while scanning for user ID occured", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT into user_in_conversation (user_id, conversation_id) VALUES (?, ?)", userId, conversationID)

	if err != nil {
		http.Error(w, "something happend while inserting data into 'user_in_conversation' table", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "The new convo was created",
	})

}

func JoinConvo(w http.ResponseWriter, r *http.Request) {

	var userId int

	convoID, err := strconv.Atoi(r.PathValue("convoID"))

	if err != nil {
		http.Error(w, "Invalid convID", http.StatusInternalServerError)
		return
	}
	cookie, err := r.Cookie("session_token")

	err = db.QueryRow(`SELECT users.id FROM sessions JOIN users ON users.username = sessions.username WHERE sessions.session_token = ?`, cookie.Value).Scan(&userId)

	if err != nil {
		http.Error(w, "the user id was not obtained from the cookies in the db", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT into user_in_conversation (user_id, conversation_id) VALUES (?, ?)", userId, convoID)

	if err != nil {
		http.Error(w, "something happend while inserting data into 'user_in_conversation' table", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "The user was added to the conversation",
	})

}

func convoInfo(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	conversationID, err := strconv.Atoi(r.PathValue("convoID"))
	if err != nil {
		http.Error(w, "Invalid conversation ID", http.StatusBadRequest)
		return
	}

	var conversation ConversationDetails

	err = db.QueryRow(
		"SELECT id, name FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&conversation.ID, &conversation.Name)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Conversation not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Could not obtain conversation", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(`
        SELECT
            users.id,
            users.username,
            EXISTS(
                SELECT 1
                FROM sessions
                WHERE sessions.username = users.username
            ) AS online
        FROM user_in_conversation
        JOIN users
            ON user_in_conversation.user_id = users.id
        WHERE user_in_conversation.conversation_id = ?
    `, conversationID)

	if err != nil {
		http.Error(w, "Could not obtain conversation members", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var member Member

		err := rows.Scan(
			&member.ID,
			&member.Username,
			&member.Online,
		)

		if err != nil {
			http.Error(w, "Could not scan member", http.StatusInternalServerError)
			return
		}

		conversation.Members = append(conversation.Members, member)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Error while reading members", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(conversation)

}

func leaveConvo(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")

	convoId, err := strconv.Atoi(r.PathValue("convoID"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var userId int

	err = db.QueryRow(`SELECT users.id FROM sessions JOIN users ON sessions.username = users.username WHERE sessions.session_token = ?`, cookie.Value).Scan(&userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`DELETE FROM user_in_conversation WHERE conversation_id = ? AND user_id = ?`, convoId, userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User was deleted from the convo",
	})

}
