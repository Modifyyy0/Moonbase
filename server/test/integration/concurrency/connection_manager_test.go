package concurrency_test

import (
	"sync"
	"testing"

	moonws "Moonbase/src/websocket"

	gorilla "github.com/gorilla/websocket"

	"Moonbase/src/db"
	"Moonbase/test/testutil"
	"fmt"
	"time"
)

func TestIT_CON_003_ConnectionManagerConcurrency(t *testing.T) {
	manager := moonws.NewConnectionManager()

	// These clients are never removed.
	// They give us stable entries that must remain in the
	// registry while other goroutines modify the map.
	const stableClients = 10

	for i := 0; i < stableClients; i++ {
		client := &moonws.Client{
			UserID:  i + 1,
			Conn:    (*gorilla.Conn)(nil),
			Message: make(chan moonws.OutgoingMessage, 1),
			Done:    make(chan struct{}),
		}

		manager.AddClient(client)
	}

	var wg sync.WaitGroup

	// ---------------------------------------------------------
	// Concurrent AddClient()
	// ---------------------------------------------------------

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			userID := 1000 + i

			client := &moonws.Client{
				UserID:  userID,
				Conn:    (*gorilla.Conn)(nil),
				Message: make(chan moonws.OutgoingMessage, 1),
				Done:    make(chan struct{}),
			}

			manager.AddClient(client)
		}(i)
	}

	// ---------------------------------------------------------
	// Concurrent RemoveClient()
	// ---------------------------------------------------------

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			// These IDs overlap with the concurrently added
			// clients, so AddClient and RemoveClient can operate
			// on the same registry entries concurrently.
			manager.RemoveClient(1000 + i)
		}(i)
	}

	// ---------------------------------------------------------
	// Concurrent GetClient()
	// ---------------------------------------------------------

	for i := 0; i < 200; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			// Half the lookups target stable clients.
			// The others target clients being concurrently
			// added/removed.
			var userID int

			if i%2 == 0 {
				userID = (i % stableClients) + 1
			} else {
				userID = 1000 + (i % 100)
			}

			client, ok := manager.GetClient(userID)

			if i%2 == 0 {
				// Stable clients must never disappear.
				if !ok {
					t.Errorf(
						"stable client %d unexpectedly disappeared",
						userID,
					)
					return
				}

				if client == nil {
					t.Errorf(
						"GetClient(%d) returned nil client",
						userID,
					)
				}
			}
		}(i)
	}

	wg.Wait()

	// ---------------------------------------------------------
	// Final consistency check
	// ---------------------------------------------------------

	clients := manager.GetClients()

	stableFound := 0

	for _, client := range clients {
		if client == nil {
			t.Fatal("connection manager contains a nil client")
		}

		if client.UserID >= 1 && client.UserID <= stableClients {
			stableFound++
		}
	}

	if stableFound != stableClients {
		t.Fatalf(
			"expected all %d stable clients to remain, found %d",
			stableClients,
			stableFound,
		)
	}

	t.Logf(
		"IT-CON-003 PASS: concurrent AddClient, RemoveClient, and GetClient operations completed with a consistent connection registry",
	)
}

// ---------------------------------------------------------
// CON-004
// Database Concurrency Testing
// ---------------------------------------------------------

