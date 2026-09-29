package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"Moonbase/src/db"
)

func SetupDatabase(t *testing.T) *sql.DB {
	t.Helper()

	// Find the server directory based on this source file's location.
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test utility file location")
	}

	serverDir := filepath.Clean(
		filepath.Join(filepath.Dir(currentFile), "..", ".."),
	)

	// db.Connect() expects .env to be in its working directory.
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	if err := os.Chdir(serverDir); err != nil {
		t.Fatalf("failed to change to server directory: %v", err)
	}

	// Restore the original working directory after the test setup.
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})

	if db.DB == nil {
		if err := db.Connect(); err != nil {
			t.Fatalf("failed to connect to database: %v", err)
		}
	}

	return db.DB
}

func TestUsername(testID string) string {
	return fmt.Sprintf("test_%s_%d", testID, time.Now().UnixNano())
}

func DeleteUsersByUsername(t *testing.T, database *sql.DB, usernames ...string) {
	t.Helper()

	for _, username := range usernames {
		_, err := database.Exec(
			"DELETE FROM users WHERE username = ?",
			username,
		)

		if err != nil {
			t.Errorf(
				"failed to clean up test user %q: %v",
				username,
				err,
			)
		}
	}
}

func DeleteConversationsByID(t *testing.T, database *sql.DB, conversationIDs ...int64) {
	t.Helper()

	for _, conversationID := range conversationIDs {
		_, err := database.Exec(
			"DELETE FROM conversations WHERE id = ?",
			conversationID,
		)

		if err != nil {
			t.Errorf(
				"failed to clean up conversation %d: %v",
				conversationID,
				err,
			)
		}
	}
}
