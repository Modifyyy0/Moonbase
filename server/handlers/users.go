package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"Moonbase/src/db"
	"Moonbase/src/models"
)

func getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT username FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User

	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	json.NewEncoder(w).Encode(users)
}

func NewUser(w http.ResponseWriter, r *http.Request) {
	var u models.User
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

	var u models.User

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

	_, err = db.Exec(`INSERT INTO sessions (username, session_token, createdAt) values (?, ?, ?)`, UserIn, sessionID, time.Now())

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(`SELECT id FROM users WHERE username = ?`, UserIn).Scan(&u.ID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {

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