func TestIT_CON_004_DatabaseConcurrency(t *testing.T) {
	testutil.SetupDatabase(t)

	timestamp := time.Now().UnixNano()

	username := fmt.Sprintf(
		"test_CON-004_%d",
		timestamp,
	)

	// ---------------------------------------------------------
	// Create test user
	// ---------------------------------------------------------

	result, err := db.DB.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	userID64, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get user ID: %v", err)
	}

	userID := int(userID64)

	// ---------------------------------------------------------
	// Create test conversation
	// ---------------------------------------------------------

	result, err = db.DB.Exec(
		`INSERT INTO conversations
		 (name, conversation_type)
		 VALUES (?, 'group')`,
		"test_CON-004_conversation",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID64, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	conversationID := int(conversationID64)

	// Add user to conversation.
	_, err = db.DB.Exec(
		`INSERT INTO user_in_conversation
		 (user_id, conversation_id)
		 VALUES (?, ?)`,
		userID,
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to add user to conversation: %v", err)
	}

	// ---------------------------------------------------------
	// Concurrent database operations
	// ---------------------------------------------------------

	const messageCount = 100
	const readerCount = 50

	var wg sync.WaitGroup

	// Concurrent message creation.
	for i := 0; i < messageCount; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			content := fmt.Sprintf(
				"test-CON-004-message-%d",
				index,
			)

			_, err := db.DB.Exec(
				`INSERT INTO messages
				 (user_id, conversation_id, content)
				 VALUES (?, ?, ?)`,
				userID,
				conversationID,
				content,
			)

			if err != nil {
				t.Errorf(
					"message %d failed to insert: %v",
					index,
					err,
				)
			}
		}(i)
	}

	// Concurrent message reads.
	for i := 0; i < readerCount; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			var count int

			err := db.DB.QueryRow(
				`SELECT COUNT(*)
				 FROM messages
				 WHERE conversation_id = ?`,
				conversationID,
			).Scan(&count)

			if err != nil {
				t.Errorf(
					"reader %d failed: %v",
					index,
					err,
				)
			}
		}(i)
	}

	wg.Wait()

	// ---------------------------------------------------------
	// Verify all messages were persisted.
	// ---------------------------------------------------------

	var persistedCount int

	err = db.DB.QueryRow(
		`SELECT COUNT(*)
		 FROM messages
		 WHERE conversation_id = ?
		   AND content LIKE 'test-CON-004-message-%'`,
		conversationID,
	).Scan(&persistedCount)

	if err != nil {
		t.Fatalf(
			"failed to count persisted messages: %v",
			err,
		)
	}

	if persistedCount != messageCount {
		t.Fatalf(
			"message loss detected: got %d, want %d",
			persistedCount,
			messageCount,
		)
	}

	// ---------------------------------------------------------
	// Verify foreign-key consistency.
	// ---------------------------------------------------------

	var invalidReferences int

	err = db.DB.QueryRow(
		`SELECT COUNT(*)
		 FROM messages m
		 LEFT JOIN users u ON m.user_id = u.id
		 LEFT JOIN conversations c ON m.conversation_id = c.id
		 WHERE m.conversation_id = ?
		   AND (u.id IS NULL OR c.id IS NULL)`,
		conversationID,
	).Scan(&invalidReferences)

	if err != nil {
		t.Fatalf(
			"failed checking foreign-key consistency: %v",
			err,
		)
	}

	if invalidReferences != 0 {
		t.Fatalf(
			"found %d messages with invalid foreign-key references",
			invalidReferences,
		)
	}

	t.Logf(
		"IT-CON-004 PASS: %d concurrent writes and %d concurrent reads completed; all %d messages persisted with valid foreign keys",
		messageCount,
		readerCount,
		persistedCount,
	)
}

// ---------------------------------------------------------
// CON-005
// Transaction Testing
// ---------------------------------------------------------

func TestIT_CON_005_TransactionRollback(t *testing.T) {
	testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_CON-005_%d",
		time.Now().UnixNano(),
	)

	// ---------------------------------------------------------
	// Begin transaction
	// ---------------------------------------------------------

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	// Safety net.
	defer tx.Rollback()

	// ---------------------------------------------------------
	// Operation A
	// ---------------------------------------------------------

	_, err = tx.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf(
			"operation A failed unexpectedly: %v",
			err,
		)
	}

	// Confirm the user exists inside the transaction.
	var count int

	err = tx.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username,
	).Scan(&count)

	if err != nil {
		t.Fatalf(
			"failed checking transaction state: %v",
			err,
		)
	}

	if count != 1 {
		t.Fatalf(
			"expected user inside transaction, got %d rows",
			count,
		)
	}

	// ---------------------------------------------------------
	// Operation B
	//
	// Deliberately violate UNIQUE(username).
	// ---------------------------------------------------------

	_, err = tx.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)

	if err == nil {
		t.Fatal(
			"expected duplicate username operation to fail",
		)
	}

	t.Logf(
		"expected transaction failure: %v",
		err,
	)

	// ---------------------------------------------------------
	// Rollback
	// ---------------------------------------------------------

	if err := tx.Rollback(); err != nil {
		t.Fatalf(
			"failed to rollback transaction: %v",
			err,
		)
	}

	// ---------------------------------------------------------
	// Verify Operation A was also rolled back.
	// ---------------------------------------------------------

	err = db.DB.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username,
	).Scan(&count)

	if err != nil {
		t.Fatalf(
			"failed checking database after rollback: %v",
			err,
		)
	}

	if count != 0 {
		t.Fatalf(
			"partial database state remains after rollback: found %d user(s)",
			count,
		)
	}

	t.Log(
		"IT-CON-005 PASS: transaction failure rolled back all operations with no partial database state",
	)
}

