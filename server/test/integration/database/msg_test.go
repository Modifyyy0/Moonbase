package database_test

import (
	"testing"
	"database/sql"

	"Moonbase/src/models"
	"Moonbase/test/testutil"
)

func TestIT_DB_011_CreateMessage(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db011_user")
	conversationName := testutil.TestUsername("db011_conversation")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-011 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-011 failed: could not find test user %q: %v",
			username,
			err,
		)
	}

	// Clean up the test user.
	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username = ?",
			username,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-011 cleanup failed: could not delete test user %q: %v",
				username,
				err,
			)
		}
	})

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-011 failed: could not create test conversation %q: %v",
			conversationName,
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-011 failed: could not find test conversation %q: %v",
			conversationName,
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-011 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	// Clean up the test conversation.
	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-011 cleanup failed: could not delete test conversation: %v",
				err,
			)
		}
	})

	// Add the user to the conversation.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-011 failed: could not add user to conversation: %v",
			err,
		)
	}

	content := "Hello from IT-DB-011"

	// Create the message.
	message, err := models.CreateMessage(
		user.ID,
		conversationID,
		content,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-011 failed: CreateMessage() returned an error: %v",
			err,
		)
	}

	if message == nil {
		t.Fatalf(
			"IT-DB-011 failed: CreateMessage() returned nil message",
		)
	}

	// Clean up the message explicitly.
	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM messages WHERE id = ?",
			message.ID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-011 cleanup failed: could not delete test message: %v",
				err,
			)
		}
	})

	// Verify the returned message.
	if message.ID <= 0 {
		t.Fatalf(
			"IT-DB-011 failed: expected valid message ID, got %d",
			message.ID,
		)
	}

	if message.UserID != user.ID {
		t.Fatalf(
			"IT-DB-011 failed: expected user ID %d, got %d",
			user.ID,
			message.UserID,
		)
	}

	if message.ConversationID != conversationID {
		t.Fatalf(
			"IT-DB-011 failed: expected conversation ID %d, got %d",
			conversationID,
			message.ConversationID,
		)
	}

	if message.Content != content {
		t.Fatalf(
			"IT-DB-011 failed: expected content %q, got %q",
			content,
			message.Content,
		)
	}

	if message.SentAt.IsZero() {
		t.Fatalf(
			"IT-DB-011 failed: expected sent_at to be populated",
		)
	}

	t.Logf(
		"IT-DB-011 PASS: message %d was created successfully in conversation %d",
		message.ID,
		conversationID,
	)
}

func TestIT_DB_012_FindMessageByID(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db012_user")
	conversationName := testutil.TestUsername("db012_conversation")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-012 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-012 failed: could not find test user %q: %v",
			username,
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username = ?",
			username,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-012 cleanup failed: could not delete test user %q: %v",
				username,
				err,
			)
		}
	})

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-012 failed: could not create test conversation %q: %v",
			conversationName,
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-012 failed: could not find test conversation %q: %v",
			conversationName,
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-012 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-012 cleanup failed: could not delete test conversation: %v",
				err,
			)
		}
	})

	// Add user to conversation.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-012 failed: could not add user to conversation: %v",
			err,
		)
	}

	// Create test message.
	content := "Hello from IT-DB-012"

	message, err := models.CreateMessage(
		user.ID,
		conversationID,
		content,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-012 failed: CreateMessage() returned an error: %v",
			err,
		)
	}

	if message == nil {
		t.Fatalf(
			"IT-DB-012 failed: CreateMessage() returned nil message",
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM messages WHERE id = ?",
			message.ID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-012 cleanup failed: could not delete test message: %v",
				err,
			)
		}
	})

	// Retrieve the message by ID.
	foundMessage, err := models.FindMessageByID(message.ID)
	if err != nil {
		t.Fatalf(
			"IT-DB-012 failed: FindMessageByID() returned an error: %v",
			err,
		)
	}

	if foundMessage == nil {
		t.Fatalf(
			"IT-DB-012 failed: FindMessageByID() returned nil message",
		)
	}

	// Verify retrieved data.
	if foundMessage.ID != message.ID {
		t.Fatalf(
			"IT-DB-012 failed: expected message ID %d, got %d",
			message.ID,
			foundMessage.ID,
		)
	}

	if foundMessage.UserID != user.ID {
		t.Fatalf(
			"IT-DB-012 failed: expected user ID %d, got %d",
			user.ID,
			foundMessage.UserID,
		)
	}

	if foundMessage.ConversationID != conversationID {
		t.Fatalf(
			"IT-DB-012 failed: expected conversation ID %d, got %d",
			conversationID,
			foundMessage.ConversationID,
		)
	}

	if foundMessage.Content != content {
		t.Fatalf(
			"IT-DB-012 failed: expected content %q, got %q",
			content,
			foundMessage.Content,
		)
	}

	if foundMessage.SentAt.IsZero() {
		t.Fatalf(
			"IT-DB-012 failed: expected sent_at to be populated",
		)
	}

	t.Logf(
		"IT-DB-012 PASS: message %d was successfully retrieved",
		foundMessage.ID,
	)
}

