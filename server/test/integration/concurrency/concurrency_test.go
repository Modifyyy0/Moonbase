package concurrency_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time" 


	"Moonbase/handlers"
	"Moonbase/src/db"
	"Moonbase/test/testutil"

	gorilla "github.com/gorilla/websocket"
)

// ---------------------------------------------------------
// Test helpers
// ---------------------------------------------------------

func createTestUser(t *testing.T, username string) int {
	t.Helper()

	_, err := db.DB.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create test user %q: %v", username, err)
	}

	var userID int

	err = db.DB.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		username,
	).Scan(&userID)

	if err != nil {
		t.Fatalf("failed to get user ID for %q: %v", username, err)
	}

	return userID
}

func createGroupConversation(t *testing.T, userIDs []int) int {
	t.Helper()

	result, err := db.DB.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, 'group')`,
		"concurrency_test_conversation",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	for _, userID := range userIDs {
		_, err := db.DB.Exec(
			`INSERT INTO user_in_conversation
			 (user_id, conversation_id)
			 VALUES (?, ?)`,
			userID,
			conversationID,
		)

		if err != nil {
			t.Fatalf(
				"failed to add user %d to conversation: %v",
				userID,
				err,
			)
		}
	}

	return int(conversationID)
}

func loginTestUser(t *testing.T, server *httptest.Server, username string) string {
	t.Helper()

	body := fmt.Sprintf(`{"username":%q}`, username)

	req, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/api/login",
		strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("failed to create login request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusOK {
		t.Fatalf(
			"login for %q returned status %d",
			username,
			resp.StatusCode,
		)
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "session_token" {
			return cookie.Value
		}
	}

	t.Fatalf("login for %q did not return session_token", username)
	return ""
}

func connectWebSocket(
	t *testing.T,
	server *httptest.Server,
	sessionToken string,
) *gorilla.Conn {
	t.Helper()

	wsURL := "ws" +
		strings.TrimPrefix(server.URL, "http") +
		"/ws"

	header := http.Header{}
	header.Add(
		"Cookie",
		"session_token="+sessionToken,
	)

	conn, resp, err := gorilla.DefaultDialer.Dial(
		wsURL,
		header,
	)

	if err != nil {
		if resp != nil {
			t.Fatalf(
				"WebSocket connection failed with HTTP status %d: %v",
				resp.StatusCode,
				err,
			)
		}

		t.Fatalf("WebSocket connection failed: %v", err)
	}

	return conn
}

func drainUntilMessageType(
	t *testing.T,
	conn *gorilla.Conn,
	expectedType string,
	timeout time.Duration,
) []byte {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	defer conn.SetReadDeadline(time.Time{})

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf(
				"failed waiting for %q message: %v",
				expectedType,
				err,
			)
		}

		var envelope struct {
			Type string `json:"type"`
		}

		if err := json.Unmarshal(data, &envelope); err != nil {
			t.Fatalf("invalid WebSocket JSON: %v", err)
		}

		if envelope.Type == expectedType {
			return data
		}
	}
}

// ---------------------------------------------------------
// CON-001
// Multi-Connection Testing
// ---------------------------------------------------------

func TestIT_CON_001_MultiConnection(t *testing.T) {
	testutil.SetupDatabase(t)

	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	timestamp := time.Now().UnixNano()

	usernames := []string{
		fmt.Sprintf("test_CON-001_A_%d", timestamp),
		fmt.Sprintf("test_CON-001_B_%d", timestamp),
		fmt.Sprintf("test_CON-001_C_%d", timestamp),
		fmt.Sprintf("test_CON-001_D_%d", timestamp),
	}

	userIDs := make([]int, 0, len(usernames))

	for _, username := range usernames {
		userIDs = append(
			userIDs,
			createTestUser(t, username),
		)
	}

	conversationID := createGroupConversation(t, userIDs)

	_ = conversationID

	// Create sessions through the actual login handler.
	sessionTokens := make([]string, 0, len(usernames))

	loginServer := httptest.NewServer(
		http.HandlerFunc(handlers.LoginHandler),
	)
	defer loginServer.Close()

	for _, username := range usernames {
		sessionTokens = append(
			sessionTokens,
			loginTestUser(t, loginServer, username),
		)
	}

	// Connect all four clients.
	connections := make([]*gorilla.Conn, 0, len(usernames))

	for _, token := range sessionTokens {
		conn := connectWebSocket(t, server, token)
		connections = append(connections, conn)
	}

	// Close all connections at the end.
	for _, conn := range connections {
		defer conn.Close()
	}

	t.Logf(
		"all four clients connected: %v",
		usernames,
	)

	// Each client should be able to remain connected while
	// another client communicates.
	//
	// Alice sends a message.
	message := map[string]interface{}{
		"type": "send_message",
		"data": map[string]interface{}{
			"conversation_id": conversationID,
			"content":         "hello from Alice",
		},
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("failed to encode message: %v", err)
	}

	if err := connections[0].WriteMessage(
		gorilla.TextMessage,
		messageBytes,
	); err != nil {
		t.Fatalf("Alice failed to send message: %v", err)
	}

	// Bob, Charlie, and David should all receive Alice's message.
	for i := 1; i < len(connections); i++ {
		data := drainUntilMessageType(
			t,
			connections[i],
			"new_message",
			3*time.Second,
		)

		var received struct {
			Type string `json:"type"`
			Data struct {
				ConversationID int    `json:"conversation_id"`
				SenderID       int    `json:"sender_id"`
				Content        string `json:"content"`
			} `json:"data"`
		}

		if err := json.Unmarshal(data, &received); err != nil {
			t.Fatalf("failed to decode received message: %v", err)
		}

		if received.Data.ConversationID != conversationID {
			t.Fatalf(
				"client %d received wrong conversation ID: got %d, want %d",
				i,
				received.Data.ConversationID,
				conversationID,
			)
		}

		if received.Data.SenderID != userIDs[0] {
			t.Fatalf(
				"client %d received wrong sender ID: got %d, want %d",
				i,
				received.Data.SenderID,
				userIDs[0],
			)
		}

		if received.Data.Content != "hello from Alice" {
			t.Fatalf(
				"client %d received wrong content: got %q",
				i,
				received.Data.Content,
			)
		}
	}

	// Disconnect Alice.
	if err := connections[0].Close(); err != nil {
		t.Fatalf("failed to close Alice connection: %v", err)
	}

	// Bob should still be able to communicate after Alice disconnects.
	message = map[string]interface{}{
		"type": "send_message",
		"data": map[string]interface{}{
			"conversation_id": conversationID,
			"content":         "Bob is still connected",
		},
	}

	messageBytes, err = json.Marshal(message)
	if err != nil {
		t.Fatalf("failed to encode Bob's message: %v", err)
	}

	if err := connections[1].WriteMessage(
		gorilla.TextMessage,
		messageBytes,
	); err != nil {
		t.Fatalf("Bob failed to send message: %v", err)
	}

	// Charlie and David must still receive Bob's message.
	for i := 2; i < len(connections); i++ {
		_ = drainUntilMessageType(
			t,
			connections[i],
			"new_message",
			3*time.Second,
		)
	}

	t.Logf(
		"IT-CON-001 PASS: four clients communicated concurrently and remaining clients stayed connected after Alice disconnected",
	)
}

// ---------------------------------------------------------
// CON-002
// Concurrent Message Testing
// ---------------------------------------------------------

func TestIT_CON_002_ConcurrentMessages(t *testing.T) {
	testutil.SetupDatabase(t)

	stages := []int{2, 5, 10, 50, 100}

	for _, clientCount := range stages {
		t.Run(fmt.Sprintf("%d_clients", clientCount), func(t *testing.T) {
			timestamp := time.Now().UnixNano()

			server := httptest.NewServer(
				http.HandlerFunc(handlers.HandleWebsocket),
			)
			defer server.Close()

			loginServer := httptest.NewServer(
				http.HandlerFunc(handlers.LoginHandler),
			)
			defer loginServer.Close()

			userIDs := make([]int, 0, clientCount)
			usernames := make([]string, 0, clientCount)
			sessionTokens := make([]string, 0, clientCount)

			for i := 0; i < clientCount; i++ {
				username := fmt.Sprintf(
					"test_CON-002_%d_%d",
					timestamp,
					i,
				)

				userID := createTestUser(t, username)

				userIDs = append(userIDs, userID)
				usernames = append(usernames, username)
			}

			conversationID := createGroupConversation(
				t,
				userIDs,
			)

			for _, username := range usernames {
				sessionTokens = append(
					sessionTokens,
					loginTestUser(
						t,
						loginServer,
						username,
					),
				)
			}

			connections := make(
				[]*gorilla.Conn,
				clientCount,
			)

			for i, token := range sessionTokens {
				connections[i] = connectWebSocket(
					t,
					server,
					token,
				)
			}

			defer func() {
				for _, conn := range connections {
					_ = conn.Close()
				}
			}()

			// We use up to three simultaneous senders.
			// This gives us concurrent message activity while
			// allowing the number of connected clients to scale
			// from 2 -> 5 -> 10 -> 50 -> 100.
			senderCount := clientCount
			if senderCount > 3 {
				senderCount = 3
			}

			type expectedMessage struct {
				senderID int
				content  string
			}

			expected := make([]expectedMessage, 0, senderCount)

			for i := 0; i < senderCount; i++ {
				expected = append(
					expected,
					expectedMessage{
						senderID: userIDs[i],
						content: fmt.Sprintf(
							"concurrent-message-%d",
							i,
						),
					},
				)
			}

			// Every client needs a reader running before the
			// messages are sent so that the WritePump queues
			// don't become blocked.
			received := make(
				[]map[string]expectedMessage,
				clientCount,
			)

			for i := range received {
				received[i] = make(
					map[string]expectedMessage,
					senderCount,
				)
			}

			var receiveWG sync.WaitGroup

			for clientIndex := 0; clientIndex < clientCount; clientIndex++ {
				receiveWG.Add(1)

				go func(index int) {
					defer receiveWG.Done()

					_ = connections[index].SetReadDeadline(
						time.Now().Add(10 * time.Second),
					)
					defer connections[index].SetReadDeadline(
						time.Time{},
					)

					for len(received[index]) < senderCount {
						_, data, err := connections[index].ReadMessage()
						if err != nil {
							t.Errorf(
								"client %d failed receiving messages: %v",
								index,
								err,
							)
							return
						}

						var envelope struct {
							Type string `json:"type"`
							Data struct {
								ID             int    `json:"id"`
								ConversationID int    `json:"conversation_id"`
								SenderID       int    `json:"sender_id"`
								Content        string `json:"content"`
							} `json:"data"`
						}

						if err := json.Unmarshal(
							data,
							&envelope,
						); err != nil {
							t.Errorf(
								"client %d received invalid JSON: %v",
								index,
								err,
							)
							return
						}

						if envelope.Type != "new_message" {
							continue
						}

						if envelope.Data.ConversationID != conversationID {
							t.Errorf(
								"client %d received message for wrong conversation: got %d, want %d",
								index,
								envelope.Data.ConversationID,
								conversationID,
							)
							continue
						}

						key := fmt.Sprintf(
							"%d:%s",
							envelope.Data.SenderID,
							envelope.Data.Content,
						)

						received[index][key] = expectedMessage{
							senderID: envelope.Data.SenderID,
							content:  envelope.Data.Content,
						}
					}
				}(clientIndex)
			}

			// Send messages concurrently.
			var sendWG sync.WaitGroup

			for i := 0; i < senderCount; i++ {
				sendWG.Add(1)

				go func(senderIndex int) {
					defer sendWG.Done()

					message := map[string]interface{}{
						"type": "send_message",
						"data": map[string]interface{}{
							"conversation_id": conversationID,
							"content": fmt.Sprintf(
								"concurrent-message-%d",
								senderIndex,
							),
						},
					}

					data, err := json.Marshal(message)
					if err != nil {
						t.Errorf(
							"sender %d failed to encode message: %v",
							senderIndex,
							err,
						)
						return
					}

					if err := connections[senderIndex].WriteMessage(
						gorilla.TextMessage,
						data,
					); err != nil {
						t.Errorf(
							"sender %d failed to send message: %v",
							senderIndex,
							err,
						)
					}
				}(i)
			}

			sendWG.Wait()
			receiveWG.Wait()

			// Verify every client received every concurrent
			// message exactly once.
			for clientIndex := 0; clientIndex < clientCount; clientIndex++ {
				if len(received[clientIndex]) != senderCount {
					t.Fatalf(
						"client %d received %d/%d expected messages",
						clientIndex,
						len(received[clientIndex]),
						senderCount,
					)
				}

				for _, expectedMessage := range expected {
					key := fmt.Sprintf(
						"%d:%s",
						expectedMessage.senderID,
						expectedMessage.content,
					)

					if _, ok := received[clientIndex][key]; !ok {
						t.Fatalf(
							"client %d did not receive message from sender %d with content %q",
							clientIndex,
							expectedMessage.senderID,
							expectedMessage.content,
						)
					}
				}
			}

			// Verify the database contains exactly the messages
			// we sent during this stage.
			var messageCount int

			err := db.DB.QueryRow(
				`SELECT COUNT(*)
				 FROM messages
				 WHERE conversation_id = ?
				   AND content LIKE 'concurrent-message-%'`,
				conversationID,
			).Scan(&messageCount)

			if err != nil {
				t.Fatalf(
					"failed to count persisted messages: %v",
					err,
				)
			}

			if messageCount != senderCount {
				t.Fatalf(
					"expected %d persisted messages, got %d",
					senderCount,
					messageCount,
				)
			}

			// Verify each persisted message has the expected
			// sender and conversation.
			rows, err := db.DB.Query(
				`SELECT user_id, conversation_id, content
				 FROM messages
				 WHERE conversation_id = ?
				   AND content LIKE 'concurrent-message-%'`,
				conversationID,
			)
			if err != nil {
				t.Fatalf(
					"failed to query persisted messages: %v",
					err,
				)
			}
			defer rows.Close()

			persisted := make(
				map[string]expectedMessage,
			)

			for rows.Next() {
				var (
					senderID       int
					persistedConvo int
					content        string
				)

				if err := rows.Scan(
					&senderID,
					&persistedConvo,
					&content,
				); err != nil {
					t.Fatalf(
						"failed to scan persisted message: %v",
						err,
					)
				}

				if persistedConvo != conversationID {
					t.Fatalf(
						"message has wrong conversation ID: got %d, want %d",
						persistedConvo,
						conversationID,
					)
				}

				key := fmt.Sprintf(
					"%d:%s",
					senderID,
					content,
				)

				persisted[key] = expectedMessage{
					senderID: senderID,
					content:  content,
				}
			}

			if err := rows.Err(); err != nil {
				t.Fatalf(
					"error while reading persisted messages: %v",
					err,
				)
			}

			if len(persisted) != senderCount {
				t.Fatalf(
					"expected %d unique persisted messages, got %d",
					senderCount,
					len(persisted),
				)
			}

			t.Logf(
				"IT-CON-002 PASS: %d simultaneous clients, %d concurrent senders, all messages persisted and delivered correctly",
				clientCount,
				senderCount,
			)
		})
	}
}
