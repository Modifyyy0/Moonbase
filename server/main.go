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
)

type User struct {
	Name string `json:"name"`
}

type Session struct {
	Username      string
	Session_token string
	CreatedAt     time.Time
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

	http.HandleFunc("/users", handleUsers)
	http.HandleFunc("/me", handleMe)
	http.HandleFunc("/logout", LogoutHandler)

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func generateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawStdEncoding.EncodeToString(b)
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

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var u User

	err := json.NewDecoder(r.Body).Decode(&u)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := "INSERT INTO users (username) VALUES (?)"

	_, err = db.Exec(query, u.Name)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	_, err = db.Exec(query, u.Name, sessionID, time.Now())

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

	err = db.QueryRow(query, cookie.Value).Scan(&username)

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
