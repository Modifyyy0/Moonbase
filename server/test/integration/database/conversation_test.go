package database_test

import (
	"testing"

	"Moonbase/src/models"
	"Moonbase/test/testutil"
)

func TestIT_DB_004_CreateConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	name := testutil.TestUsername("db004_conversation")

	// Make sure the test starts clean.
	_, err := database.Exec(
		"DELETE FROM conversations WHERE name = ?",
		name,
	)
	if err != nil {
		t.Fatalf(
			"IT-DB-004 setup failed: could not clean up test conversation: %v",
			err,
		)
	}

	// Create the conversation.
	err = models.CreateConvo(name)

	if err != nil {
		t.Fatalf(
			"IT-DB-004 failed: CreateConvo(%q) returned an error: %v",
			name,
			err,
		)
	}

	// Always clean up the test conversation.
	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE name = ?",
			name,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-004 cleanup failed: could not delete test conversation %q: %v",
				name,
				err,
			)
		}
	})

	// Verify that exactly one conversation was created.
	var count int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations WHERE name = ?",
		name,
	).Scan(&count)

	if err != nil {
		t.Fatalf(
			"IT-DB-004 failed: could not verify conversation count: %v",
			err,
		)
	}

	if count != 1 {
		t.Fatalf(
			"IT-DB-004 failed: expected exactly 1 conversation, found %d",
			count,
		)
	}

	// Verify the actual conversation data.
	var id int
	var conversationType string

	err = database.QueryRow(
		`SELECT id, conversation_type
		 FROM conversations
		 WHERE name = ?`,
		name,
	).Scan(&id, &conversationType)

	if err != nil {
		t.Fatalf(
			"IT-DB-004 failed: could not retrieve created conversation: %v",
			err,
		)
	}

	if id <= 0 {
		t.Fatalf(
			"IT-DB-004 failed: expected valid conversation ID, got %d",
			id,
		)
	}

	if conversationType != "group" {
		t.Fatalf(
			"IT-DB-004 failed: expected conversation type %q, got %q",
			"group",
			conversationType,
		)
	}

	t.Logf(
		"IT-DB-004 PASS: conversation %q was created successfully with ID %d",
		name,
		id,
	)
}

func TestIT_DB_005_FindConversationByID(t *testing.T) {
	database := testutil.SetupDatabase(t)

	name := testutil.TestUsername("db005_conversation")

	// Make sure the test starts clean.
	_, err := database.Exec(
		"DELETE FROM conversations WHERE name = ?",
		name,
	)
	if err != nil {
		t.Fatalf(
			"IT-DB-005 setup failed: could not clean up test conversation",
		)
	}

	// Create the conversation.
	err = models.CreateConvo(name)
	if err != nil {
		t.Fatalf(
			"IT-DB-005 failed: CreateConvo(%q) returned an error: %v",
			name,
			err,
		)
	}

	// Always clean up the test conversation.
	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE name = ?",
			name,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-005 cleanup failed: could not delete test conversation %q: %v",
				name,
				err,
			)
		}
	})

	// Get the ID of the conversation we just created.
	var conversationID int

	err = database.QueryRow(
		"SELECT id FROM conversations WHERE name = ?",
		name,
	).Scan(&conversationID)

	if err != nil {
		t.Fatalf(
			"IT-DB-005 failed: could not retrieve created conversation ID: %v",
			err,
		)
	}

	// Find the conversation using the model function.
	conversation, err := models.FindConvoById(conversationID)

	if err != nil {
		t.Fatalf(
			"IT-DB-005 failed: FindConvoById(%d) returned an error: %v",
			conversationID,
			err,
		)
	}

	if conversation == nil {
		t.Fatalf(
			"IT-DB-005 failed: FindConvoById(%d) returned nil conversation",
			conversationID,
		)
	}

	// Verify the returned data.
	if conversation.ID != conversationID {
		t.Fatalf(
			"IT-DB-005 failed: expected ID %d, got %d",
			conversationID,
			conversation.ID,
		)
	}

	if conversation.Name != name {
		t.Fatalf(
			"IT-DB-005 failed: expected name %q, got %q",
			name,
			conversation.Name,
		)
	}

	if conversation.ConversationType != "group" {
		t.Fatalf(
			"IT-DB-005 failed: expected conversation type %q, got %q",
			"group",
			conversation.ConversationType,
		)
	}

	if conversation.CreatedAt.IsZero() {
		t.Fatalf(
			"IT-DB-005 failed: expected created_at to be populated",
		)
	}

	t.Logf(
		"IT-DB-005 PASS: conversation ID %d was retrieved successfully",
		conversationID,
	)
}

