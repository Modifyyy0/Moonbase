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