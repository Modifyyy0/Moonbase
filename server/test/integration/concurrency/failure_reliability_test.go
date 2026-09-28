package concurrency_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"
	"io"

	"Moonbase/handlers"
	"Moonbase/src/db"
	"Moonbase/test/testutil"
	"Moonbase/src/models"

	gorilla "github.com/gorilla/websocket"
)

func TestIT_FAIL_001_ConnectionFailure(t *testing.T) {
	testutil.SetupDatabase(t)

	// ---------------------------------------------------------
	// Setup users
	// ---------------------------------------------------------

	timestamp := time.Now().UnixNano()

	aliceUsername := fmt.Sprintf(
		"test_FAIL-001_A_%d",
		timestamp,
	)

	bobUsername := fmt.Sprintf(
		"test_FAIL-001_B_%d",
		timestamp,
	)

	aliceID := createTestUser(t, aliceUsername)
	bobID := createTestUser(t, bobUsername)

	conversationID := createGroupConversation(
		t,
		[]int{aliceID, bobID},
	)

	// ---------------------------------------------------------
	// Login server
	// ---------------------------------------------------------

	loginServer := httptest.NewServer(
		http.HandlerFunc(handlers.LoginHandler),
	)
	defer loginServer.Close()

	aliceToken := loginTestUser(
		t,
		loginServer,
		aliceUsername,
	)

	bobToken := loginTestUser(
		t,
		loginServer,
		bobUsername,
	)

	// ---------------------------------------------------------
	// WebSocket server
	// ---------------------------------------------------------

	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	aliceConn := connectWebSocket(
		t,
		server,
		aliceToken,
	)

	bobConn := connectWebSocket(
		t,
		server,
		bobToken,
	)

	defer bobConn.Close()

	// ---------------------------------------------------------
	// Bob may receive Alice's user_online event.
	//
	// Drain it so it does not interfere with the later
	// user_offline check.
	// ---------------------------------------------------------

	_ = drainUntilMessageType(
		t,
		bobConn,
		"user_online",
		3*time.Second,
	)

	// ---------------------------------------------------------
	// Deliberately terminate Alice's connection.
	// ---------------------------------------------------------

	if err := aliceConn.Close(); err != nil {
		t.Fatalf(
			"failed to close Alice's connection: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Bob should receive Alice's offline event.
	// ---------------------------------------------------------

	offlineData := drainUntilMessageType(
		t,
		bobConn,
		"user_offline",
		5*time.Second,
	)

	var offlineMessage struct {
		Type string `json:"type"`
		Data struct {
			UserID int `json:"user_id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(
		offlineData,
		&offlineMessage,
	); err != nil {
		t.Fatalf(
			"failed to decode user_offline message: %v",
			err,
		)
	}

	if offlineMessage.Data.UserID != aliceID {
		t.Fatalf(
			"user_offline contained wrong user ID: got %d, want %d",
			offlineMessage.Data.UserID,
			aliceID,
		)
	}

	// ---------------------------------------------------------
	// Bob must remain connected and able to communicate.
	// ---------------------------------------------------------

	content := "Bob remains connected"

	message := map[string]interface{}{
		"type": "send_message",
		"data": map[string]interface{}{
			"conversation_id": conversationID,
			"content":         content,
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf(
			"failed to encode Bob's message: %v",
			err,
		)
	}

	if err := bobConn.WriteMessage(
		gorilla.TextMessage,
		data,
	); err != nil {
		t.Fatalf(
			"Bob could not send after Alice disconnected: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Wait for Bob's asynchronous ReadPump to process the
	// message and for CreateMessage() to persist it.
	// ---------------------------------------------------------

	deadline := time.Now().Add(5 * time.Second)

	for {
		var messageCount int

		err := db.DB.QueryRow(
			`SELECT COUNT(*)
			 FROM messages
			 WHERE conversation_id = ?
			   AND user_id = ?
			   AND content = ?`,
			conversationID,
			bobID,
			content,
		).Scan(&messageCount)

		if err != nil {
			t.Fatalf(
				"failed to verify Bob's message: %v",
				err,
			)
		}

		if messageCount == 1 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf(
				"timed out waiting for Bob's message to be persisted; got count %d",
				messageCount,
			)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// ---------------------------------------------------------
	// Final result
	// ---------------------------------------------------------

	t.Log(
		"IT-FAIL-001 PASS: Alice disconnected, server detected the failure, Bob remained connected, and Bob's message was persisted",
	)
}

func TestIT_FAIL_002_SlowClient(t *testing.T) {
	testutil.SetupDatabase(t)

	timestamp := time.Now().UnixNano()

	aliceUsername := fmt.Sprintf(
		"test_FAIL-002_A_%d",
		timestamp,
	)

	bobUsername := fmt.Sprintf(
		"test_FAIL-002_B_%d",
		timestamp,
	)

	charlieUsername := fmt.Sprintf(
		"test_FAIL-002_C_%d",
		timestamp,
	)

	aliceID := createTestUser(t, aliceUsername)
	bobID := createTestUser(t, bobUsername)
	charlieID := createTestUser(t, charlieUsername)

	conversationID := createGroupConversation(
		t,
		[]int{aliceID, bobID, charlieID},
	)

	// ---------------------------------------------------------
	// Login
	// ---------------------------------------------------------

	loginServer := httptest.NewServer(
		http.HandlerFunc(handlers.LoginHandler),
	)
	defer loginServer.Close()

	aliceToken := loginTestUser(
		t,
		loginServer,
		aliceUsername,
	)

	bobToken := loginTestUser(
		t,
		loginServer,
		bobUsername,
	)

	charlieToken := loginTestUser(
		t,
		loginServer,
		charlieUsername,
	)

	// ---------------------------------------------------------
	// WebSocket connections
	// ---------------------------------------------------------

	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	aliceConn := connectWebSocket(
		t,
		server,
		aliceToken,
	)
	defer aliceConn.Close()

	bobConn := connectWebSocket(
		t,
		server,
		bobToken,
	)
	defer bobConn.Close()

	charlieConn := connectWebSocket(
		t,
		server,
		charlieToken,
	)
	defer charlieConn.Close()

	// ---------------------------------------------------------
	// Drain initial presence messages.
	// ---------------------------------------------------------

	_ = drainUntilMessageType(
		t,
		aliceConn,
		"user_online",
		3*time.Second,
	)

	_ = drainUntilMessageType(
		t,
		charlieConn,
		"user_online",
		3*time.Second,
	)

	// IMPORTANT:
	//
	// Bob deliberately does not read from bobConn after this point.
	//
	// Bob is our slow client.
	// ---------------------------------------------------------

	// ---------------------------------------------------------
	// Alice sends a message.
	// ---------------------------------------------------------

	content := fmt.Sprintf(
		"Alice message during slow Bob %d",
		timestamp,
	)

	message := map[string]interface{}{
		"type": "send_message",
		"data": map[string]interface{}{
			"conversation_id": conversationID,
			"content":         content,
		},
	}

	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf(
			"failed to encode Alice's message: %v",
			err,
		)
	}

	if err := aliceConn.WriteMessage(
		gorilla.TextMessage,
		data,
	); err != nil {
		t.Fatalf(
			"Alice could not send message: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Charlie must still receive Alice's message.
	// ---------------------------------------------------------

	received := drainUntilMessageType(
		t,
		charlieConn,
		"new_message",
		5*time.Second,
	)

	var receivedMessage struct {
		Type string `json:"type"`
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}

	if err := json.Unmarshal(
		received,
		&receivedMessage,
	); err != nil {
		t.Fatalf(
			"failed to decode Charlie's message: %v",
			err,
		)
	}

	if receivedMessage.Data.Content != content {
		t.Fatalf(
			"Charlie received wrong message: got %q, want %q",
			receivedMessage.Data.Content,
			content,
		)
	}

	// ---------------------------------------------------------
	// Verify Alice's message was persisted.
	// ---------------------------------------------------------

	deadline := time.Now().Add(5 * time.Second)

	for {
		var count int

		err := db.DB.QueryRow(
			`SELECT COUNT(*)
			 FROM messages
			 WHERE conversation_id = ?
			   AND user_id = ?
			   AND content = ?`,
			conversationID,
			aliceID,
			content,
		).Scan(&count)

		if err != nil {
			t.Fatalf(
				"failed to query message: %v",
				err,
			)
		}

		if count == 1 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf(
				"timed out waiting for Alice's message to be persisted; got count %d",
				count,
			)
		}

		time.Sleep(50 * time.Millisecond)
	}

	// ---------------------------------------------------------
	// Charlie sends another message.
	//
	// This proves communication remains functional while Bob
	// is not consuming outgoing data.
	// ---------------------------------------------------------

	charlieContent := fmt.Sprintf(
		"Charlie message during slow Bob %d",
		timestamp,
	)

	charlieMessage := map[string]interface{}{
		"type": "send_message",
		"data": map[string]interface{}{
			"conversation_id": conversationID,
			"content":         charlieContent,
		},
	}

	charlieData, err := json.Marshal(charlieMessage)
	if err != nil {
		t.Fatalf(
			"failed to encode Charlie's message: %v",
			err,
		)
	}

	if err := charlieConn.WriteMessage(
		gorilla.TextMessage,
		charlieData,
	); err != nil {
		t.Fatalf(
			"Charlie could not send while Bob was slow: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Alice must eventually receive Charlie's message.
	//
	// Alice may receive her OWN message first because the server
	// broadcasts messages to all conversation members, including
	// the sender. Therefore, keep reading until Charlie's message
	// is found.
	// ---------------------------------------------------------

	receivedCharlie := false
	deadline = time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		received = drainUntilMessageType(
			t,
			aliceConn,
			"new_message",
			1*time.Second,
		)

		if err := json.Unmarshal(
			received,
			&receivedMessage,
		); err != nil {
			t.Fatalf(
				"failed to decode Alice's message: %v",
				err,
			)
		}

		if receivedMessage.Data.Content == charlieContent {
			receivedCharlie = true
			break
		}

		// Alice's own earlier message is expected.
	}

	if !receivedCharlie {
		t.Fatalf(
			"Alice never received Charlie's message while Bob was slow",
		)
	}

	t.Log(
		"IT-FAIL-002 PASS: Bob remained a slow/non-reading client while Alice and Charlie continued communicating",
	)

	_ = bobID
}

func TestIT_FAIL_003_GoroutineLeak(t *testing.T) {
	testutil.SetupDatabase(t)

	timestamp := time.Now().UnixNano()

	userIDs := make([]int, 0, 20)
	tokens := make([]string, 0, 20)

	loginServer := httptest.NewServer(
		http.HandlerFunc(handlers.LoginHandler),
	)
	defer loginServer.Close()

	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	// ---------------------------------------------------------
	// Create users and sessions.
	// ---------------------------------------------------------

	for i := 0; i < 20; i++ {
		username := fmt.Sprintf(
			"test_FAIL-003_%d_%d",
			timestamp,
			i,
		)

		userID := createTestUser(t, username)

		token := loginTestUser(
			t,
			loginServer,
			username,
		)

		userIDs = append(userIDs, userID)
		tokens = append(tokens, token)
	}

	// ---------------------------------------------------------
	// Record baseline.
	// ---------------------------------------------------------

	runtime.GC()

	baseline := runtime.NumGoroutine()

	t.Logf(
		"baseline goroutines: %d",
		baseline,
	)

	// ---------------------------------------------------------
	// Repeat connection lifecycle.
	// ---------------------------------------------------------

	for cycle := 0; cycle < 3; cycle++ {

		connections := make([]*gorilla.Conn, 0, len(tokens))

		for _, token := range tokens {
			conn := connectWebSocket(
				t,
				server,
				token,
			)

			connections = append(
				connections,
				conn,
			)
		}

		// -----------------------------------------------------
		// Deliberately close every connection.
		// -----------------------------------------------------

		for _, conn := range connections {
			_ = conn.Close()
		}

		// -----------------------------------------------------
		// Give ReadPump / WritePump time to terminate.
		// -----------------------------------------------------

		time.Sleep(500 * time.Millisecond)
	}

	// ---------------------------------------------------------
	// Allow goroutines to settle.
	// ---------------------------------------------------------

	time.Sleep(1 * time.Second)

	runtime.GC()
	runtime.Gosched()

	final := runtime.NumGoroutine()

	t.Logf(
		"baseline goroutines: %d",
		baseline,
	)

	t.Logf(
		"final goroutines: %d",
		final,
	)

	// ---------------------------------------------------------
	// Allow some tolerance for Go runtime / HTTP machinery.
	// ---------------------------------------------------------

	const tolerance = 20

	if final > baseline+tolerance {
		t.Fatalf(
			"possible goroutine leak: baseline=%d final=%d tolerance=%d",
			baseline,
			final,
			tolerance,
		)
	}

	t.Log(
		"IT-FAIL-003 PASS: goroutine count returned close to baseline after connection cleanup",
	)

	_ = userIDs
}

func TestIT_FAIL_004_Stress(t *testing.T) {
	testutil.SetupDatabase(t)

	const clientCount = 100
	const messagesPerClient = 10
	const usersPerConversation = 10

	timestamp := time.Now().UnixNano()

	loginServer := httptest.NewServer(
		http.HandlerFunc(handlers.LoginHandler),
	)
	defer loginServer.Close()

	server := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer server.Close()

	// ---------------------------------------------------------
	// Create users and sessions.
	// ---------------------------------------------------------

	userIDs := make([]int, clientCount)
	tokens := make([]string, clientCount)

	for i := 0; i < clientCount; i++ {
		username := fmt.Sprintf(
			"test_FAIL-004_%d_%d",
			timestamp,
			i,
		)

		userIDs[i] = createTestUser(
			t,
			username,
		)

		tokens[i] = loginTestUser(
			t,
			loginServer,
			username,
		)
	}

	// ---------------------------------------------------------
	// Create multiple conversations.
	//
	// 100 clients are divided into groups of 10.
	// ---------------------------------------------------------

	conversationIDs := make([]int, clientCount/usersPerConversation)

	for i := 0; i < len(conversationIDs); i++ {
		start := i * usersPerConversation
		end := start + usersPerConversation

		conversationIDs[i] = createGroupConversation(
			t,
			userIDs[start:end],
		)
	}

	// ---------------------------------------------------------
	// Record goroutine baseline.
	// ---------------------------------------------------------

	runtime.GC()

	baseline := runtime.NumGoroutine()

	// ---------------------------------------------------------
	// Connect all clients concurrently.
	// ---------------------------------------------------------

	connections := make([]*gorilla.Conn, clientCount)

	var connectWG sync.WaitGroup
	connectWG.Add(clientCount)

	for i := 0; i < clientCount; i++ {
		i := i

		go func() {
			defer connectWG.Done()

			connections[i] = connectWebSocket(
				t,
				server,
				tokens[i],
			)
		}()
	}

	connectWG.Wait()

	// ---------------------------------------------------------
	// Send messages concurrently.
	// ---------------------------------------------------------

	var messageWG sync.WaitGroup
	messageWG.Add(clientCount)

	for i := 0; i < clientCount; i++ {
		i := i

		go func() {
			defer messageWG.Done()

			conversationIndex := i / usersPerConversation
			conversationID := conversationIDs[conversationIndex]

			for j := 0; j < messagesPerClient; j++ {

				content := fmt.Sprintf(
					"stress-message-%d-%d-%d",
					timestamp,
					i,
					j,
				)

				message := map[string]interface{}{
					"type": "send_message",
					"data": map[string]interface{}{
						"conversation_id": conversationID,
						"content":         content,
					},
				}

				data, err := json.Marshal(message)
				if err != nil {
					t.Errorf(
						"client %d failed to encode message: %v",
						i,
						err,
					)
					return
				}

				if connections[i] == nil {
					t.Errorf(
						"client %d has no WebSocket connection",
						i,
					)
					return
				}

				if err := connections[i].WriteMessage(
					gorilla.TextMessage,
					data,
				); err != nil {
					t.Errorf(
						"client %d failed to send message %d: %v",
						i,
						j,
						err,
					)
					return
				}
			}
		}()
	}

	messageWG.Wait()

	// ---------------------------------------------------------
	// Give server time to finish asynchronous processing.
	// ---------------------------------------------------------

	expectedMessages := clientCount * messagesPerClient

	deadline := time.Now().Add(30 * time.Second)

	lastCount := -1

	for {
		var count int

		err := db.DB.QueryRow(
			`SELECT COUNT(*)
		 FROM messages
		 WHERE content LIKE ?`,
			fmt.Sprintf(
				"stress-message-%d-%%",
				timestamp,
			),
		).Scan(&count)

		if err != nil {
			t.Fatalf(
				"failed to query stress-test messages: %v",
				err,
			)
		}

		if count != lastCount {
			t.Logf(
				"stress progress: %d/%d messages persisted",
				count,
				expectedMessages,
			)
			lastCount = count
		}

		if count >= expectedMessages {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf(
				"stress test timed out: expected %d messages, got %d",
				expectedMessages,
				count,
			)
		}

		time.Sleep(500 * time.Millisecond)
	}

	// ---------------------------------------------------------
	// Disconnect all clients.
	// ---------------------------------------------------------

	for _, conn := range connections {
		if conn != nil {
			_ = conn.Close()
		}
	}

	// ---------------------------------------------------------
	// Allow cleanup.
	// ---------------------------------------------------------

	time.Sleep(2 * time.Second)

	runtime.GC()
	runtime.Gosched()

	final := runtime.NumGoroutine()

	t.Logf(
		"stress test goroutines: baseline=%d final=%d",
		baseline,
		final,
	)

	const tolerance = 50

	if final > baseline+tolerance {
		t.Fatalf(
			"possible goroutine growth after stress test: baseline=%d final=%d",
			baseline,
			final,
		)
	}

	// ---------------------------------------------------------
	// Final database integrity check.
	// ---------------------------------------------------------

	var finalMessageCount int

	err := db.DB.QueryRow(
		`SELECT COUNT(*)
		 FROM messages
		 WHERE content LIKE ?`,
		fmt.Sprintf(
			"stress-message-%d-%%",
			timestamp,
		),
	).Scan(&finalMessageCount)

	if err != nil {
		t.Fatalf(
			"failed final database check: %v",
			err,
		)
	}

	if finalMessageCount != expectedMessages {
		t.Fatalf(
			"unexpected message count after stress test: got %d, want %d",
			finalMessageCount,
			expectedMessages,
		)
	}

	t.Log(
		"IT-FAIL-004 PASS: stress test completed without message loss or uncontrolled goroutine growth",
	)
}

func TestIT_FAIL_005_EndToEnd(t *testing.T) {
	testutil.SetupDatabase(t)

	// ---------------------------------------------------------
	// 1. Create test users
	// ---------------------------------------------------------

	aliceUsername := fmt.Sprintf("test_FAIL-005_alice_%d", time.Now().UnixNano())
	bobUsername := fmt.Sprintf("test_FAIL-005_bob_%d", time.Now().UnixNano())

	aliceID := createTestUser(t, aliceUsername)
	bobID := createTestUser(t, bobUsername)

	// ---------------------------------------------------------
	// 2. Login Alice and Bob
	// ---------------------------------------------------------

	loginServer := httptest.NewServer(
		http.HandlerFunc(handlers.LoginHandler),
	)
	defer loginServer.Close()

	aliceToken := loginTestUser(t, loginServer, aliceUsername)
	bobToken := loginTestUser(t, loginServer, bobUsername)

	t.Log("IT-FAIL-005: Alice and Bob logged in successfully")

	// ---------------------------------------------------------
	// 3. Verify sessions were created
	// ---------------------------------------------------------

	var sessionCount int

	err := db.DB.QueryRow(
		`SELECT COUNT(*)
		 FROM sessions
		 WHERE username IN (?, ?)`,
		aliceUsername,
		bobUsername,
	).Scan(&sessionCount)

	if err != nil {
		t.Fatalf("failed to verify sessions: %v", err)
	}

	if sessionCount != 2 {
		t.Fatalf(
			"expected 2 sessions, got %d",
			sessionCount,
		)
	}

	t.Log("IT-FAIL-005: sessions verified in database")

	// ---------------------------------------------------------
	// 4. Create conversation
	// ---------------------------------------------------------

	conversationID := createGroupConversation(
		t,
		[]int{aliceID, bobID},
	)

	t.Logf(
		"IT-FAIL-005: conversation %d created",
		conversationID,
	)

	// Verify conversation membership
	var memberCount int

	err = db.DB.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE conversation_id = ?`,
		conversationID,
	).Scan(&memberCount)

	if err != nil {
		t.Fatalf(
			"failed to verify conversation members: %v",
			err,
		)
	}

	if memberCount != 2 {
		t.Fatalf(
			"expected 2 conversation members, got %d",
			memberCount,
		)
	}

	// ---------------------------------------------------------
	// 5. Connect Alice and Bob through WebSocket
	// ---------------------------------------------------------

	wsServer := httptest.NewServer(
		http.HandlerFunc(handlers.HandleWebsocket),
	)
	defer wsServer.Close()

	aliceConn := connectWebSocket(
		t,
		wsServer,
		aliceToken,
	)
	defer aliceConn.Close()

	bobConn := connectWebSocket(
		t,
		wsServer,
		bobToken,
	)
	defer bobConn.Close()

	t.Log("IT-FAIL-005: Alice and Bob connected through WebSocket")

	// ---------------------------------------------------------
	// 6. Alice sends a message
	// ---------------------------------------------------------

	messageContent := fmt.Sprintf(
		"IT-FAIL-005 E2E message %d",
		time.Now().UnixNano(),
	)

	sendMessage := map[string]any{
		"type": "send_message",
		"data": map[string]any{
			"conversation_id": conversationID,
			"content":         messageContent,
		},
	}

	err = aliceConn.WriteJSON(sendMessage)
	if err != nil {
		t.Fatalf(
			"Alice failed to send message: %v",
			err,
		)
	}

	t.Log("IT-FAIL-005: Alice sent message")

	// ---------------------------------------------------------
	// 7. Bob receives the message
	// ---------------------------------------------------------

	type receivedMessageData struct {
		ID             int       `json:"id"`
		ConversationID int       `json:"conversation_id"`
		SenderID       int       `json:"sender_id"`
		SenderUsername string    `json:"sender_username"`
		Content        string    `json:"content"`
		SentTime       time.Time `json:"sent_time"`
	}

	type receivedMessage struct {
		Type string               `json:"type"`
		Data receivedMessageData `json:"data"`
	}

	var received receivedMessage

	bobConn.SetReadDeadline(time.Now().Add(5 * time.Second))

	for {
		var message receivedMessage

		err = bobConn.ReadJSON(&message)
		if err != nil {
			t.Fatalf(
				"Bob failed to receive message: %v",
				err,
			)
		}

		if message.Type != "new_message" {
			continue
		}

		received = message
		break
	}

	if received.Data.Content != messageContent {
		t.Fatalf(
			"Bob received incorrect message content: got %q, want %q",
			received.Data.Content,
			messageContent,
		)
	}

	if received.Data.ConversationID != conversationID {
		t.Fatalf(
			"Bob received incorrect conversation ID: got %d, want %d",
			received.Data.ConversationID,
			conversationID,
		)
	}

	if received.Data.SenderID != aliceID {
		t.Fatalf(
			"Bob received incorrect sender ID: got %d, want %d",
			received.Data.SenderID,
			aliceID,
		)
	}

	if received.Data.SenderUsername != aliceUsername {
		t.Fatalf(
			"Bob received incorrect sender username: got %q, want %q",
			received.Data.SenderUsername,
			aliceUsername,
		)
	}

	if received.Data.ID == 0 {
		t.Fatal("Bob received message with invalid message ID 0")
	}

	t.Log("IT-FAIL-005: Bob received Alice's message correctly")

	// ---------------------------------------------------------
	// 8. Verify message exists in database
	// ---------------------------------------------------------

	var dbMessageCount int

	err = db.DB.QueryRow(
		`SELECT COUNT(*)
		 FROM messages
		 WHERE id = ?
		   AND user_id = ?
		   AND conversation_id = ?
		   AND content = ?`,
		received.Data.ID,
		aliceID,
		conversationID,
		messageContent,
	).Scan(&dbMessageCount)

	if err != nil {
		t.Fatalf(
			"failed to verify message in database: %v",
			err,
		)
	}

	if dbMessageCount != 1 {
		t.Fatalf(
			"message was not stored correctly in database: got %d rows",
			dbMessageCount,
		)
	}

	t.Log("IT-FAIL-005: message verified in database")

	// ---------------------------------------------------------
	// 9. Verify Bob's delivery receipt
	// ---------------------------------------------------------

	deadline := time.Now().Add(5 * time.Second)
	delivered := false

	for time.Now().Before(deadline) {
		var deliveredCount int

		err = db.DB.QueryRow(
			`SELECT COUNT(*)
			 FROM message_receipts
			 WHERE message_id = ?
			   AND user_id = ?
			   AND delivered_at IS NOT NULL`,
			received.Data.ID,
			bobID,
		).Scan(&deliveredCount)

		if err != nil {
			t.Fatalf(
				"failed to verify delivery receipt: %v",
				err,
			)
		}

		if deliveredCount == 1 {
			delivered = true
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if !delivered {
		t.Fatal(
			"Bob's delivery receipt was not recorded within 5 seconds",
		)
	}

	t.Log("IT-FAIL-005: delivery receipt verified")

	// ---------------------------------------------------------
	// 10. Bob retrieves message history through HTTP
	// ---------------------------------------------------------

	historyMux := http.NewServeMux()

	historyMux.HandleFunc(
		"GET /api/conversations/{convoID}/messages",
		handlers.GetMessages,
	)

	historyServer := httptest.NewServer(historyMux)
	defer historyServer.Close()

	historyURL := fmt.Sprintf(
		"%s/api/conversations/%d/messages",
		historyServer.URL,
		conversationID,
	)

	req, err := http.NewRequest(
		http.MethodGet,
		historyURL,
		nil,
	)
	if err != nil {
		t.Fatalf(
			"failed to create message history request: %v",
			err,
		)
	}

	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: bobToken,
	})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf(
			"failed to request message history: %v",
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		t.Fatalf(
			"message history request returned status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var history []models.Message

	err = json.NewDecoder(resp.Body).Decode(&history)
	if err != nil {
		t.Fatalf(
			"failed to decode message history: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// 11. Verify the sent message exists in history
	// ---------------------------------------------------------

	found := false

	for _, message := range history {
		if message.ID == received.Data.ID &&
			message.UserID == aliceID &&
			message.ConversationID == conversationID &&
			message.Content == messageContent {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf(
			"message was not found in conversation history",
		)
	}

	t.Log("IT-FAIL-005: message verified in conversation history")

	// ---------------------------------------------------------
	// 12. Final result
	// ---------------------------------------------------------

	t.Log(
		"IT-FAIL-005 PASS: complete login -> session -> WebSocket -> send -> store -> deliver -> history flow verified",
	)
}