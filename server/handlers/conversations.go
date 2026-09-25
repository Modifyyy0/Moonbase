package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

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
		SELECT conversations.id, conversations.name, conversations.conversation_type, COUNT(all_members.user_id)
		FROM user_in_conversation AS current_members
		JOIN conversations
			ON current_members.conversation_id = conversations.id
		JOIN user_in_conversation AS all_members
			ON all_members.conversation_id = conversations.id
		WHERE current_members.user_id = ?
		GROUP BY conversations.id, conversations.name, conversations.conversation_type
		ORDER BY conversations.id
	`, userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	type conversationSummary struct {
		ID               int    `json:"id"`
		Name             string `json:"name"`
		ConversationType string `json:"type"`
		MemberCount      int    `json:"member_count"`
	}

	conversations := make([]conversationSummary, 0)

	for rows.Next() {
		var conversation conversationSummary

		err := rows.Scan(
			&conversation.ID,
			&conversation.Name,
			&conversation.ConversationType,
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
		http.Error(w, "Invalid conversation request", http.StatusBadRequest)
		return
	}

	conversationType := strings.ToLower(strings.TrimSpace(req.Type))
	if conversationType == "" {
		conversationType = "group"
	}
	if conversationType != "direct" && conversationType != "group" {
		http.Error(w, "Conversation type must be direct or group", http.StatusBadRequest)
		return
	}

	uniqueMembers := make([]string, 0, len(req.Members))
	seenMembers := make(map[string]bool)
	for _, username := range req.Members {
		username = strings.TrimSpace(username)
		if username != "" && !seenMembers[username] {
			seenMembers[username] = true
			uniqueMembers = append(uniqueMembers, username)
		}
	}
	if conversationType == "direct" && len(uniqueMembers) != 1 {
		http.Error(w, "A direct conversation must have exactly one other member", http.StatusBadRequest)
		return
	}
	if conversationType == "group" && len(uniqueMembers) < 2 {
		http.Error(w, "A group conversation must have at least two other members", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_token")

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var currentUserID int

	err = db.QueryRow(`SELECT users.id FROM sessions JOIN users ON users.username = sessions.username WHERE sessions.session_token = ?`, cookie.Value).Scan(&currentUserID)
	if err != nil {
		http.Error(w, "Invalid session", http.StatusUnauthorized)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	memberIDs := make([]int, 0, len(uniqueMembers))
	for _, username := range uniqueMembers {
		var userID int
		if err := tx.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&userID); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "One or more selected users do not exist", http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if userID == currentUserID {
			http.Error(w, "You cannot add yourself as another member", http.StatusBadRequest)
			return
		}
		memberIDs = append(memberIDs, userID)
	}

	if conversationType == "direct" {
		var existingID int
		var existingName string
		err = tx.QueryRow(`
			SELECT c.id, c.name
			FROM conversations AS c
			JOIN user_in_conversation AS members ON members.conversation_id = c.id
			WHERE c.conversation_type = 'direct'
			  AND members.user_id IN (?, ?)
			GROUP BY c.id, c.name
			HAVING COUNT(DISTINCT members.user_id) = 2
			   AND (SELECT COUNT(*) FROM user_in_conversation WHERE conversation_id = c.id) = 2
		`, currentUserID, memberIDs[0]).Scan(&existingID, &existingName)
		if err == nil {
			if err := tx.Commit(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeConversationCreated(w, http.StatusOK, int64(existingID), existingName, "direct", 2)
			return
		}
		if err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	conversationName := strings.TrimSpace(req.Name)
	if conversationType == "direct" {
		conversationName = uniqueMembers[0]
	} else if conversationName == "" {
		http.Error(w, "A group conversation needs a name", http.StatusBadRequest)
		return
	}

	result, err := tx.Exec(`INSERT INTO conversations(name, conversation_type) VALUES (?, ?)`, conversationName, conversationType)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	conversationId, err := result.LastInsertId()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`INSERT INTO user_in_conversation(user_id, conversation_id) VALUES (?, ?)`, currentUserID, conversationId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, userID := range memberIDs {
		_, err = tx.Exec(
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
	err = tx.QueryRow(
		"SELECT COUNT(*) FROM user_in_conversation WHERE conversation_id = ?",
		conversationId,
	).Scan(&memberCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeConversationCreated(w, http.StatusCreated, conversationId, conversationName, conversationType, memberCount)

}

func writeConversationCreated(w http.ResponseWriter, status int, id int64, name, conversationType string, memberCount int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation": map[string]interface{}{
			"id":          id,
			"name":        name,
			"type":        conversationType,
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

	err = db.QueryRow(`SELECT id, name, conversation_type FROM conversations WHERE id = ?`, convoID).Scan(&conversation.ID, &conversation.Name, &conversation.ConversationType)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation": map[string]interface{}{
			"id":   conversation.ID,
			"name": conversation.Name,
			"type": conversation.ConversationType,
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
		"SELECT id, name, conversation_type FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&conversation.ID, &conversation.Name, &conversation.ConversationType)

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
            users.username
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
		)

		if err != nil {
			http.Error(w, "Could not scan member", http.StatusInternalServerError)
			return
		}

		_, member.Online = manager.GetClient(member.ID)
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