func TestIT_DB_013_FindMessagesByConversationID(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db013_user")
	conversationName := testutil.TestUsername("db013_conversation")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-013 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-013 failed: could not find test user %q: %v",
			username,
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username = ?",
			username,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-013 cleanup failed: could not delete test user %q: %v",
				username,
				err,
			)
		}
	})

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-013 failed: could not create test conversation %q: %v",
			conversationName,
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-013 failed: could not find test conversation %q: %v",
			conversationName,
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-013 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-013 cleanup failed: could not delete test conversation: %v",
				err,
			)
		}
	})

	// Add user to conversation.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-013 failed: could not add user to conversation: %v",
			err,
		)
	}

	// Create two messages.
	content1 := "First message from IT-DB-013"
	content2 := "Second message from IT-DB-013"

	message1, err := models.CreateMessage(
		user.ID,
		conversationID,
		content1,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-013 failed: could not create first message: %v",
			err,
		)
	}

	message2, err := models.CreateMessage(
		user.ID,
		conversationID,
		content2,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-013 failed: could not create second message: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM messages WHERE id IN (?, ?)",
			message1.ID,
			message2.ID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-013 cleanup failed: could not delete test messages: %v",
				err,
			)
		}
	})

	// Find all messages in the conversation.
	messages, err := models.FindMessagesByConversationID(conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-013 failed: FindMessagesByConversationID() returned an error: %v",
			err,
		)
	}

	if len(messages) != 2 {
		t.Fatalf(
			"IT-DB-013 failed: expected 2 messages, found %d",
			len(messages),
		)
	}

	// Verify first message.
	if messages[0].ID != message1.ID {
		t.Fatalf(
			"IT-DB-013 failed: expected first message ID %d, got %d",
			message1.ID,
			messages[0].ID,
		)
	}

	if messages[0].UserID != user.ID {
		t.Fatalf(
			"IT-DB-013 failed: expected first message user ID %d, got %d",
			user.ID,
			messages[0].UserID,
		)
	}

	if messages[0].Username != username {
		t.Fatalf(
			"IT-DB-013 failed: expected first message username %q, got %q",
			username,
			messages[0].Username,
		)
	}

	if messages[0].ConversationID != conversationID {
		t.Fatalf(
			"IT-DB-013 failed: expected first message conversation ID %d, got %d",
			conversationID,
			messages[0].ConversationID,
		)
	}

	if messages[0].Content != content1 {
		t.Fatalf(
			"IT-DB-013 failed: expected first message content %q, got %q",
			content1,
			messages[0].Content,
		)
	}

	// Verify second message.
	if messages[1].ID != message2.ID {
		t.Fatalf(
			"IT-DB-013 failed: expected second message ID %d, got %d",
			message2.ID,
			messages[1].ID,
		)
	}

	if messages[1].UserID != user.ID {
		t.Fatalf(
			"IT-DB-013 failed: expected second message user ID %d, got %d",
			user.ID,
			messages[1].UserID,
		)
	}

	if messages[1].Username != username {
		t.Fatalf(
			"IT-DB-013 failed: expected second message username %q, got %q",
			username,
			messages[1].Username,
		)
	}

	if messages[1].ConversationID != conversationID {
		t.Fatalf(
			"IT-DB-013 failed: expected second message conversation ID %d, got %d",
			conversationID,
			messages[1].ConversationID,
		)
	}

	if messages[1].Content != content2 {
		t.Fatalf(
			"IT-DB-013 failed: expected second message content %q, got %q",
			content2,
			messages[1].Content,
		)
	}

	t.Logf(
		"IT-DB-013 PASS: retrieved %d messages from conversation %d",
		len(messages),
		conversationID,
	)
}

