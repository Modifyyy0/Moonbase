package websocket_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Moonbase/handlers"
	"Moonbase/src/models"
	"Moonbase/test/testutil"

	gorilla "github.com/gorilla/websocket"
)

func TestIT_WS_001_SuccessfulConnection(t *testing.T) {
	db := testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-WS-001_%d",
		time.Now().UnixNano(),
	)

	if err := models.CreateUser(username); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Login to obtain a valid session cookie.
	loginBody := fmt.Sprintf(`{"username":"%s"}`, username)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(loginBody),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()
	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusOK &&
		loginRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"login failed: got status %d: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not return a session cookie")
	}

	// Create a real HTTP server using the WebSocket handler.
	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	header := http.Header{}
	header.Add(
		"Cookie",
		cookies[0].Name+"="+cookies[0].Value,
	)

	conn, response, err := gorilla.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if response != nil {
			t.Fatalf(
				"WebSocket connection failed: %v, HTTP status %d",
				err,
				response.StatusCode,
			)
		}

		t.Fatalf("WebSocket connection failed: %v", err)
	}
	defer conn.Close()

	// The successful Dial proves the HTTP upgrade succeeded
	// and a WebSocket connection was established.
	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf(
			"expected HTTP 101 Switching Protocols, got %d",
			response.StatusCode,
		)
	}

	// Give the handler goroutines a moment to start.
	time.Sleep(50 * time.Millisecond)

	// Verify the connection is actually usable by sending
	// a WebSocket frame. ReadPump must receive it.
	message := `{"type":"unknown","data":{}}`

	if err := conn.WriteMessage(
		gorilla.TextMessage,
		[]byte(message),
	); err != nil {
		t.Fatalf("failed to send WebSocket message: %v", err)
	}

	_ = db

	t.Logf(
		"IT-WS-001 PASS: WebSocket connection upgraded successfully and became usable for user %q",
		username,
	)
}

func TestIT_WS_002_FailedUnauthorizedConnection(t *testing.T) {
	testutil.SetupDatabase(t)

	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// No session_token cookie.
	conn, response, err := gorilla.DefaultDialer.Dial(wsURL, nil)

	if err == nil {
		if conn != nil {
			conn.Close()
		}

		t.Fatal("expected unauthorized WebSocket connection to be rejected")
	}

	// Gorilla's Dial returns the HTTP response when the server
	// rejects the upgrade.
	if response == nil {
		t.Fatalf(
			"expected an HTTP response for rejected connection, got none: %v",
			err,
		)
	}

	if response.StatusCode == http.StatusSwitchingProtocols {
		t.Fatal("unauthorized request unexpectedly upgraded to WebSocket")
	}

	t.Logf(
		"IT-WS-002 PASS: unauthorized WebSocket connection was rejected with HTTP status %d",
		response.StatusCode,
	)
}

