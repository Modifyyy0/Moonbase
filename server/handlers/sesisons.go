package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"Moonbase/src/db"
	"Moonbase/src/models"
)

func generateSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawStdEncoding.EncodeToString(b)
}

func HandleMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized: No session cookie found", http.StatusUnauthorized)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var currentUser models.User

	err = db.QueryRow("SELECT username FROM sessions WHERE session_token = ?", cookie.Value).Scan(&currentUser.Name)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "No users were found", http.StatusUnauthorized)
			return
		}

		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow("SELECT id FROM users WHERE username = ?", currentUser.Name).Scan(&currentUser.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentUser)
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