func TestIT_DB_014_FindMessagesByUserID(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username1 := testutil.TestUsername("db014_user1")
	username2 := testutil.TestUsername("db014_user2")
	conversationName := testutil.TestUsername("db014_conversation")

	// Create test users.
	err := models.CreateUser(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not create first test user %q: %v",
			username1,
			err,
		)
	}

	err = models.CreateUser(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not create second test user %q: %v",
			username2,
			err,
		)
	}

	user1, err := models.FindUserByName(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not find first test user: %v",
			err,
		)
	}

	user2, err := models.FindUserByName(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not find second test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username IN (?, ?)",
			username1,
			username2,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-014 cleanup failed: %v",
				err,
			)
		}
	})

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not create test conversation: %v",
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not find test conversation: %v",
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-014 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-014 cleanup failed: %v",
				err,
			)
		}
	})

	// Add both users to the conversation.
	err = models.AddUserToConvo(user1.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not add first user to conversation: %v",
			err,
		)
	}

	err = models.AddUserToConvo(user2.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not add second user to conversation: %v",
			err,
		)
	}

	// Create messages from both users.
	content1 := "Message from user 1"
	content2 := "Message from user 2"

	message1, err := models.CreateMessage(
		user1.ID,
		conversationID,
		content1,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not create first message: %v",
			err,
		)
	}

	message2, err := models.CreateMessage(
		user2.ID,
		conversationID,
		content2,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: could not create second message: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM messages WHERE id IN (?, ?)",
			message1.ID,
			message2.ID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-014 cleanup failed: %v",
				err,
			)
		}
	})

	// Find messages belonging to user 1.
	messages, err := models.FindMessagesByUserID(user1.ID)
	if err != nil {
		t.Fatalf(
			"IT-DB-014 failed: FindMessagesByUserID() returned an error: %v",
			err,
		)
	}

	// Only user 1's message should be returned.
	if len(messages) != 1 {
		t.Fatalf(
			"IT-DB-014 failed: expected 1 message for user 1, found %d",
			len(messages),
		)
	}

	if messages[0].ID != message1.ID {
		t.Fatalf(
			"IT-DB-014 failed: expected message ID %d, got %d",
			message1.ID,
			messages[0].ID,
		)
	}

	if messages[0].UserID != user1.ID {
		t.Fatalf(
			"IT-DB-014 failed: expected user ID %d, got %d",
			user1.ID,
			messages[0].UserID,
		)
	}

	if messages[0].ConversationID != conversationID {
		t.Fatalf(
			"IT-DB-014 failed: expected conversation ID %d, got %d",
			conversationID,
			messages[0].ConversationID,
		)
	}

	if messages[0].Content != content1 {
		t.Fatalf(
			"IT-DB-014 failed: expected content %q, got %q",
			content1,
			messages[0].Content,
		)
	}

	// Make sure user 2's message was not returned.
	if messages[0].ID == message2.ID {
		t.Fatalf(
			"IT-DB-014 failed: returned another user's message",
		)
	}

	t.Logf(
		"IT-DB-014 PASS: retrieved only user %d's messages",
		user1.ID,
	)
}

