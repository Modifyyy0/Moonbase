package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Moonbase/handlers"
	"Moonbase/src/models"
	"Moonbase/test/testutil"
)

func TestIT_API_001_ValidLogin(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("IT-API-001")

	requestBody := `{"username":"` + username + `"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(requestBody),
	)

	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handlers.LoginHandler(recorder, req)

	// ---------------------------------------------------------
	// Verify HTTP status
	// ---------------------------------------------------------

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected HTTP 201 Created, got %d",
			recorder.Code,
		)
	}

	// ---------------------------------------------------------
	// Verify response JSON
	// ---------------------------------------------------------

	var user models.User

	err := json.Unmarshal(
		recorder.Body.Bytes(),
		&user,
	)

	if err != nil {
		t.Fatalf(
			"failed to decode response JSON: %v",
			err,
		)
	}

	if user.ID <= 0 {
		t.Fatalf(
			"expected valid user ID, got %d",
			user.ID,
		)
	}

	if user.Name != username {
		t.Fatalf(
			"expected username %q, got %q",
			username,
			user.Name,
		)
	}

	// ---------------------------------------------------------
	// Verify session cookie
	// ---------------------------------------------------------

	cookies := recorder.Result().Cookies()

	var sessionCookie *http.Cookie

	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session_token cookie, got none")
	}

	if sessionCookie.Value == "" {
		t.Fatal("expected session_token cookie to contain a value")
	}

	if !sessionCookie.HttpOnly {
		t.Fatal("expected session_token cookie to be HttpOnly")
	}

	// ---------------------------------------------------------
	// Verify user exists in database
	// ---------------------------------------------------------

	databaseUser, err := models.FindUserByName(username)
	if err != nil {
		t.Fatalf(
			"expected user to exist in database: %v",
			err,
		)
	}

	if databaseUser.ID != user.ID {
		t.Fatalf(
			"response user ID %d does not match database user ID %d",
			user.ID,
			databaseUser.ID,
		)
	}

	// ---------------------------------------------------------
	// Verify session exists in database
	// ---------------------------------------------------------

	var sessionUsername string

	err = database.QueryRow(`
		SELECT username
		FROM sessions
		WHERE session_token = ?
	`, sessionCookie.Value).Scan(&sessionUsername)

	if err != nil {
		t.Fatalf(
			"expected session to exist in database: %v",
			err,
		)
	}

	if sessionUsername != username {
		t.Fatalf(
			"expected session username %q, got %q",
			username,
			sessionUsername,
		)
	}

	t.Logf(
		"IT-API-001 PASS: user %q logged in, session created, and response verified",
		username,
	)

	// ---------------------------------------------------------
	// Cleanup
	// ---------------------------------------------------------

	_, err = database.Exec(
		"DELETE FROM users WHERE id = ?",
		user.ID,
	)

	if err != nil {
		t.Fatalf(
			"failed to clean up test user: %v",
			err,
		)
	}
}

func TestIT_API_002_ExistingUserLogin(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := testutil.TestUsername("IT-API-002")

	// Create the user first.
	result, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get test user ID: %v", err)
	}

	t.Cleanup(func() {
		database.Exec("DELETE FROM users WHERE id = ?", userID)
	})

	// Send login request.
	requestBody := `{"username":"` + username + `"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(requestBody),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handlers.LoginHandler(recorder, req)

	// Verify status.
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200 OK, got %d",
			recorder.Code,
		)
	}

	// Verify response.
	var user models.User

	if err := json.Unmarshal(recorder.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if user.ID != int(userID) {
		t.Fatalf(
			"expected user ID %d, got %d",
			userID,
			user.ID,
		)
	}

	if user.Name != username {
		t.Fatalf(
			"expected username %q, got %q",
			username,
			user.Name,
		)
	}

	// Verify session cookie.
	cookies := recorder.Result().Cookies()

	var sessionCookie *http.Cookie

	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session_token cookie, got none")
	}

	if sessionCookie.Value == "" {
		t.Fatal("expected session_token cookie to contain a value")
	}

	// Verify session exists in DB.
	var sessionUsername string

	err = database.QueryRow(`
		SELECT username
		FROM sessions
		WHERE session_token = ?
	`, sessionCookie.Value).Scan(&sessionUsername)

	if err != nil {
		t.Fatalf(
			"expected session to exist in database: %v",
			err,
		)
	}

	if sessionUsername != username {
		t.Fatalf(
			"expected session username %q, got %q",
			username,
			sessionUsername,
		)
	}

	// Verify no duplicate user was created.
	var userCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username,
	).Scan(&userCount)

	if err != nil {
		t.Fatalf("failed to count test users: %v", err)
	}

	if userCount != 1 {
		t.Fatalf(
			"expected exactly 1 user, found %d",
			userCount,
		)
	}

	t.Logf(
		"IT-API-002 PASS: existing user %q logged in and received a new session",
		username,
	)
}