// ---------------------------------------------------------
// CON-006
// Concurrent Database Conflict Testing
// ---------------------------------------------------------

func TestIT_CON_006_DatabaseConflicts(t *testing.T) {
	testutil.SetupDatabase(t)

	t.Run(
		"ConcurrentDuplicateUsernames",
		testConcurrentDuplicateUsernames,
	)

	t.Run(
		"ConcurrentMessagesSameConversation",
		testConcurrentMessagesSameConversation,
	)

	t.Run(
		"ConcurrentMembershipInsert",
		testConcurrentMembershipInsert,
	)
}

// ---------------------------------------------------------
// Conflict 1
//
// Multiple goroutines attempt to create the same username.
// UNIQUE(username) should allow exactly one.
// ---------------------------------------------------------

func testConcurrentDuplicateUsernames(t *testing.T) {
	username := fmt.Sprintf(
		"test_CON-006_duplicate_%d",
		time.Now().UnixNano(),
	)

	const attempts = 100

	var wg sync.WaitGroup

	var mutex sync.Mutex
	successCount := 0
	failureCount := 0

	for i := 0; i < attempts; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := db.DB.Exec(
				"INSERT INTO users (username) VALUES (?)",
				username,
			)

			mutex.Lock()
			defer mutex.Unlock()

			if err == nil {
				successCount++
			} else {
				failureCount++
			}
		}()
	}

	wg.Wait()

	// Exactly one INSERT should succeed.
	if successCount != 1 {
		t.Fatalf(
			"expected exactly 1 successful insert, got %d",
			successCount,
		)
	}

	if failureCount != attempts-1 {
		t.Fatalf(
			"expected %d failed inserts, got %d",
			attempts-1,
			failureCount,
		)
	}

	// Verify database contains exactly one row.
	var rowCount int

	err := db.DB.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username,
	).Scan(&rowCount)

	if err != nil {
		t.Fatalf(
			"failed checking username rows: %v",
			err,
		)
	}

	if rowCount != 1 {
		t.Fatalf(
			"expected exactly one user row, got %d",
			rowCount,
		)
	}

	t.Logf(
		"ConcurrentDuplicateUsernames PASS: %d concurrent attempts produced exactly one valid user",
		attempts,
	)
}

// ---------------------------------------------------------
// Conflict 2
//
// Multiple users send messages to the same conversation.
// ---------------------------------------------------------