func TestIT_DB_015_DeleteMessage(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db015_user")
	conversationName := testutil.TestUsername("db015_conversation")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-015 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-015 failed: could not find test user %q: %v",
			username,
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username = ?",
			username,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-015 cleanup failed: could not delete test user: %v",
				err,
			)
		}
	})

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-015 failed: could not create test conversation: %v",
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-015 failed: could not find test conversation: %v",
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-015 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-015 cleanup failed: could not delete test conversation: %v",
				err,
			)
		}
	})

	// Add user to conversation.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-015 failed: could not add user to conversation: %v",
			err,
		)
	}

	// Create test message.
	message, err := models.CreateMessage(
		user.ID,
		conversationID,
		"Message to be deleted",
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-015 failed: could not create test message: %v",
			err,
		)
	}

	// Delete the message.
	err = message.DeleteMessage()
	if err != nil {
		t.Fatalf(
			"IT-DB-015 failed: DeleteMessage() returned an error: %v",
			err,
		)
	}

	// Verify the message no longer exists.
	_, err = models.FindMessageByID(message.ID)

	if err == nil {
		t.Fatalf(
			"IT-DB-015 failed: expected deleted message %d to no longer exist",
			message.ID,
		)
	}

	if err != sql.ErrNoRows {
		t.Fatalf(
			"IT-DB-015 failed: expected sql.ErrNoRows after deletion, got: %v",
			err,
		)
	}

	t.Logf(
		"IT-DB-015 PASS: message %d was successfully deleted",
		message.ID,
	)
}

func TestIT_DB_016_CreateMessageReceipts(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username1 := testutil.TestUsername("db016_user1")
	username2 := testutil.TestUsername("db016_user2")
	username3 := testutil.TestUsername("db016_user3")
	conversationName := testutil.TestUsername("db016_conversation")

	// Create three users.
	for _, username := range []string{
		username1,
		username2,
		username3,
	} {
		err := models.CreateUser(username)
		if err != nil {
			t.Fatalf(
				"IT-DB-016 failed: could not create user %q: %v",
				username,
				err,
			)
		}
	}

	user1, err := models.FindUserByName(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not find user 1: %v",
			err,
		)
	}

	user2, err := models.FindUserByName(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not find user 2: %v",
			err,
		)
	}

	user3, err := models.FindUserByName(username3)
	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not find user 3: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username IN (?, ?, ?)",
			username1,
			username2,
			username3,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-016 cleanup failed: could not delete test users: %v",
				err,
			)
		}
	})

	// Create conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not create conversation: %v",
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not find conversation: %v",
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-016 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-016 cleanup failed: could not delete conversation: %v",
				err,
			)
		}
	})

	// Add all three users to the conversation.
	for _, userID := range []int{
		user1.ID,
		user2.ID,
		user3.ID,
	} {
		err := models.AddUserToConvo(userID, conversationID)
		if err != nil {
			t.Fatalf(
				"IT-DB-016 failed: could not add user %d to conversation: %v",
				userID,
				err,
			)
		}
	}

	// User 1 sends the message.
	message, err := models.CreateMessage(
		user1.ID,
		conversationID,
		"Message requiring receipts",
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not create message: %v",
			err,
		)
	}

	// Create receipts for everyone except sender.
	err = models.CreateMessageReceipts(
		message.ID,
		conversationID,
		user1.ID,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: CreateMessageReceipts() returned an error: %v",
			err,
		)
	}

	// Check number of receipts.
	var receiptCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM message_receipts WHERE message_id = ?",
		message.ID,
	).Scan(&receiptCount)

	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not count receipts: %v",
			err,
		)
	}

	if receiptCount != 2 {
		t.Fatalf(
			"IT-DB-016 failed: expected 2 receipts, found %d",
			receiptCount,
		)
	}

	// Verify user 2 has a receipt.
	var exists int

	err = database.QueryRow(`
		SELECT COUNT(*)
		FROM message_receipts
		WHERE message_id = ?
		AND user_id = ?
	`, message.ID, user2.ID).Scan(&exists)

	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not check user 2 receipt: %v",
			err,
		)
	}

	if exists != 1 {
		t.Fatalf(
			"IT-DB-016 failed: expected receipt for user 2",
		)
	}

	// Verify user 3 has a receipt.
	err = database.QueryRow(`
		SELECT COUNT(*)
		FROM message_receipts
		WHERE message_id = ?
		AND user_id = ?
	`, message.ID, user3.ID).Scan(&exists)

	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not check user 3 receipt: %v",
			err,
		)
	}

	if exists != 1 {
		t.Fatalf(
			"IT-DB-016 failed: expected receipt for user 3",
		)
	}

	// Verify sender does NOT have a receipt.
	err = database.QueryRow(`
		SELECT COUNT(*)
		FROM message_receipts
		WHERE message_id = ?
		AND user_id = ?
	`, message.ID, user1.ID).Scan(&exists)

	if err != nil {
		t.Fatalf(
			"IT-DB-016 failed: could not check sender receipt: %v",
			err,
		)
	}

	if exists != 0 {
		t.Fatalf(
			"IT-DB-016 failed: sender should not have a receipt",
		)
	}

	t.Logf(
		"IT-DB-016 PASS: created receipts for 2 recipients and excluded sender",
	)
}

