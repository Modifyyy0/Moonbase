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

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "The user json was not decoded", http.StatusBadRequest)
		return
	}

	created := false
	err := db.QueryRow(`SELECT id FROM users WHERE username = ?`, u.Name).Scan(&u.ID)
	if err == sql.ErrNoRows {
		result, insertErr := db.Exec(`INSERT INTO users (username) VALUES (?)`, u.Name)
		if insertErr != nil {
			http.Error(w, "could not create user", http.StatusInternalServerError)
			return
		}

		id, idErr := result.LastInsertId()
		if idErr != nil {
			http.Error(w, "could not read new user ID", http.StatusInternalServerError)
			return
		}
		u.ID = int(id)
		created = true
	} else if err != nil {
		http.Error(w, "could not find user", http.StatusInternalServerError)
		return
	}

	sessionID := generateSessionID()
	_, err = db.Exec(
		`INSERT INTO sessions (username, session_token, createdAt) VALUES (?, ?, ?)`,
		u.Name,
		sessionID,
		time.Now(),
	)
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	if created {
		w.WriteHeader(http.StatusCreated)
	}
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