func TestIT_API_003_InvalidJSON(t *testing.T) {
	database := testutil.SetupDatabase(t)

	// Record the database state before the request.
	var emptyUsersBefore int

	err := database.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ''",
	).Scan(&emptyUsersBefore)

	if err != nil {
		t.Fatalf("failed to check initial database state: %v", err)
	}

	requestBody := `{"username":"broken"`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(requestBody),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handlers.LoginHandler(recorder, req)

	// Verify HTTP status.
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected HTTP 400 Bad Request, got %d",
			recorder.Code,
		)
	}

	// Verify no session cookie was created.
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			t.Fatal("did not expect a session_token cookie")
		}
	}

	// Verify the invalid request did not modify the database.
	var emptyUsersAfter int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ''",
	).Scan(&emptyUsersAfter)

	if err != nil {
		t.Fatalf("failed to check database state: %v", err)
	}

	if emptyUsersAfter != emptyUsersBefore {
		t.Fatalf(
			"invalid JSON changed empty-user count from %d to %d",
			emptyUsersBefore,
			emptyUsersAfter,
		)
	}

	t.Log(
		"IT-API-003 PASS: invalid JSON rejected with no user or session created",
	)
}

func TestIT_API_004_EmptyUsername(t *testing.T) {
	database := testutil.SetupDatabase(t)

	// Record the database state before the request.
	var emptyUsersBefore int

	err := database.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ''",
	).Scan(&emptyUsersBefore)

	if err != nil {
		t.Fatalf("failed to check initial database state: %v", err)
	}

	var emptySessionsBefore int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE username = ''",
	).Scan(&emptySessionsBefore)

	if err != nil {
		t.Fatalf("failed to check initial session state: %v", err)
	}

	// Send a valid JSON request with an empty username.
	requestBody := `{"username":""}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(requestBody),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handlers.LoginHandler(recorder, req)

	// Empty username should be rejected.
	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status for empty username, got %d",
			recorder.Code,
		)
	}

	// Verify no session cookie was created.
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			t.Fatal("did not expect a session_token cookie")
		}
	}

	// Verify no empty user was created.
	var emptyUsersAfter int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ''",
	).Scan(&emptyUsersAfter)

	if err != nil {
		t.Fatalf("failed to check final user state: %v", err)
	}

	if emptyUsersAfter != emptyUsersBefore {
		t.Fatalf(
			"empty username changed user count from %d to %d",
			emptyUsersBefore,
			emptyUsersAfter,
		)
	}

	// Verify no empty-user session was created.
	var emptySessionsAfter int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE username = ''",
	).Scan(&emptySessionsAfter)

	if err != nil {
		t.Fatalf("failed to check final session state: %v", err)
	}

	if emptySessionsAfter != emptySessionsBefore {
		t.Fatalf(
			"empty username changed session count from %d to %d",
			emptySessionsBefore,
			emptySessionsAfter,
		)
	}

	t.Log(
		"IT-API-004 PASS: empty username rejected with no user or session created",
	)
}