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

	var userID int

	cookie, err := r.Cookie("session_token")

	if err != nil {
		http.Error(w, "Unauthorzed bitrh, get away from my screen", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(`SELECT users.id FROM sessions JOIN users ON users.username = sessions.username WHERE sessions.session_token = ?`, cookie.Value).Scan(&userID)

	if err != nil {
		http.Error(w, "the user id was not obtained from the cookies in the db", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(`
		SELECT conversations.id, conversations.name, COUNT(all_members.user_id)
		FROM user_in_conversation AS current_members
		JOIN conversations
			ON current_members.conversation_id = conversations.id
		JOIN user_in_conversation AS all_members
			ON all_members.conversation_id = conversations.id
		WHERE current_members.user_id = ?
		GROUP BY conversations.id, conversations.name
		ORDER BY conversations.id
	`, userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	type conversationSummary struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		MemberCount int    `json:"member_count"`
	}

	conversations := make([]conversationSummary, 0)

	for rows.Next() {
		var conversation conversationSummary

		err := rows.Scan(
			&conversation.ID,
			&conversation.Name,
			&conversation.MemberCount,
		)
		if err != nil {
			http.Error(w, "aint scan the conversation", http.StatusInternalServerError)
			return

		}

		conversations = append(conversations, conversation)
	}

	// w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)

}

func CreateConversation(w http.ResponseWriter, r *http.Request) {

	var req models.CreateConversationRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cookie, err := r.Cookie("session_token")

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var currentUserID int

	err = db.QueryRow(`SELECT users.id FROM sessions JOIN users ON users.username = sessions.username WHERE sessions.session_token = ?`, cookie.Value).Scan(&currentUserID)

	result, err := db.Exec(`INSERT INTO conversations(name) VALUES (?)`, req.Name)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	conversationId, err := result.LastInsertId()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`INSERT INTO user_in_conversation(user_id, conversation_id) VALUES (?, ?)`, currentUserID, conversationId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, username := range req.Members {
		var userID int

		err := db.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&userID)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = db.Exec(
			`INSERT INTO user_in_conversation(user_id, conversation_id)
			VALUES (?, ?)`,
			userID,
			conversationId,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var memberCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM user_in_conversation WHERE conversation_id = ?",
		conversationId,
	).Scan(&memberCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation": map[string]interface{}{
			"id":          conversationId,
			"name":        req.Name,
			"memberCount": memberCount,
		},
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
		http.Error(w, "The user is already in the conversation", http.StatusInternalServerError)
		return
	}

	var conversation models.Conversation

	err = db.QueryRow(`SELECT id, name FROM conversations WHERE id = ?`, convoID).Scan(&conversation.ID, &conversation.Name)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation": map[string]interface{}{
			"id":   conversation.ID,
			"name": conversation.Name,
		},
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

	err = db.QueryRow(
		"SELECT COUNT(*) FROM user_in_conversation WHERE conversation_id = ?",
		conversationID,
	).Scan(&conversation.MemberCount)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
