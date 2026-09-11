package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"strconv"

	"Moonbase/src/db"
	"Moonbase/src/models"
)

func HandleConversations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		getUserConversation(w, r)
	case http.MethodPost:
		CreateConversation(w, r)
	}
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

	var conv models.Conversation

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

func ConvoInfo(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	conversationID, err := strconv.Atoi(r.PathValue("convoID"))
	if err != nil {
		http.Error(w, "Invalid conversation ID", http.StatusBadRequest)
		return
	}

	var conversation models.ConversationDetails

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
		var member models.Member

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

func LeaveConvo(w http.ResponseWriter, r *http.Request) {
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
