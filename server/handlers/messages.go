package handlers

import (
	"encoding/json"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"strconv"

	"Moonbase/src/db"
	"Moonbase/src/models"
)

func GetMessages(w http.ResponseWriter, r *http.Request) {

	convoID, err := strconv.Atoi(r.PathValue("convoID"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var userID int

	err = db.QueryRow(`
		SELECT users.id
		FROM sessions
		JOIN users ON sessions.username = users.username
		WHERE sessions.session_token = ?
	`, cookie.Value).Scan(&userID)

	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var isMember bool

	err = db.QueryRow(`
    SELECT EXISTS(
        SELECT 1
        FROM user_in_conversation
        WHERE user_id = ?
        AND conversation_id = ?
    )
`, userID, convoID).Scan(&isMember)

	if err != nil {
		http.Error(
			w,
			"Could not verify conversation membership",
			http.StatusInternalServerError,
		)
		return
	}

	if !isMember {
		http.Error(
			w,
			"You are not a member of this conversation",
			http.StatusForbidden,
		)
		return
	}

	var conversationExists bool

	err = db.DB.QueryRow(
		`SELECT EXISTS(
			SELECT 1
			FROM conversations
			WHERE id = ?
		)`,
		convoID,
	).Scan(&conversationExists)

	if err != nil {
		http.Error(
			w,
			"Could not verify conversation",
			http.StatusInternalServerError,
		)
		return
	}

	if !conversationExists {
		http.Error(
			w,
			"Conversation not found",
			http.StatusNotFound,
		)
		return
	}

	var messages []models.Message

	messages, err = models.FindMessagesByConversationID(convoID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