func TestIT_DB_006_AddUserToConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db006_user")
	conversationName := testutil.TestUsername("db006_conversation")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-006 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	// Clean up user.
	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM users WHERE username = ?",
			username,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-006 cleanup failed: could not delete test user %q: %v",
				username,
				err,
			)
		}
	})

	// Get the user's ID.
	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-006 failed: could not find test user %q: %v",
			username,
			err,
		)
	}

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-006 failed: could not create test conversation %q: %v",
			conversationName,
			err,
		)
	}

	// Clean up conversation.
	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE name = ?",
			conversationName,
		)

		if err != nil {
			t.Errorf(
				"IT-DB-006 cleanup failed: could not delete test conversation %q: %v",
				conversationName,
				err,
			)
		}
	})

	// Get the conversation's ID.
	conversation, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-006 failed: could not find test conversation %q: %v",
			conversationName,
			err,
		)
	}

	if len(conversation) != 1 {
		t.Fatalf(
			"IT-DB-006 failed: expected 1 conversation, found %d",
			len(conversation),
		)
	}

	conversationID := conversation[0].ID

	// Add the user to the conversation.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-006 failed: AddUserToConvo(%d, %d) returned an error: %v",
			user.ID,
			conversationID,
			err,
		)
	}

	// Verify the membership exists in the database.
	var count int

	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ?
		   AND conversation_id = ?`,
		user.ID,
		conversationID,
	).Scan(&count)

	if err != nil {
		t.Fatalf(
			"IT-DB-006 failed: could not verify conversation membership: %v",
			err,
		)
	}

	if count != 1 {
		t.Fatalf(
			"IT-DB-006 failed: expected 1 membership record, found %d",
			count,
		)
	}

	t.Logf(
		"IT-DB-006 PASS: user %d was added to conversation %d successfully",
		user.ID,
		conversationID,
	)
}

func TestIT_DB_007_DuplicateUserInConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db007_user")
	conversationName := testutil.TestUsername("db007_conversation")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-007 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	// Get the user's ID.
	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-007 failed: could not find test user %q: %v",
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
				"IT-DB-007 cleanup failed: could not delete test user %q: %v",
				username,
				err,
			)
		}
	})

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-007 failed: could not create test conversation %q: %v",
			conversationName,
			err,
		)
	}

	// Get the conversation ID.
	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-007 failed: could not find test conversation %q: %v",
			conversationName,
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-007 failed: expected 1 conversation, found %d",
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
				"IT-DB-007 cleanup failed: could not delete test conversation: %v",
				err,
			)
		}
	})

	// First membership should succeed.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-007 failed: first AddUserToConvo(%d, %d) returned an error: %v",
			user.ID,
			conversationID,
			err,
		)
	}

	// Second identical membership should fail.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err == nil {
		t.Fatalf(
			"IT-DB-007 failed: expected duplicate membership to return an error",
		)
	}

	// Verify that only one membership exists.
	var count int

	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ?
		   AND conversation_id = ?`,
		user.ID,
		conversationID,
	).Scan(&count)

	if err != nil {
		t.Fatalf(
			"IT-DB-007 failed: could not verify membership count: %v",
			err,
		)
	}

	if count != 1 {
		t.Fatalf(
			"IT-DB-007 failed: expected exactly 1 membership, found %d",
			count,
		)
	}

	t.Logf(
		"IT-DB-007 PASS: duplicate membership for user %d in conversation %d was rejected",
		user.ID,
		conversationID,
	)
}

