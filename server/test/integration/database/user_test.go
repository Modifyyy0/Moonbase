package database_test

import (
	"database/sql"
	"testing"

	"Moonbase/src/models"
	"Moonbase/test/testutil"
)

func TestIT_DB_001_ValidUserCreation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db001")

	// Make sure the test starts clean.
	testutil.DeleteUserByUsername(t, database, username)

	// Always clean up the test user after the test.
	t.Cleanup(func() {
		testutil.DeleteUserByUsername(t, database, username)
	})

	// Execute the operation being tested.
	err := models.CreateUser(username)

	if err != nil {
		t.Fatalf(
			"IT-DB-001 failed: CreateUser(%q) returned an error: %v",
			username,
			err,
		)
	}

	// Verify that the database actually contains the user.
	var id int

	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		username,
	).Scan(&id)

	if err != nil {
		t.Fatalf(
			"IT-DB-001 failed: user %q was not found after creation: %v",
			username,
			err,
		)
	}

	if id <= 0 {
		t.Fatalf(
			"IT-DB-001 failed: expected a valid user ID, got %d",
			id,
		)
	}

	// Verify that exactly one record exists.
	var count int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username,
	).Scan(&count)

	if err != nil {
		t.Fatalf(
			"IT-DB-001 failed: could not verify user count: %v",
			err,
		)
	}

	if count != 1 {
		t.Fatalf(
			"IT-DB-001 failed: expected exactly 1 user, found %d",
			count,
		)
	}

	t.Logf(
		"IT-DB-001 PASS: user %q was created successfully with ID %d",
		username,
		id,
	)

	_ = sql.ErrNoRows
}

func TestIT_DB_002_DuplicateUsername(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db002")

	// Make sure the test starts clean.
	testutil.DeleteUserByUsername(t, database, username)

	// Always clean up the test user after the test.
	t.Cleanup(func() {
		testutil.DeleteUserByUsername(t, database, username)
	})

	// Create the user for the first time.
	err := models.CreateUser(username)

	if err != nil {
		t.Fatalf(
			"IT-DB-002 setup failed: could not create initial user %q: %v",
			username,
			err,
		)
	}

	// Try to create the same username again.
	err = models.CreateUser(username)

	if err == nil {
		t.Fatalf(
			"IT-DB-002 failed: expected duplicate username creation to fail",
		)
	}

	// Verify that only one record exists.
	var count int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username,
	).Scan(&count)

	if err != nil {
		t.Fatalf(
			"IT-DB-002 failed: could not verify user count: %v",
			err,
		)
	}

	if count != 1 {
		t.Fatalf(
			"IT-DB-002 failed: expected exactly 1 user, found %d",
			count,
		)
	}

	t.Logf(
		"IT-DB-002 PASS: duplicate username %q was rejected and existing record was preserved",
		username,
	)
}

func TestIT_DB_003_InvalidUserLookup(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("db003")

	// Make sure this username does not exist.
	testutil.DeleteUserByUsername(t, database, username)

	// Look up a user that does not exist.
	user, err := models.FindUserByName(username)

	if err != sql.ErrNoRows {
		t.Fatalf(
			"IT-DB-003 failed: expected sql.ErrNoRows, got user=%v, err=%v",
			user,
			err,
		)
	}

	if user != nil {
		t.Fatalf(
			"IT-DB-003 failed: expected no user, got %+v",
			user,
		)
	}

	t.Logf(
		"IT-DB-003 PASS: non-existent user %q returned sql.ErrNoRows",
		username,
	)
}