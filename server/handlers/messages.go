package handlers

import (
	"encoding/json"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"strconv"

	"Moonbase/src/models"
)

func GetMessages(w http.ResponseWriter, r *http.Request) {

	convoID, err := strconv.Atoi(r.PathValue("convoID"))

	var messages []models.Message

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messages, err = models.FindMessagesByConversationID(convoID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)

}