func TestIT_DB_008_RemoveUserFromConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db008_user")
	conversationName := testutil.TestUsername("db008_conversation")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-008 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	// Get the user's ID.
	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-008 failed: could not find test user %q: %v",
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
				"IT-DB-008 cleanup failed: could not delete test user %q: %v",
				username,
				err,
			)
		}
	})

	// Create test conversation.
	err = models.CreateConvo(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-008 failed: could not create test conversation %q: %v",
			conversationName,
			err,
		)
	}

	// Get the conversation ID.
	conversations, err := models.FindConvoByName(conversationName)
	if err != nil {
		t.Fatalf(
			"IT-DB-008 failed: could not find test conversation %q: %v",
			conversationName,
			err,
		)
	}

	if len(conversations) != 1 {
		t.Fatalf(
			"IT-DB-008 failed: expected 1 conversation, found %d",
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
				"IT-DB-008 cleanup failed: could not delete test conversation: %v",
				err,
			)
		}
	})

	// Add the user to the conversation.
	err = models.AddUserToConvo(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-008 failed: could not add user to conversation: %v",
			err,
		)
	}

	// Remove the user from the conversation.
	err = models.RemoveUserFromConversation(user.ID, conversationID)
	if err != nil {
		t.Fatalf(
			"IT-DB-008 failed: RemoveUserFromConversation(%d, %d) returned an error: %v",
			user.ID,
			conversationID,
			err,
		)
	}

	// Verify that the membership was removed.
	var count int

	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ?
		   AND conversation_id = ?`,
		user.ID,
		conversationID,
	).Scan(&count)

	if err != nil {
		t.Fatalf(
			"IT-DB-008 failed: could not verify membership removal: %v",
			err,
		)
	}

	if count != 0 {
		t.Fatalf(
			"IT-DB-008 failed: expected 0 membership records, found %d",
			count,
		)
	}

	t.Logf(
		"IT-DB-008 PASS: user %d was removed from conversation %d successfully",
		user.ID,
		conversationID,
	)
}

func TestIT_DB_009_FindAllConversationsFromUser(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db009_user")
	conversationName1 := testutil.TestUsername("db009_conversation1")
	conversationName2 := testutil.TestUsername("db009_conversation2")
	conversationName3 := testutil.TestUsername("db009_conversation3")

	// Create test user.
	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-009 failed: could not create test user %q: %v",
			username,
			err,
		)
	}

	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"IT-DB-009 failed: could not find test user %q: %v",
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
				"IT-DB-009 cleanup failed: could not delete test user %q: %v",
				username,
				err,
			)
		}
	})

	// Create three conversations.
	conversationNames := []string{
		conversationName1,
		conversationName2,
		conversationName3,
	}

	for _, name := range conversationNames {
		err := models.CreateConvo(name)
		if err != nil {
			t.Fatalf(
				"IT-DB-009 failed: could not create test conversation %q: %v",
				name,
				err,
			)
		}
	}

	// Clean up the conversations.
	t.Cleanup(func() {
		for _, name := range conversationNames {
			_, err := database.Exec(
				"DELETE FROM conversations WHERE name = ?",
				name,
			)

			if err != nil {
				t.Errorf(
					"IT-DB-009 cleanup failed: could not delete conversation %q: %v",
					name,
					err,
				)
			}
		}
	})

	// Get the conversation IDs.
	var conversationIDs []int

	for _, name := range conversationNames {
		conversations, err := models.FindConvoByName(name)
		if err != nil {
			t.Fatalf(
				"IT-DB-009 failed: could not find conversation %q: %v",
				name,
				err,
			)
		}

		if len(conversations) != 1 {
			t.Fatalf(
				"IT-DB-009 failed: expected 1 conversation named %q, found %d",
				name,
				len(conversations),
			)
		}

		conversationIDs = append(conversationIDs, conversations[0].ID)
	}

	// Add the user to only the first two conversations.
	for _, conversationID := range conversationIDs[:2] {
		err := models.AddUserToConvo(user.ID, conversationID)
		if err != nil {
			t.Fatalf(
				"IT-DB-009 failed: could not add user %d to conversation %d: %v",
				user.ID,
				conversationID,
				err,
			)
		}
	}

	// Find all conversations belonging to the user.
	conversations, err := models.FindAllConvoFromUserByID(user.ID)
	if err != nil {
		t.Fatalf(
			"IT-DB-009 failed: FindAllConvoFromUserByID(%d) returned an error: %v",
			user.ID,
			err,
		)
	}

	// The user should belong to exactly two conversations.
	if len(conversations) != 2 {
		t.Fatalf(
			"IT-DB-009 failed: expected 2 conversations, found %d",
			len(conversations),
		)
	}

	// Verify that the returned conversations are the correct ones.
	found := make(map[int]bool)

	for _, conversation := range conversations {
		found[conversation.ID] = true
	}

	if !found[conversationIDs[0]] {
		t.Fatalf(
			"IT-DB-009 failed: conversation %d was not returned",
			conversationIDs[0],
		)
	}

	if !found[conversationIDs[1]] {
		t.Fatalf(
			"IT-DB-009 failed: conversation %d was not returned",
			conversationIDs[1],
		)
	}

	// The third conversation should not be returned.
	if found[conversationIDs[2]] {
		t.Fatalf(
			"IT-DB-009 failed: conversation %d was returned even though the user is not a member",
			conversationIDs[2],
		)
	}

	t.Logf(
		"IT-DB-009 PASS: user %d correctly retrieved %d conversations",
		user.ID,
		len(conversations),
	)
}

func TestIT_DB_010_FindAllUsersFromConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username1 := testutil.TestUsername("db010_user1")
	username2 := testutil.TestUsername("db010_user2")
	username3 := testutil.TestUsername("db010_user3")
	conversationName1 := testutil.TestUsername("db010_conversation1")
	conversationName2 := testutil.TestUsername("db010_conversation2")

	usernames := []string{
		username1,
		username2,
		username3,
	}

	// Create test users.
	for _, username := range usernames {
		err := models.CreateUser(username)
		if err != nil {
			t.Fatalf(
				"IT-DB-010 failed: could not create test user %q: %v",
				username,
				err,
			)
		}
	}

	// Clean up test users.
	t.Cleanup(func() {
		for _, username := range usernames {
			_, err := database.Exec(
				"DELETE FROM users WHERE username = ?",
				username,
			)

			if err != nil {
				t.Errorf(
					"IT-DB-010 cleanup failed: could not delete user %q: %v",
					username,
					err,
				)
			}
		}
	})

	// Retrieve the users so we have their IDs.
	user1, err := models.FindUserByName(username1)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: could not find user %q: %v",
			username1,
			err,
		)
	}

	user2, err := models.FindUserByName(username2)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: could not find user %q: %v",
			username2,
			err,
		)
	}

	user3, err := models.FindUserByName(username3)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: could not find user %q: %v",
			username3,
			err,
		)
	}

	// Create two conversations.
	conversationNames := []string{
		conversationName1,
		conversationName2,
	}

	for _, name := range conversationNames {
		err := models.CreateConvo(name)
		if err != nil {
			t.Fatalf(
				"IT-DB-010 failed: could not create conversation %q: %v",
				name,
				err,
			)
		}
	}

	// Clean up conversations.
	t.Cleanup(func() {
		for _, name := range conversationNames {
			_, err := database.Exec(
				"DELETE FROM conversations WHERE name = ?",
				name,
			)

			if err != nil {
				t.Errorf(
					"IT-DB-010 cleanup failed: could not delete conversation %q: %v",
					name,
					err,
				)
			}
		}
	})

	// Retrieve conversation IDs.
	conversation1, err := models.FindConvoByName(conversationName1)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: could not find conversation %q: %v",
			conversationName1,
			err,
		)
	}

	if len(conversation1) != 1 {
		t.Fatalf(
			"IT-DB-010 failed: expected 1 conversation, found %d",
			len(conversation1),
		)
	}

	conversation2, err := models.FindConvoByName(conversationName2)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: could not find conversation %q: %v",
			conversationName2,
			err,
		)
	}

	if len(conversation2) != 1 {
		t.Fatalf(
			"IT-DB-010 failed: expected 1 conversation, found %d",
			len(conversation2),
		)
	}

	conversationID1 := conversation1[0].ID
	conversationID2 := conversation2[0].ID

	// Add users 1 and 2 to conversation 1.
	err = models.AddUserToConvo(user1.ID, conversationID1)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: could not add user %d to conversation %d: %v",
			user1.ID,
			conversationID1,
			err,
		)
	}

	err = models.AddUserToConvo(user2.ID, conversationID1)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: could not add user %d to conversation %d: %v",
			user2.ID,
			conversationID1,
			err,
		)
	}

	// Add user 3 only to conversation 2.
	err = models.AddUserToConvo(user3.ID, conversationID2)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: could not add user %d to conversation %d: %v",
			user3.ID,
			conversationID2,
			err,
		)
	}

	// Find all users in conversation 1.
	users, err := models.FindAllUserFromConvoByID(conversationID1)
	if err != nil {
		t.Fatalf(
			"IT-DB-010 failed: FindAllUserFromConvoByID(%d) returned an error: %v",
			conversationID1,
			err,
		)
	}

	// Conversation 1 should contain exactly users 1 and 2.
	if len(users) != 2 {
		t.Fatalf(
			"IT-DB-010 failed: expected 2 users, found %d",
			len(users),
		)
	}

	// Verify the returned users.
	found := make(map[int]bool)

	for _, user := range users {
		found[user.ID] = true
	}

	if !found[user1.ID] {
		t.Fatalf(
			"IT-DB-010 failed: user %d was not returned",
			user1.ID,
		)
	}

	if !found[user2.ID] {
		t.Fatalf(
			"IT-DB-010 failed: user %d was not returned",
			user2.ID,
		)
	}

	// User 3 should not be returned.
	if found[user3.ID] {
		t.Fatalf(
			"IT-DB-010 failed: user %d was returned even though they are not a member",
			user3.ID,
		)
	}

	t.Logf(
		"IT-DB-010 PASS: conversation %d correctly retrieved %d users",
		conversationID1,
		len(users),
	)
}

func TestIT_DB_021_AddNonExistentUser(t *testing.T) {
	database := testutil.SetupDatabase(t)

	// Create a valid conversation.
	result, err := database.Exec(
		"INSERT INTO conversations (name, conversation_type) VALUES (?, 'group')",
		"IT-DB-021-conversation",
	)
	if err != nil {
		t.Fatalf("failed to create test conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	// Use a user ID that does not exist.
	nonExistentUserID := 999999999

	err = models.AddUserToConvo(
		int(nonExistentUserID),
		int(conversationID),
	)

	if err == nil {
		t.Fatal("expected error when adding non-existent user, got nil")
	}

	t.Logf(
		"IT-DB-021 PASS: adding non-existent user correctly failed: %v",
		err,
	)

	// Cleanup.
	_, err = database.Exec(
		"DELETE FROM conversations WHERE id = ?",
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to clean up test conversation: %v", err)
	}
}

func TestIT_DB_022_RemoveNonMember(t *testing.T) {
	database := testutil.SetupDatabase(t)

	// Create two users.
	user1Name := testutil.TestUsername("IT-DB-022-user1")
	user2Name := testutil.TestUsername("IT-DB-022-user2")

	err := models.CreateUser(user1Name)
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	err = models.CreateUser(user2Name)
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	user1, err := models.FindUserByName(user1Name)
	if err != nil {
		t.Fatalf("failed to find user1: %v", err)
	}

	user2, err := models.FindUserByName(user2Name)
	if err != nil {
		t.Fatalf("failed to find user2: %v", err)
	}

	// Create a conversation.
	result, err := database.Exec(
		"INSERT INTO conversations (name, conversation_type) VALUES (?, 'group')",
		"IT-DB-022-conversation",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	// Add only user1.
	err = models.AddUserToConvo(user1.ID, int(conversationID))
	if err != nil {
		t.Fatalf("failed to add user1 to conversation: %v", err)
	}

	// user2 is NOT a member.
	err = models.RemoveUserFromConversation(
		user2.ID,
		int(conversationID),
	)

	if err != nil {
		t.Fatalf(
			"expected removing non-member to complete without database error, got: %v",
			err,
		)
	}

	// Verify user1 is still a member.
	var count int

	err = database.QueryRow(`
		SELECT COUNT(*)
		FROM user_in_conversation
		WHERE user_id = ?
		AND conversation_id = ?
	`, user1.ID, conversationID).Scan(&count)

	if err != nil {
		t.Fatalf("failed to verify membership: %v", err)
	}

	if count != 1 {
		t.Fatalf(
			"expected user1 to remain a member, got membership count %d",
			count,
		)
	}

	t.Log(
		"IT-DB-022 PASS: removing a non-member caused no error and did not affect existing membership",
	)

	// Cleanup.
	_, err = database.Exec(
		"DELETE FROM users WHERE id IN (?, ?)",
		user1.ID,
		user2.ID,
	)
	if err != nil {
		t.Fatalf("failed to clean up test users: %v", err)
	}

	_, err = database.Exec(
		"DELETE FROM conversations WHERE id = ?",
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to clean up test conversation: %v", err)
	}
}

func TestIT_DB_023_DeleteConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	// Create a test user.
	username := testutil.TestUsername("IT-DB-023-user")

	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	// Create a conversation.
	result, err := database.Exec(
		"INSERT INTO conversations (name, conversation_type) VALUES (?, 'group')",
		"IT-DB-023-conversation",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	// Add the user to the conversation.
	err = models.AddUserToConvo(user.ID, int(conversationID))
	if err != nil {
		t.Fatalf("failed to add user to conversation: %v", err)
	}

	// Create a message.
	message, err := models.CreateMessage(
		user.ID,
		int(conversationID),
		"IT-DB-023 test message",
	)
	if err != nil {
		t.Fatalf("failed to create message: %v", err)
	}

	// Create a message receipt.
	//
	// The sender is excluded from receipts, so create another
	// user and add them to the conversation.
	recipientName := testutil.TestUsername("IT-DB-023-recipient")

	err = models.CreateUser(recipientName)
	if err != nil {
		t.Fatalf("failed to create recipient: %v", err)
	}

	recipient, err := models.FindUserByName(recipientName)
	if err != nil {
		t.Fatalf("failed to find recipient: %v", err)
	}

	err = models.AddUserToConvo(
		recipient.ID,
		int(conversationID),
	)
	if err != nil {
		t.Fatalf("failed to add recipient to conversation: %v", err)
	}

	err = models.CreateMessageReceipts(
		message.ID,
		int(conversationID),
		user.ID,
	)
	if err != nil {
		t.Fatalf("failed to create message receipt: %v", err)
	}

	// Delete the conversation.
	conversation, err := models.FindConvoById(int(conversationID))
	if err != nil {
		t.Fatalf("failed to find conversation: %v", err)
	}

	err = conversation.DeleteConvo()
	if err != nil {
		t.Fatalf("failed to delete conversation: %v", err)
	}

	// Verify conversation was deleted.
	var count int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&count)

	if err != nil {
		t.Fatalf("failed to verify conversation deletion: %v", err)
	}

	if count != 0 {
		t.Fatalf("conversation still exists after deletion")
	}

	// Verify memberships were deleted.
	err = database.QueryRow(`
		SELECT COUNT(*)
		FROM user_in_conversation
		WHERE conversation_id = ?
	`, conversationID).Scan(&count)

	if err != nil {
		t.Fatalf("failed to verify membership deletion: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"expected memberships to be deleted, found %d",
			count,
		)
	}

	// Verify messages were deleted.
	err = database.QueryRow(`
		SELECT COUNT(*)
		FROM messages
		WHERE conversation_id = ?
	`, conversationID).Scan(&count)

	if err != nil {
		t.Fatalf("failed to verify message deletion: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"expected messages to be deleted, found %d",
			count,
		)
	}

	// Verify message receipts were deleted.
	err = database.QueryRow(`
		SELECT COUNT(*)
		FROM message_receipts
		WHERE message_id = ?
	`, message.ID).Scan(&count)

	if err != nil {
		t.Fatalf("failed to verify receipt deletion: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"expected message receipts to be deleted, found %d",
			count,
		)
	}

	t.Log(
		"IT-DB-023 PASS: deleting conversation cascaded to memberships, messages, and message receipts",
	)

	// Users are not owned by the conversation, so clean them up separately.
	_, err = database.Exec(
		"DELETE FROM users WHERE id IN (?, ?)",
		user.ID,
		recipient.ID,
	)
	if err != nil {
		t.Fatalf("failed to clean up test users: %v", err)
	}
}

func TestIT_DB_024_ForeignKeyIntegrity(t *testing.T) {
	database := testutil.SetupDatabase(t)

	// Create one valid user.
	username := testutil.TestUsername("IT-DB-024-user")

	err := models.CreateUser(username)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	// Create one valid conversation.
	result, err := database.Exec(
		"INSERT INTO conversations (name, conversation_type) VALUES (?, 'group')",
		"IT-DB-024-conversation",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	// ---------------------------------------------------------
	// Test 1: Valid user + invalid conversation
	// ---------------------------------------------------------

	nonExistentConversationID := 999999999

	err = models.AddUserToConvo(
		user.ID,
		nonExistentConversationID,
	)

	if err == nil {
		t.Fatal(
			"expected foreign-key error when using non-existent conversation, got nil",
		)
	}

	t.Logf(
		"valid user + invalid conversation correctly rejected: %v",
		err,
	)

	// ---------------------------------------------------------
	// Test 2: Invalid user + valid conversation
	// ---------------------------------------------------------

	nonExistentUserID := 999999999

	err = models.AddUserToConvo(
		nonExistentUserID,
		int(conversationID),
	)

	if err == nil {
		t.Fatal(
			"expected foreign-key error when using non-existent user, got nil",
		)
	}

	t.Logf(
		"invalid user + valid conversation correctly rejected: %v",
		err,
	)

	// ---------------------------------------------------------
	// Verify no invalid rows were inserted.
	// ---------------------------------------------------------

	var count int

	err = database.QueryRow(`
		SELECT COUNT(*)
		FROM user_in_conversation
		WHERE conversation_id = ?
	`, conversationID).Scan(&count)

	if err != nil {
		t.Fatalf("failed to verify membership rows: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"expected 0 membership rows after rejected inserts, found %d",
			count,
		)
	}

	t.Log(
		"IT-DB-024 PASS: foreign-key constraints correctly rejected invalid user and conversation references",
	)

	// Cleanup.
	_, err = database.Exec(
		"DELETE FROM users WHERE id = ?",
		user.ID,
	)
	if err != nil {
		t.Fatalf("failed to clean up test user: %v", err)
	}

	_, err = database.Exec(
		"DELETE FROM conversations WHERE id = ?",
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to clean up test conversation: %v", err)
	}
}

func TestIT_DB_025_ConversationTypes(t *testing.T) {
	database := testutil.SetupDatabase(t)

	// Create a direct conversation.
	directResult, err := database.Exec(
		"INSERT INTO conversations (name, conversation_type) VALUES (?, 'direct')",
		"IT-DB-025-direct",
	)
	if err != nil {
		t.Fatalf("failed to create direct conversation: %v", err)
	}

	directID, err := directResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get direct conversation ID: %v", err)
	}

	// Create a group conversation.
	groupResult, err := database.Exec(
		"INSERT INTO conversations (name, conversation_type) VALUES (?, 'group')",
		"IT-DB-025-group",
	)
	if err != nil {
		t.Fatalf("failed to create group conversation: %v", err)
	}

	groupID, err := groupResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get group conversation ID: %v", err)
	}

	// Retrieve the direct conversation.
	directConversation, err := models.FindConvoById(int(directID))
	if err != nil {
		t.Fatalf("failed to retrieve direct conversation: %v", err)
	}

	if directConversation.ConversationType != "direct" {
		t.Fatalf(
			"expected direct conversation type, got %q",
			directConversation.ConversationType,
		)
	}

	// Retrieve the group conversation.
	groupConversation, err := models.FindConvoById(int(groupID))
	if err != nil {
		t.Fatalf("failed to retrieve group conversation: %v", err)
	}

	if groupConversation.ConversationType != "group" {
		t.Fatalf(
			"expected group conversation type, got %q",
			groupConversation.ConversationType,
		)
	}

	t.Log(
		"IT-DB-025 PASS: direct and group conversation types were stored and retrieved correctly",
	)

	// Cleanup.
	_, err = database.Exec(
		"DELETE FROM conversations WHERE id IN (?, ?)",
		directID,
		groupID,
	)
	if err != nil {
		t.Fatalf("failed to clean up conversations: %v", err)
	}
}