func testConcurrentMessagesSameConversation(t *testing.T) {
	timestamp := time.Now().UnixNano()

	usernames := []string{
		fmt.Sprintf("test_CON-006_A_%d", timestamp),
		fmt.Sprintf("test_CON-006_B_%d", timestamp),
		fmt.Sprintf("test_CON-006_C_%d", timestamp),
	}

	userIDs := make([]int, 0, len(usernames))

	for _, username := range usernames {
		result, err := db.DB.Exec(
			"INSERT INTO users (username) VALUES (?)",
			username,
		)
		if err != nil {
			t.Fatalf("failed creating user: %v", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("failed getting user ID: %v", err)
		}

		userIDs = append(userIDs, int(id))
	}

	// Create conversation.
	result, err := db.DB.Exec(
		`INSERT INTO conversations
		 (name, conversation_type)
		 VALUES (?, 'group')`,
		"test_CON-006_shared_conversation",
	)
	if err != nil {
		t.Fatalf(
			"failed creating conversation: %v",
			err,
		)
	}

	conversationID64, err := result.LastInsertId()
	if err != nil {
		t.Fatalf(
			"failed getting conversation ID: %v",
			err,
		)
	}

	conversationID := int(conversationID64)

	// Add users.
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
				"failed adding user %d: %v",
				userID,
				err,
			)
		}
	}

	// ---------------------------------------------------------
	// Concurrent message creation.
	// ---------------------------------------------------------

	const messagesPerUser = 20

	var wg sync.WaitGroup

	for _, userID := range userIDs {
		wg.Add(1)

		go func(senderID int) {
			defer wg.Done()

			for i := 0; i < messagesPerUser; i++ {
				content := fmt.Sprintf(
					"CON-006-user-%d-message-%d",
					senderID,
					i,
				)

				_, err := db.DB.Exec(
					`INSERT INTO messages
					 (user_id, conversation_id, content)
					 VALUES (?, ?, ?)`,
					senderID,
					conversationID,
					content,
				)

				if err != nil {
					t.Errorf(
						"user %d failed to insert message %d: %v",
						senderID,
						i,
						err,
					)
				}
			}
		}(userID)
	}

	wg.Wait()

	expectedCount := len(userIDs) * messagesPerUser

	var actualCount int

	err = db.DB.QueryRow(
		`SELECT COUNT(*)
		 FROM messages
		 WHERE conversation_id = ?
		   AND content LIKE 'CON-006-user-%'`,
		conversationID,
	).Scan(&actualCount)

	if err != nil {
		t.Fatalf(
			"failed counting messages: %v",
			err,
		)
	}

	if actualCount != expectedCount {
		t.Fatalf(
			"message conflict caused data loss: got %d, want %d",
			actualCount,
			expectedCount,
		)
	}

	t.Logf(
		"ConcurrentMessagesSameConversation PASS: %d concurrent messages persisted",
		actualCount,
	)
}

// ---------------------------------------------------------
// Conflict 3
//
// Multiple goroutines attempt to create the same membership.
// PRIMARY KEY(user_id, conversation_id) should allow exactly one.
// ---------------------------------------------------------

func testConcurrentMembershipInsert(t *testing.T) {
	timestamp := time.Now().UnixNano()

	username := fmt.Sprintf(
		"test_CON-006_membership_%d",
		timestamp,
	)

	result, err := db.DB.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed creating user: %v", err)
	}

	userID64, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed getting user ID: %v", err)
	}

	userID := int(userID64)

	result, err = db.DB.Exec(
		`INSERT INTO conversations
		 (name, conversation_type)
		 VALUES (?, 'group')`,
		"test_CON-006_membership_conversation",
	)
	if err != nil {
		t.Fatalf(
			"failed creating conversation: %v",
			err,
		)
	}

	conversationID64, err := result.LastInsertId()
	if err != nil {
		t.Fatalf(
			"failed getting conversation ID: %v",
			err,
		)
	}

	conversationID := int(conversationID64)

	const attempts = 50

	var wg sync.WaitGroup

	successCount := 0
	var mutex sync.Mutex

	for i := 0; i < attempts; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := db.DB.Exec(
				`INSERT INTO user_in_conversation
				 (user_id, conversation_id)
				 VALUES (?, ?)`,
				userID,
				conversationID,
			)

			if err == nil {
				mutex.Lock()
				successCount++
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()

	// Exactly one membership should exist.
	if successCount != 1 {
		t.Fatalf(
			"expected exactly one successful membership insert, got %d",
			successCount,
		)
	}

	var membershipCount int

	err = db.DB.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ?
		   AND conversation_id = ?`,
		userID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed checking membership: %v",
			err,
		)
	}

	if membershipCount != 1 {
		t.Fatalf(
			"expected exactly one membership row, got %d",
			membershipCount,
		)
	}

	t.Logf(
		"ConcurrentMembershipInsert PASS: %d concurrent attempts produced exactly one valid membership",
		attempts,
	)
}