func TestIT_DB_017_MarkMessageDelivered(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username1 := testutil.TestUsername("db017_user1")
	username2 := testutil.TestUsername("db017_user2")
	conversationName := testutil.TestUsername("db017_conversation")

	// Create users.
	err := models.CreateUser(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not create user 1: %v",
			err,
		)
	}

	err = models.CreateUser(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not create user 2: %v",
			err,
		)
	}

	user1, err := models.FindUserByName(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not find user 1: %v",
			err,
		)
	}

	user2, err := models.FindUserByName(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not find user 2: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username IN (?, ?)",
			username1,
			username2,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-017 cleanup failed: could not delete users: %v",
				err,
			)
		}
	})

	// Create conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not create conversation: %v",
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not find conversation: %v",
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-017 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-017 cleanup failed: could not delete conversation: %v",
				err,
			)
		}
	})

	// Add both users to conversation.
	err = models.AddUserToConvo(user1.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not add user 1 to conversation: %v",
			err,
		)
	}

	err = models.AddUserToConvo(user2.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not add user 2 to conversation: %v",
			err,
		)
	}

	// Create message from user 1.
	message, err := models.CreateMessage(
		user1.ID,
		conversationID,
		"Message for delivery test",
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not create message: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM messages WHERE id = ?",
			message.ID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-017 cleanup failed: could not delete message: %v",
				err,
			)
		}
	})

	// Mark message as delivered to user 2.
	receipt, err := models.MarkMessageDelivered(
		message.ID,
		user2.ID,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: MarkMessageDelivered() returned an error: %v",
			err,
		)
	}

	if receipt == nil {
		t.Fatalf(
			"IT-DB-017 failed: MarkMessageDelivered() returned nil receipt",
		)
	}

	// Verify returned receipt.
	if receipt.MessageID != message.ID {
		t.Fatalf(
			"IT-DB-017 failed: expected message ID %d, got %d",
			message.ID,
			receipt.MessageID,
		)
	}

	if receipt.ConversationID != conversationID {
		t.Fatalf(
			"IT-DB-017 failed: expected conversation ID %d, got %d",
			conversationID,
			receipt.ConversationID,
		)
	}

	if receipt.UserID != user2.ID {
		t.Fatalf(
			"IT-DB-017 failed: expected user ID %d, got %d",
			user2.ID,
			receipt.UserID,
		)
	}

	if receipt.Status != "delivered" {
		t.Fatalf(
			"IT-DB-017 failed: expected status %q, got %q",
			"delivered",
			receipt.Status,
		)
	}

	// Verify the database receipt.
	var deliveredAt sql.NullTime

	err = database.QueryRow(`
		SELECT delivered_at
		FROM message_receipts
		WHERE message_id = ?
		AND user_id = ?
	`, message.ID, user2.ID).Scan(&deliveredAt)

	if err != nil {
		t.Fatalf(
			"IT-DB-017 failed: could not retrieve delivery receipt: %v",
			err,
		)
	}

	if !deliveredAt.Valid {
		t.Fatalf(
			"IT-DB-017 failed: expected delivered_at to be populated",
		)
	}

	t.Logf(
		"IT-DB-017 PASS: message %d was marked delivered to user %d",
		message.ID,
		user2.ID,
	)
}