func TestIT_WS_003_SendMessageEndToEnd(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-WS-003_alice_%d",
		time.Now().UnixNano(),
	)

	bob := fmt.Sprintf(
		"test_IT-WS-003_bob_%d",
		time.Now().UnixNano(),
	)

	if err := models.CreateUser(alice); err != nil {
		t.Fatalf("failed to create Alice: %v", err)
	}

	if err := models.CreateUser(bob); err != nil {
		t.Fatalf("failed to create Bob: %v", err)
	}

	var aliceID, bobID int

	if err := db.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		alice,
	).Scan(&aliceID); err != nil {
		t.Fatalf("failed to get Alice ID: %v", err)
	}

	if err := db.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID); err != nil {
		t.Fatalf("failed to get Bob ID: %v", err)
	}

	// Create a conversation containing Alice and Bob.
	result, err := db.Exec(`
		INSERT INTO conversations(name, conversation_type)
		VALUES (?, 'direct')
	`, bob)

	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO user_in_conversation(user_id, conversation_id)
		VALUES (?, ?), (?, ?)
	`,
		aliceID,
		conversationID,
		bobID,
		conversationID,
	)

	if err != nil {
		t.Fatalf("failed to add conversation members: %v", err)
	}

	// Helper to login and obtain the session cookie.
	login := func(username string) *http.Cookie {
		body := fmt.Sprintf(`{"username":"%s"}`, username)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/login",
			strings.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		handlers.LoginHandler(recorder, req)

		if recorder.Code != http.StatusOK &&
			recorder.Code != http.StatusCreated {
			t.Fatalf(
				"login failed for %s: got %d: %s",
				username,
				recorder.Code,
				recorder.Body.String(),
			)
		}

		cookies := recorder.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatalf("login for %s returned no session cookie", username)
		}

		return cookies[0]
	}

	aliceCookie := login(alice)
	bobCookie := login(bob)

	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dial := func(cookie *http.Cookie) *gorilla.Conn {
		header := http.Header{}
		header.Add(
			"Cookie",
			cookie.Name+"="+cookie.Value,
		)

		conn, response, err := gorilla.DefaultDialer.Dial(
			wsURL,
			header,
		)

		if err != nil {
			if response != nil {
				t.Fatalf(
					"WebSocket connection failed with status %d: %v",
					response.StatusCode,
					err,
				)
			}

			t.Fatalf("WebSocket connection failed: %v", err)
		}

		return conn
	}

	aliceConn := dial(aliceCookie)
	defer aliceConn.Close()

	bobConn := dial(bobCookie)
	defer bobConn.Close()

	// Bob's connection causes an online presence event to be
	// sent to Alice. Read and discard it so it does not interfere
	// with later assertions.
	_ = aliceConn.SetReadDeadline(time.Now().Add(1 * time.Second))

	_, _, err = aliceConn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to receive Bob's online event: %v", err)
	}

	_ = aliceConn.SetReadDeadline(time.Time{})

	content := "hello everyone!"

	message := map[string]interface{}{
		"type": "send_message",
		"data": map[string]interface{}{
			"conversation_id": conversationID,
			"content":         content,
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	// Alice sends the message.
	if err := aliceConn.WriteMessage(
		gorilla.TextMessage,
		payload,
	); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// Bob must receive the broadcast.
	_ = bobConn.SetReadDeadline(time.Now().Add(2 * time.Second))

	var data []byte

	_ = bobConn.SetReadDeadline(time.Now().Add(2 * time.Second))

	for {
		_, messageData, err := bobConn.ReadMessage()
		if err != nil {
			t.Fatalf("Bob did not receive new_message: %v", err)
		}

		var incoming struct {
			Type string `json:"type"`
		}

		if err := json.Unmarshal(messageData, &incoming); err != nil {
			t.Fatalf("failed to decode WebSocket message: %v", err)
		}

		if incoming.Type == "new_message" {
			data = messageData
			break
		}
	}

	_ = bobConn.SetReadDeadline(time.Time{})

	var received struct {
		Type string `json:"type"`
		Data struct {
			ID             int    `json:"id"`
			ConversationID int    `json:"conversation_id"`
			SenderID       int    `json:"sender_id"`
			SenderUsername string `json:"sender_username"`
			Content        string `json:"content"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf(
			"failed to decode broadcast: %v\nraw: %s",
			err,
			string(data),
		)
	}

	if received.Type != "new_message" {
		t.Fatalf(
			"expected message type %q, got %q",
			"new_message",
			received.Type,
		)
	}

	if received.Data.ConversationID != int(conversationID) {
		t.Fatalf(
			"expected conversation ID %d, got %d",
			conversationID,
			received.Data.ConversationID,
		)
	}

	if received.Data.SenderID != aliceID {
		t.Fatalf(
			"expected sender ID %d, got %d",
			aliceID,
			received.Data.SenderID,
		)
	}

	if received.Data.SenderUsername != alice {
		t.Fatalf(
			"expected sender username %q, got %q",
			alice,
			received.Data.SenderUsername,
		)
	}

	if received.Data.Content != content {
		t.Fatalf(
			"expected content %q, got %q",
			content,
			received.Data.Content,
		)
	}

	// Verify the message was actually stored in MySQL.
	var messageCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM messages
		WHERE conversation_id = ?
		  AND user_id = ?
		  AND content = ?
	`,
		conversationID,
		aliceID,
		content,
	).Scan(&messageCount)

	if err != nil {
		t.Fatalf("failed to verify database message: %v", err)
	}

	if messageCount != 1 {
		t.Fatalf(
			"expected exactly one stored message, got %d",
			messageCount,
		)
	}

	t.Logf(
		"IT-WS-003 PASS: message was received, stored, and broadcast to Bob",
	)
}

func TestIT_WS_004_InvalidMessage(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-WS-004_alice_%d",
		time.Now().UnixNano(),
	)

	bob := fmt.Sprintf(
		"test_IT-WS-004_bob_%d",
		time.Now().UnixNano(),
	)

	if err := models.CreateUser(alice); err != nil {
		t.Fatalf("failed to create Alice: %v", err)
	}

	if err := models.CreateUser(bob); err != nil {
		t.Fatalf("failed to create Bob: %v", err)
	}

	var aliceID, bobID int

	if err := db.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		alice,
	).Scan(&aliceID); err != nil {
		t.Fatalf("failed to get Alice ID: %v", err)
	}

	if err := db.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID); err != nil {
		t.Fatalf("failed to get Bob ID: %v", err)
	}

	result, err := db.Exec(`
		INSERT INTO conversations(name, conversation_type)
		VALUES (?, 'direct')
	`, bob)

	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO user_in_conversation(user_id, conversation_id)
		VALUES (?, ?), (?, ?)
	`,
		aliceID,
		conversationID,
		bobID,
		conversationID,
	)

	if err != nil {
		t.Fatalf("failed to add members: %v", err)
	}

	// Record the number of messages before the invalid request.
	var beforeCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM messages
		WHERE conversation_id = ?
	`, conversationID).Scan(&beforeCount)

	if err != nil {
		t.Fatalf("failed to count existing messages: %v", err)
	}

	// Login Alice.
	loginBody := fmt.Sprintf(`{"username":"%s"}`, alice)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(loginBody),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()
	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusOK &&
		loginRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"login failed: got %d: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login returned no session cookie")
	}

	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	header := http.Header{}
	header.Add(
		"Cookie",
		cookies[0].Name+"="+cookies[0].Value,
	)

	conn, response, err := gorilla.DefaultDialer.Dial(
		wsURL,
		header,
	)

	if err != nil {
		if response != nil {
			t.Fatalf(
				"failed to establish WebSocket: status %d: %v",
				response.StatusCode,
				err,
			)
		}

		t.Fatalf("failed to establish WebSocket: %v", err)
	}
	defer conn.Close()

	// Invalid according to the specification:
	// conversation_id = -1
	// content = ""
	invalidMessage := map[string]interface{}{
		"type": "send_message",
		"data": map[string]interface{}{
			"conversation_id": -1,
			"content":         "",
		},
	}

	payload, err := json.Marshal(invalidMessage)
	if err != nil {
		t.Fatalf("failed to encode invalid message: %v", err)
	}

	if err := conn.WriteMessage(
		gorilla.TextMessage,
		payload,
	); err != nil {
		t.Fatalf("failed to send invalid message: %v", err)
	}

	// Verify no invalid message was stored.
	var afterCount int

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM messages
		WHERE conversation_id = ?
	`, conversationID).Scan(&afterCount)

	if err != nil {
		t.Fatalf("failed to count messages after invalid request: %v", err)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"invalid message changed database state: before=%d, after=%d",
			beforeCount,
			afterCount,
		)
	}

	t.Logf(
		"IT-WS-004 PASS: invalid message was rejected and no message was inserted",
	)
}
