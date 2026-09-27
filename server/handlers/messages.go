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