func TestIT_DB_018_MarkMessageRead(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username1 := testutil.TestUsername("db018_user1")
	username2 := testutil.TestUsername("db018_user2")
	conversationName := testutil.TestUsername("db018_conversation")

	// Create users.
	err := models.CreateUser(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not create user 1: %v",
			err,
		)
	}

	err = models.CreateUser(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not create user 2: %v",
			err,
		)
	}

	user1, err := models.FindUserByName(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not find user 1: %v",
			err,
		)
	}

	user2, err := models.FindUserByName(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not find user 2: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username IN (?, ?)",
			username1,
			username2,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-018 cleanup failed: could not delete users: %v",
				err,
			)
		}
	})

	// Create conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not create conversation: %v",
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not find conversation: %v",
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-018 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-018 cleanup failed: could not delete conversation: %v",
				err,
			)
		}
	})

	// Add both users to conversation.
	err = models.AddUserToConvo(user1.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not add user 1 to conversation: %v",
			err,
		)
	}

	err = models.AddUserToConvo(user2.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not add user 2 to conversation: %v",
			err,
		)
	}

	// Create message from user 1.
	message, err := models.CreateMessage(
		user1.ID,
		conversationID,
		"Message for read test",
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not create message: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM messages WHERE id = ?",
			message.ID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-018 cleanup failed: could not delete message: %v",
				err,
			)
		}
	})

	// Mark message as read by user 2.
	receipt, err := models.MarkMessageRead(
		message.ID,
		user2.ID,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: MarkMessageRead() returned an error: %v",
			err,
		)
	}

	if receipt == nil {
		t.Fatalf(
			"IT-DB-018 failed: MarkMessageRead() returned nil receipt",
		)
	}

	// Verify returned receipt.
	if receipt.MessageID != message.ID {
		t.Fatalf(
			"IT-DB-018 failed: expected message ID %d, got %d",
			message.ID,
			receipt.MessageID,
		)
	}

	if receipt.ConversationID != conversationID {
		t.Fatalf(
			"IT-DB-018 failed: expected conversation ID %d, got %d",
			conversationID,
			receipt.ConversationID,
		)
	}

	if receipt.UserID != user2.ID {
		t.Fatalf(
			"IT-DB-018 failed: expected user ID %d, got %d",
			user2.ID,
			receipt.UserID,
		)
	}

	if receipt.Status != "read" {
		t.Fatalf(
			"IT-DB-018 failed: expected status %q, got %q",
			"read",
			receipt.Status,
		)
	}

	// Verify both timestamps in the database.
	var deliveredAt sql.NullTime
	var readAt sql.NullTime

	err = database.QueryRow(`
		SELECT delivered_at, read_at
		FROM message_receipts
		WHERE message_id = ?
		AND user_id = ?
	`, message.ID, user2.ID).Scan(
		&deliveredAt,
		&readAt,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-018 failed: could not retrieve read receipt: %v",
			err,
		)
	}

	if !deliveredAt.Valid {
		t.Fatalf(
			"IT-DB-018 failed: expected delivered_at to be populated",
		)
	}

	if !readAt.Valid {
		t.Fatalf(
			"IT-DB-018 failed: expected read_at to be populated",
		)
	}

	t.Logf(
		"IT-DB-018 PASS: message %d was marked read by user %d",
		message.ID,
		user2.ID,
	)
}

func TestIT_DB_019_IsConversationMember(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username1 := testutil.TestUsername("db019_user1")
	username2 := testutil.TestUsername("db019_user2")
	conversationName := testutil.TestUsername("db019_conversation")

	// Create users.
	err := models.CreateUser(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: could not create user 1: %v",
			err,
		)
	}

	err = models.CreateUser(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: could not create user 2: %v",
			err,
		)
	}

	user1, err := models.FindUserByName(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: could not find user 1: %v",
			err,
		)
	}

	user2, err := models.FindUserByName(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: could not find user 2: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username IN (?, ?)",
			username1,
			username2,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-019 cleanup failed: could not delete users: %v",
				err,
			)
		}
	})

	// Create conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: could not create conversation: %v",
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: could not find conversation: %v",
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-019 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-019 cleanup failed: could not delete conversation: %v",
				err,
			)
		}
	})

	// Add only user 1 to the conversation.
	err = models.AddUserToConvo(user1.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: could not add user 1 to conversation: %v",
			err,
		)
	}

	// User 1 should be a member.
	isMember, err := models.IsConversationMember(
		user1.ID,
		conversationID,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: IsConversationMember() returned an error for member: %v",
			err,
		)
	}

	if !isMember {
		t.Fatalf(
			"IT-DB-019 failed: expected user 1 to be a member",
		)
	}

	// User 2 should not be a member.
	isMember, err = models.IsConversationMember(
		user2.ID,
		conversationID,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-019 failed: IsConversationMember() returned an error for non-member: %v",
			err,
		)
	}

	if isMember {
		t.Fatalf(
			"IT-DB-019 failed: expected user 2 to NOT be a member",
		)
	}

	t.Logf(
		"IT-DB-019 PASS: membership correctly identified for users %d and %d",
		user1.ID,
		user2.ID,
	)
}

func TestIT_DB_020_UpdateContent(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db020_user")
	conversationName := testutil.TestUsername("db020_conversation")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-020 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-020 failed: could not find test user %q: %v",
			username,
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username = ?",
			username,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-020 cleanup failed: could not delete user: %v",
				err,
			)
		}
	})

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-020 failed: could not create conversation: %v",
			err,
		)
	}

	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-020 failed: could not find conversation: %v",
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-020 failed: expected 1 conversation, found %d",
			len(conversations),
		)
	}

	conversationID := conversations[0].ID

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-020 cleanup failed: could not delete conversation: %v",
				err,
			)
		}
	})

	// Add user to conversation.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-020 failed: could not add user to conversation: %v",
			err,
		)
	}

	// Create original message.
	originalContent := "Original message"

	message, err := models.CreateMessage(
		user.ID,
		conversationID,
		originalContent,
	)

	if err != nil {
		t.Fatalf(
			"IT-DB-020 failed: could not create message: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM messages WHERE id = ?",
			message.ID,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-020 cleanup failed: could not delete message: %v",
				err,
			)
		}
	})

	// Update message content.
	updatedContent := "Updated message"

	err = message.UpdateContent(updatedContent)
	if err != nil {
		t.Fatalf(
			"IT-DB-020 failed: UpdateContent() returned an error: %v",
			err,
		)
	}

	// Retrieve message again.
	updatedMessage, err := models.FindMessageByID(message.ID)
	if err != nil {
		t.Fatalf(
			"IT-DB-020 failed: could not retrieve updated message: %v",
			err,
		)
	}

	if updatedMessage == nil {
		t.Fatalf(
			"IT-DB-020 failed: retrieved message is nil",
		)
	}

	// Verify content changed.
	if updatedMessage.Content != updatedContent {
		t.Fatalf(
			"IT-DB-020 failed: expected content %q, got %q",
			updatedContent,
			updatedMessage.Content,
		)
	}

	// Verify the message identity did not change.
	if updatedMessage.ID != message.ID {
		t.Fatalf(
			"IT-DB-020 failed: expected message ID %d, got %d",
			message.ID,
			updatedMessage.ID,
		)
	}

	if updatedMessage.UserID != user.ID {
		t.Fatalf(
			"IT-DB-020 failed: expected user ID %d, got %d",
			user.ID,
			updatedMessage.UserID,
		)
	}

	if updatedMessage.ConversationID != conversationID {
		t.Fatalf(
			"IT-DB-020 failed: expected conversation ID %d, got %d",
			conversationID,
			updatedMessage.ConversationID,
		)
	}

	t.Logf(
		"IT-DB-020 PASS: message %d content was successfully updated",
		message.ID,
	)
}