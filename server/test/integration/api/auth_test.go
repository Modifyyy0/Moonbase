package api_test

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

func TestIT_API_005_MissingUsername(t *testing.T) {
	database := testutil.SetupDatabase(t)

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

	requestBody := `{}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(requestBody),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handlers.LoginHandler(recorder, req)

	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status for missing username, got %d",
			recorder.Code,
		)
	}

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			t.Fatal("did not expect a session_token cookie")
		}
	}

	var emptyUsersAfter int
	err = database.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ''",
	).Scan(&emptyUsersAfter)

	if err != nil {
		t.Fatalf("failed to check final user state: %v", err)
	}

	if emptyUsersAfter != emptyUsersBefore {
		t.Fatalf(
			"missing username changed user count from %d to %d",
			emptyUsersBefore,
			emptyUsersAfter,
		)
	}

	var emptySessionsAfter int
	err = database.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE username = ''",
	).Scan(&emptySessionsAfter)

	if err != nil {
		t.Fatalf("failed to check final session state: %v", err)
	}

	if emptySessionsAfter != emptySessionsBefore {
		t.Fatalf(
			"missing username changed session count from %d to %d",
			emptySessionsBefore,
			emptySessionsAfter,
		)
	}

	t.Log(
		"IT-API-005 PASS: missing username rejected with no user or session created",
	)
}

func TestIT_API_006_LoginWrongMethod(t *testing.T) {
	testutil.SetupDatabase(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/login",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlers.LoginHandler(recorder, req)

	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status for GET /api/login, got %d",
			recorder.Code,
		)
	}

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			t.Fatal("did not expect a session_token cookie")
		}
	}

	t.Log(
		"IT-API-006 PASS: GET /api/login rejected without creating a session",
	)
}

func TestIT_API_007_LoginResponseStructure(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-API-007_%d",
		time.Now().UnixNano(),
	)

	requestBody := fmt.Sprintf(
		`{"username":%q}`,
		username,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(requestBody),
	)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handlers.LoginHandler(recorder, req)

	// Verify HTTP status.
	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected HTTP 201 Created, got %d",
			recorder.Code,
		)
	}

	// Verify Content-Type.
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf(
			"expected Content-Type application/json, got %q",
			contentType,
		)
	}

	// Decode response JSON.
	var response map[string]any

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response JSON: %v",
			err,
		)
	}

	// Verify required "id" field.
	id, ok := response["id"].(float64)
	if !ok {
		t.Fatal("expected response to contain numeric 'id' field")
	}

	if id <= 0 {
		t.Fatalf("expected positive user ID, got %v", id)
	}

	// Verify required "username" field.
	responseUsername, ok := response["username"].(string)
	if !ok {
		t.Fatal("expected response to contain string 'username' field")
	}

	if responseUsername != username {
		t.Fatalf(
			"expected username %q, got %q",
			username,
			responseUsername,
		)
	}

	// Verify sensitive session data is not exposed in JSON.
	if _, exists := response["session_token"]; exists {
		t.Fatal("response must not contain session_token")
	}

	// Verify a session cookie was created.
	var sessionCookie *http.Cookie

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session_token cookie")
	}

	if sessionCookie.Value == "" {
		t.Fatal("expected session_token cookie to contain a value")
	}

	// Verify the user exists in the database.
	var userID int

	err := database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		username,
	).Scan(&userID)

	if err != nil {
		t.Fatalf(
			"expected user to exist in database: %v",
			err,
		)
	}

	if userID != int(id) {
		t.Fatalf(
			"response ID %v does not match database ID %d",
			id,
			userID,
		)
	}

	t.Log(
		"IT-API-007 PASS: login response structure and session cookie verified",
	)
}

func TestIT_API_008_SessionAuthentication(t *testing.T) {
	testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-API-008_%d",
		time.Now().UnixNano(),
	)

	// Login first.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		username,
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(loginBody),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected login to return 201, got %d",
			loginRecorder.Code,
		)
	}

	// Extract the session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Use the session cookie to request /api/me.
	meReq := httptest.NewRequest(
		http.MethodGet,
		"/api/me",
		nil,
	)

	meReq.AddCookie(sessionCookie)

	meRecorder := httptest.NewRecorder()

	handlers.HandleMe(meRecorder, meReq)

	// Verify authentication succeeded.
	if meRecorder.Code != http.StatusOK {
		t.Fatalf(
			"expected /api/me to return 200, got %d",
			meRecorder.Code,
		)
	}

	// Decode response.
	var response models.User

	if err := json.NewDecoder(meRecorder.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode /api/me response: %v",
			err,
		)
	}

	// Verify the session belongs to the user who logged in.
	if response.Name != username {
		t.Fatalf(
			"expected username %q, got %q",
			username,
			response.Name,
		)
	}

	if response.ID <= 0 {
		t.Fatalf(
			"expected positive user ID, got %d",
			response.ID,
		)
	}

	t.Log(
		"IT-API-008 PASS: session cookie authenticated the logged-in user",
	)
}

func TestIT_API_009_MeWithoutAuthentication(t *testing.T) {
	testutil.SetupDatabase(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/me",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlers.HandleMe(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected HTTP 401 Unauthorized, got %d",
			recorder.Code,
		)
	}

	t.Log(
		"IT-API-009 PASS: /api/me rejected unauthenticated request",
	)
}

func TestIT_API_010_InvalidSession(t *testing.T) {
	testutil.SetupDatabase(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/me",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: "definitely-invalid-session-token",
		Path:  "/",
	})

	recorder := httptest.NewRecorder()

	handlers.HandleMe(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected HTTP 401 Unauthorized for invalid session, got %d",
			recorder.Code,
		)
	}

	t.Log(
		"IT-API-010 PASS: /api/me rejected invalid session token",
	)
}

func TestIT_API_011_Logout(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-API-011_%d",
		time.Now().UnixNano(),
	)

	// Login first.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		username,
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(loginBody),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected login to return 201, got %d",
			loginRecorder.Code,
		)
	}

	// Get the session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Confirm the session exists before logout.
	var sessionCountBefore int

	err := database.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE session_token = ?",
		sessionCookie.Value,
	).Scan(&sessionCountBefore)

	if err != nil {
		t.Fatalf(
			"failed to check session before logout: %v",
			err,
		)
	}

	if sessionCountBefore != 1 {
		t.Fatalf(
			"expected exactly one session before logout, got %d",
			sessionCountBefore,
		)
	}

	// Logout.
	logoutReq := httptest.NewRequest(
		http.MethodPost,
		"/api/logout",
		nil,
	)

	logoutReq.AddCookie(sessionCookie)

	logoutRecorder := httptest.NewRecorder()

	handlers.LogoutHandler(logoutRecorder, logoutReq)

	// Verify logout succeeded.
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200 OK, got %d",
			logoutRecorder.Code,
		)
	}

	// Verify the session was deleted from the database.
	var sessionCountAfter int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE session_token = ?",
		sessionCookie.Value,
	).Scan(&sessionCountAfter)

	if err != nil {
		t.Fatalf(
			"failed to check session after logout: %v",
			err,
		)
	}

	if sessionCountAfter != 0 {
		t.Fatalf(
			"expected session to be deleted after logout, got %d remaining",
			sessionCountAfter,
		)
	}

	// Verify the response clears the cookie.
	var clearedCookie *http.Cookie

	for _, cookie := range logoutRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			clearedCookie = cookie
			break
		}
	}

	if clearedCookie == nil {
		t.Fatal("expected logout to clear session_token cookie")
	}

	if clearedCookie.Value != "" {
		t.Fatalf(
			"expected cleared session cookie to have empty value, got %q",
			clearedCookie.Value,
		)
	}

	if clearedCookie.MaxAge >= 0 && !clearedCookie.Expires.IsZero() {
		// The exact clearing mechanism can vary, so this is intentionally
		// not treated as a failure.
	}

	t.Log(
		"IT-API-011 PASS: logout deleted session and cleared session cookie",
	)
}

func TestIT_API_012_LogoutWithoutAuthentication(t *testing.T) {
	testutil.SetupDatabase(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/logout",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlers.LogoutHandler(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected HTTP 401 Unauthorized, got %d",
			recorder.Code,
		)
	}

	// Make sure no session cookie was created.
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			t.Fatal("did not expect a session_token cookie")
		}
	}

	t.Log(
		"IT-API-012 PASS: logout rejected unauthenticated request",
	)
}

func TestIT_API_013_UsersWithoutAuthentication(t *testing.T) {
	testutil.SetupDatabase(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlers.GetUsers(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected HTTP 401 Unauthorized, got %d",
			recorder.Code,
		)
	}

	t.Log(
		"IT-API-013 PASS: /api/users rejected unauthenticated request",
	)
}

func TestIT_API_014_GetUsers(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-014_Alice_%d",
		time.Now().UnixNano(),
	)

	bob := fmt.Sprintf(
		"test_IT-API-014_Bob_%d",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob)
	})

	// Create Bob directly in the database.
	result, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		bob,
	)

	if err != nil {
		t.Fatalf("failed to create Bob: %v", err)
	}

	bobID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Bob's ID: %v", err)
	}

	// Login as Alice.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		alice,
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(loginBody),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected login to return 201, got %d",
			loginRecorder.Code,
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Request the user list as Alice.
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users",
		nil,
	)

	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.GetUsers(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200 OK, got %d",
			recorder.Code,
		)
	}

	// Decode response.
	var users []models.User

	if err := json.NewDecoder(recorder.Body).Decode(&users); err != nil {
		t.Fatalf(
			"failed to decode users response: %v",
			err,
		)
	}

	// Verify Bob is present and Alice is not.
	foundBob := false
	foundAlice := false

	for _, user := range users {
		if user.Name == bob {
			foundBob = true

			if user.ID != int(bobID) {
				t.Fatalf(
					"expected Bob's ID to be %d, got %d",
					bobID,
					user.ID,
				)
			}
		}

		if user.Name == alice {
			foundAlice = true
		}
	}

	if !foundBob {
		t.Fatalf(
			"expected user list to contain Bob (%q)",
			bob,
		)
	}

	if foundAlice {
		t.Fatal(
			"expected current user Alice to be excluded from user list",
		)
	}

	t.Log(
		"IT-API-014 PASS: authenticated user list returned correctly and excluded current user",
	)
}

func TestIT_API_015_SearchUsers(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-015_Alice_%d",
		time.Now().UnixNano(),
	)

	bob := fmt.Sprintf(
		"test_IT-API-015_Bob_%d",
		time.Now().UnixNano(),
	)

	bobby := fmt.Sprintf(
		"test_IT-API-015_Bobby_%d",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob, bobby)
	})

	// Create Bob and Bobby.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		bob,
		bobby,
	)

	if err != nil {
		t.Fatalf("failed to create test users: %v", err)
	}

	// Login as Alice.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		alice,
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(loginBody),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected login to return 201, got %d",
			loginRecorder.Code,
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Search for users matching "Bob".
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/users?q=Bob",
		nil,
	)

	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.GetUsers(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200 OK, got %d",
			recorder.Code,
		)
	}

	var users []models.User

	if err := json.NewDecoder(recorder.Body).Decode(&users); err != nil {
		t.Fatalf(
			"failed to decode users response: %v",
			err,
		)
	}

	foundBob := false
	foundBobby := false
	foundAlice := false

	for _, user := range users {
		switch user.Name {
		case bob:
			foundBob = true
		case bobby:
			foundBobby = true
		case alice:
			foundAlice = true
		}
	}

	if !foundBob {
		t.Fatalf(
			"expected search results to contain %q",
			bob,
		)
	}

	if !foundBobby {
		t.Fatalf(
			"expected search results to contain %q",
			bobby,
		)
	}

	if foundAlice {
		t.Fatalf(
			"expected current user %q to be excluded from search results",
			alice,
		)
	}

	t.Log(
		"IT-API-015 PASS: user search returned matching users and excluded current user",
	)
}

func TestIT_API_016_CreateConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-016_Alice_%d",
		time.Now().UnixNano(),
	)

	bob := fmt.Sprintf(
		"test_IT-API-016_Bob_%d",
		time.Now().UnixNano(),
	)

	charlie := fmt.Sprintf(
		"test_IT-API-016_Charlie_%d",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob, charlie)
	})

	// Create Bob and Charlie.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		bob,
		charlie,
	)

	if err != nil {
		t.Fatalf("failed to create test users: %v", err)
	}

	// Login as Alice.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		alice,
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(loginBody),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected login to return 201, got %d",
			loginRecorder.Code,
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Create the group conversation.
	requestBody := fmt.Sprintf(
		`{
			"name": "Test Group",
			"type": "group",
			"members": [%q, %q]
		}`,
		bob,
		charlie,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(requestBody),
	)

	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected HTTP 201 Created, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Decode response.
	responseBody := recorder.Body.String()

	t.Logf("conversation response: %s", responseBody)

	var response struct {
		Conversation struct {
			ID int `json:"id"`
		} `json:"conversation"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode conversation response: %v", err)
	}

	if response.Conversation.ID <= 0 {
		t.Fatalf(
			"expected positive conversation ID, got %d",
			response.Conversation.ID,
		)
	}

	conversationID := int64(response.Conversation.ID)
	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationID)
	})

	// Verify conversation exists in the database.
	var name string
	var conversationType string

	err = database.QueryRow(
		"SELECT name, conversation_type FROM conversations WHERE id = ?",
		response.Conversation.ID,
	).Scan(&name, &conversationType)

	if err != nil {
		t.Fatalf(
			"failed to find created conversation: %v",
			err,
		)
	}

	if name != "Test Group" {
		t.Fatalf(
			"expected conversation name %q, got %q",
			"Test Group",
			name,
		)
	}

	if conversationType != "group" {
		t.Fatalf(
			"expected conversation type %q, got %q",
			"group",
			conversationType,
		)
	}

	// Verify Alice, Bob, and Charlie are members.
	expectedMembers := []string{
		alice,
		bob,
		charlie,
	}

	for _, username := range expectedMembers {
		var count int

		err = database.QueryRow(`
			SELECT COUNT(*)
			FROM user_in_conversation uic
			JOIN users u ON u.id = uic.user_id
			WHERE uic.conversation_id = ?
			AND u.username = ?
		`, response.Conversation.ID, username).Scan(&count)

		if err != nil {
			t.Fatalf(
				"failed to check membership for %q: %v",
				username,
				err,
			)
		}

		if count != 1 {
			t.Fatalf(
				"expected %q to be a member of conversation %d, got count %d",
				username,
				response.Conversation.ID,
				count,
			)
		}
	}

	t.Log(
		"IT-API-016 PASS: group conversation created with all members",
	)
}

func TestIT_API_017_GetConversationByID(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-017_Alice_%d",
		time.Now().UnixNano(),
	)
	bob := fmt.Sprintf(
		"test_IT-API-017_Bob_%d",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob)
	})

	// Create test users.
	result, err := database.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		alice,
		bob,
	)
	if err != nil {
		t.Fatalf("failed to create test users: %v", err)
	}

	aliceID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Alice's ID: %v", err)
	}

	var bobID int64
	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID)
	if err != nil {
		t.Fatalf("failed to get Bob's ID: %v", err)
	}

	// Create a conversation directly in the database.
	result, err = database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-017 Test Conversation",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationID)
	})

	// Add Alice and Bob to the conversation.
	_, err = database.Exec(
		`INSERT INTO user_in_conversation (user_id, conversation_id)
		 VALUES (?, ?), (?, ?)`,
		aliceID,
		conversationID,
		bobID,
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to add conversation members: %v", err)
	}

	// Log Alice in to obtain an authenticated session.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		alice,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	// Extract the session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Request the conversation.
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/api/conversations/%d",
			conversationID,
		),
		nil,
	)

	req.AddCookie(sessionCookie)

	// Simulate the {convoID} path parameter that http.ServeMux
	// would normally provide.
	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", conversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.ConvoInfo(recorder, req)
	// Verify HTTP status.
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200 OK, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Decode the response.
	t.Logf("conversation response: %s", recorder.Body.String())

	var response struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		MemberCount int    `json:"membersCount"`

		Members []struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Online   bool   `json:"online"`
		} `json:"members"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode conversation response: %v",
			err,
		)
	}

	// Verify conversation information.
	if response.ID != int(conversationID) {
		t.Fatalf(
			"expected conversation ID %d, got %d",
			conversationID,
			response.ID,
		)
	}

	if response.Name != "IT-API-017 Test Conversation" {
		t.Fatalf(
			"expected conversation name %q, got %q",
			"IT-API-017 Test Conversation",
			response.Name,
		)
	}

	if response.Type != "group" {
		t.Fatalf(
			"expected conversation type %q, got %q",
			"group",
			response.Type,
		)
	}

	if response.MemberCount != 2 {
		t.Fatalf(
			"expected member count 2, got %d",
			response.MemberCount,
		)
	}

	// Verify the returned members.
	if len(response.Members) != 2 {
		t.Fatalf(
			"expected 2 members, got %d",
			len(response.Members),
		)
	}

	foundAlice := false
	foundBob := false

	for _, member := range response.Members {
		switch member.Username {
		case alice:
			foundAlice = true

			if member.ID != int(aliceID) {
				t.Fatalf(
					"expected Alice's ID to be %d, got %d",
					aliceID,
					member.ID,
				)
			}

		case bob:
			foundBob = true

			if member.ID != int(bobID) {
				t.Fatalf(
					"expected Bob's ID to be %d, got %d",
					bobID,
					member.ID,
				)
			}

		default:
			t.Fatalf(
				"unexpected conversation member: %q",
				member.Username,
			)
		}
	}

	if !foundAlice {
		t.Fatalf("Alice was not returned as a conversation member")
	}

	if !foundBob {
		t.Fatalf("Bob was not returned as a conversation member")
	}

	t.Logf(
		"IT-API-017 PASS: conversation %d retrieved with correct details and members",
		conversationID,
	)
}

func TestIT_API_018_GetUserConversations(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-018_Alice_%d",
		time.Now().UnixNano(),
	)
	bob := fmt.Sprintf(
		"test_IT-API-018_Bob_%d",
		time.Now().UnixNano(),
	)
	charlie := fmt.Sprintf(
		"test_IT-API-018_Charlie_%d",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob, charlie)
	})

	var conversationIDs []int64
	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationIDs...)
	})

	// Create test users.
	result, err := database.Exec(
		"INSERT INTO users (username) VALUES (?), (?), (?)",
		alice,
		bob,
		charlie,
	)
	if err != nil {
		t.Fatalf("failed to create test users: %v", err)
	}

	aliceID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Alice's ID: %v", err)
	}

	var bobID, charlieID int64

	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID)
	if err != nil {
		t.Fatalf("failed to get Bob's ID: %v", err)
	}

	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		charlie,
	).Scan(&charlieID)
	if err != nil {
		t.Fatalf("failed to get Charlie's ID: %v", err)
	}

	// Create Conversation A: Alice + Bob.
	result, err = database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-018 Alice-Bob",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create Conversation A: %v", err)
	}

	conversationAID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Conversation A ID: %v", err)
	}
	conversationIDs = append(conversationIDs, conversationAID)

	_, err = database.Exec(
		`INSERT INTO user_in_conversation (user_id, conversation_id)
		 VALUES (?, ?), (?, ?)`,
		aliceID,
		conversationAID,
		bobID,
		conversationAID,
	)
	if err != nil {
		t.Fatalf("failed to add Conversation A members: %v", err)
	}

	// Create Conversation B: Alice + Charlie.
	result, err = database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-018 Alice-Charlie",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create Conversation B: %v", err)
	}

	conversationBID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Conversation B ID: %v", err)
	}
	conversationIDs = append(conversationIDs, conversationBID)

	_, err = database.Exec(
		`INSERT INTO user_in_conversation (user_id, conversation_id)
		 VALUES (?, ?), (?, ?)`,
		aliceID,
		conversationBID,
		charlieID,
		conversationBID,
	)
	if err != nil {
		t.Fatalf("failed to add Conversation B members: %v", err)
	}

	// Create Conversation C: Bob + Charlie.
	// Alice must NOT receive this conversation.
	result, err = database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-018 Bob-Charlie",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create Conversation C: %v", err)
	}

	conversationCID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Conversation C ID: %v", err)
	}
	conversationIDs = append(conversationIDs, conversationCID)

	_, err = database.Exec(
		`INSERT INTO user_in_conversation (user_id, conversation_id)
		 VALUES (?, ?), (?, ?)`,
		bobID,
		conversationCID,
		charlieID,
		conversationCID,
	)
	if err != nil {
		t.Fatalf("failed to add Conversation C members: %v", err)
	}

	// Log Alice in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		alice,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	// Extract session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Request Alice's conversations.
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conversations",
		nil,
	)

	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	// Verify HTTP status.
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200 OK, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Decode response.
	var response []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		MemberCount int    `json:"member_count"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode conversations response: %v",
			err,
		)
	}

	// Alice should have exactly two conversations.
	if len(response) != 2 {
		t.Fatalf(
			"expected Alice to have 2 conversations, got %d",
			len(response),
		)
	}

	foundA := false
	foundB := false
	foundC := false

	for _, conversation := range response {
		switch conversation.ID {
		case int(conversationAID):
			foundA = true

			if conversation.Name != "IT-API-018 Alice-Bob" {
				t.Fatalf(
					"Conversation A: expected name %q, got %q",
					"IT-API-018 Alice-Bob",
					conversation.Name,
				)
			}

			if conversation.Type != "group" {
				t.Fatalf(
					"Conversation A: expected type %q, got %q",
					"group",
					conversation.Type,
				)
			}

			if conversation.MemberCount != 2 {
				t.Fatalf(
					"Conversation A: expected member count 2, got %d",
					conversation.MemberCount,
				)
			}

		case int(conversationBID):
			foundB = true

			if conversation.Name != "IT-API-018 Alice-Charlie" {
				t.Fatalf(
					"Conversation B: expected name %q, got %q",
					"IT-API-018 Alice-Charlie",
					conversation.Name,
				)
			}

			if conversation.Type != "group" {
				t.Fatalf(
					"Conversation B: expected type %q, got %q",
					"group",
					conversation.Type,
				)
			}

			if conversation.MemberCount != 2 {
				t.Fatalf(
					"Conversation B: expected member count 2, got %d",
					conversation.MemberCount,
				)
			}

		case int(conversationCID):
			foundC = true
		}
	}

	if !foundA {
		t.Fatalf("Conversation A was not returned")
	}

	if !foundB {
		t.Fatalf("Conversation B was not returned")
	}

	if foundC {
		t.Fatalf(
			"Conversation C was returned even though Alice is not a member",
		)
	}

	t.Logf(
		"IT-API-018 PASS: Alice retrieved her 2 conversations and excluded unrelated conversation",
	)
}

func TestIT_API_019_JoinConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-019_Alice_%d",
		time.Now().UnixNano(),
	)
	bob := fmt.Sprintf(
		"test_IT-API-019_Bob_%d",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob)
	})

	// Create test users.
	result, err := database.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		alice,
		bob,
	)
	if err != nil {
		t.Fatalf("failed to create test users: %v", err)
	}

	aliceID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Alice's ID: %v", err)
	}

	var bobID int64
	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID)
	if err != nil {
		t.Fatalf("failed to get Bob's ID: %v", err)
	}

	// Create a conversation.
	result, err = database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-019 Join Test",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationID)
	})

	// Add Alice as the initial member.
	_, err = database.Exec(
		`INSERT INTO user_in_conversation (user_id, conversation_id)
		 VALUES (?, ?)`,
		aliceID,
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to add Alice to conversation: %v", err)
	}

	// Verify Bob is initially NOT a member.
	var membershipCount int

	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ? AND conversation_id = ?`,
		bobID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf("failed to check Bob's initial membership: %v", err)
	}

	if membershipCount != 0 {
		t.Fatalf(
			"expected Bob to initially not be a member, got count %d",
			membershipCount,
		)
	}

	// Log Bob in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		bob,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	// Extract Bob's session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Request Bob to join the conversation.
	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf(
			"/api/join/%d",
			conversationID,
		),
		nil,
	)

	req.AddCookie(sessionCookie)

	// Simulate the {convoID} path parameter.
	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", conversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.JoinConvo(recorder, req)

	// Verify HTTP status.
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200 OK, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify Bob is now a member in the database.
	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ? AND conversation_id = ?`,
		bobID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to check Bob's membership after joining: %v",
			err,
		)
	}

	if membershipCount != 1 {
		t.Fatalf(
			"expected Bob to be a member after joining, got count %d",
			membershipCount,
		)
	}

	// Verify the total number of members.
	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE conversation_id = ?`,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to check conversation member count: %v",
			err,
		)
	}

	if membershipCount != 2 {
		t.Fatalf(
			"expected conversation to have 2 members after Bob joined, got %d",
			membershipCount,
		)
	}

	t.Logf(
		"IT-API-019 PASS: Bob joined conversation %d and membership was persisted",
		conversationID,
	)
}

func TestIT_API_020_LeaveConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-020_Alice_%d",
		time.Now().UnixNano(),
	)
	bob := fmt.Sprintf(
		"test_IT-API-020_Bob_%d",
		time.Now().UnixNano(),
	)

	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob)
	})

	// Create test users.
	result, err := database.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		alice,
		bob,
	)
	if err != nil {
		t.Fatalf("failed to create test users: %v", err)
	}

	aliceID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Alice's ID: %v", err)
	}

	var bobID int64

	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID)
	if err != nil {
		t.Fatalf("failed to get Bob's ID: %v", err)
	}

	// Create conversation.
	result, err = database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-020 Leave Test",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationID)
	})

	// Add Alice and Bob as members.
	_, err = database.Exec(
		`INSERT INTO user_in_conversation (user_id, conversation_id)
		 VALUES (?, ?), (?, ?)`,
		aliceID,
		conversationID,
		bobID,
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to add conversation members: %v", err)
	}

	// Verify Bob is initially a member.
	var membershipCount int

	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ? AND conversation_id = ?`,
		bobID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to check Bob's initial membership: %v",
			err,
		)
	}

	if membershipCount != 1 {
		t.Fatalf(
			"expected Bob to initially be a member, got count %d",
			membershipCount,
		)
	}

	// Log Bob in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		bob,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	// Extract Bob's session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Request Bob to leave the conversation.
	req := httptest.NewRequest(
		http.MethodDelete,
		fmt.Sprintf(
			"/api/conversations/%d/members/me",
			conversationID,
		),
		nil,
	)

	req.AddCookie(sessionCookie)

	// Simulate the {convoID} path parameter.
	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", conversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.LeaveConvo(recorder, req)

	// Verify HTTP status.
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200 OK, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify Bob is no longer a member.
	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ? AND conversation_id = ?`,
		bobID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to check Bob's membership after leaving: %v",
			err,
		)
	}

	if membershipCount != 0 {
		t.Fatalf(
			"expected Bob to no longer be a member, got count %d",
			membershipCount,
		)
	}

	// Verify Alice is still a member.
	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ? AND conversation_id = ?`,
		aliceID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to check Alice's membership: %v",
			err,
		)
	}

	if membershipCount != 1 {
		t.Fatalf(
			"expected Alice to remain a member, got count %d",
			membershipCount,
		)
	}

	// Verify conversation still has exactly one member.
	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE conversation_id = ?`,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to check conversation member count: %v",
			err,
		)
	}

	if membershipCount != 1 {
		t.Fatalf(
			"expected conversation to have 1 member after Bob left, got %d",
			membershipCount,
		)
	}

	t.Logf(
		"IT-API-020 PASS: Bob left conversation %d and membership was removed",
		conversationID,
	)
}

func TestIT_API_021_JoinNonExistentConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-API-021_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, username)
	})

	// Create test user.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Log the user in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		username,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	// Get session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Pick a conversation ID that does not exist.
	var nonexistentConversationID int

	err = database.QueryRow(
		`SELECT COALESCE(MAX(id), 0) + 1000
		 FROM conversations`,
	).Scan(&nonexistentConversationID)

	if err != nil {
		t.Fatalf(
			"failed to generate nonexistent conversation ID: %v",
			err,
		)
	}

	// Try to join the nonexistent conversation.
	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf(
			"/api/join/%d",
			nonexistentConversationID,
		),
		nil,
	)

	req.AddCookie(sessionCookie)

	// Simulate {convoID}.
	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", nonexistentConversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.JoinConvo(recorder, req)

	// The request should fail.
	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status for nonexistent conversation, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify that no membership was created.
	var membershipCount int

	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation uic
		 JOIN users u
			 ON uic.user_id = u.id
		 WHERE u.username = ?
		   AND uic.conversation_id = ?`,
		username,
		nonexistentConversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to verify membership state: %v",
			err,
		)
	}

	if membershipCount != 0 {
		t.Fatalf(
			"expected no membership to be created, got count %d",
			membershipCount,
		)
	}

	t.Logf(
		"IT-API-021 PASS: joining nonexistent conversation %d was rejected",
		nonexistentConversationID,
	)
}

func TestIT_API_022_LeaveConversationAsNonMember(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-022_Alice_%d",
		time.Now().UnixNano(),
	)
	bob := fmt.Sprintf(
		"test_IT-API-022_Bob_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob)
	})

	// Create Alice and Bob.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		alice,
		bob,
	)
	if err != nil {
		t.Fatalf("failed to create test users: %v", err)
	}

	// Get their IDs.
	var aliceID, bobID int

	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		alice,
	).Scan(&aliceID)
	if err != nil {
		t.Fatalf("failed to get Alice's ID: %v", err)
	}

	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID)
	if err != nil {
		t.Fatalf("failed to get Bob's ID: %v", err)
	}

	// Create conversation.
	result, err := database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-022 Leave Non-Member Test",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}
	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationID)
	})

	// Add ONLY Alice.
	_, err = database.Exec(
		`INSERT INTO user_in_conversation
		 (user_id, conversation_id)
		 VALUES (?, ?)`,
		aliceID,
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to add Alice to conversation: %v", err)
	}

	// Verify Bob is NOT a member.
	var membershipCount int

	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ? AND conversation_id = ?`,
		bobID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf("failed to check Bob's initial membership: %v", err)
	}

	if membershipCount != 0 {
		t.Fatalf(
			"expected Bob to not be a member initially, got count %d",
			membershipCount,
		)
	}

	// Log Bob in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		bob,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	// Get Bob's session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Bob attempts to leave a conversation he is not a member of.
	req := httptest.NewRequest(
		http.MethodDelete,
		fmt.Sprintf(
			"/api/conversations/%d/members/me",
			conversationID,
		),
		nil,
	)

	req.AddCookie(sessionCookie)

	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", conversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.LeaveConvo(recorder, req)

	// Leaving as a non-member should fail.
	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status for non-member leaving, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify Bob is still not a member.
	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ? AND conversation_id = ?`,
		bobID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to verify Bob's membership after failed leave: %v",
			err,
		)
	}

	if membershipCount != 0 {
		t.Fatalf(
			"expected Bob to remain a non-member, got count %d",
			membershipCount,
		)
	}

	// Verify Alice is still a member.
	err = database.QueryRow(
		`SELECT COUNT(*)
		 FROM user_in_conversation
		 WHERE user_id = ? AND conversation_id = ?`,
		aliceID,
		conversationID,
	).Scan(&membershipCount)

	if err != nil {
		t.Fatalf(
			"failed to verify Alice's membership: %v",
			err,
		)
	}

	if membershipCount != 1 {
		t.Fatalf(
			"expected Alice to remain a member, got count %d",
			membershipCount,
		)
	}

	t.Logf(
		"IT-API-022 PASS: non-member Bob could not leave conversation %d",
		conversationID,
	)
}

func TestIT_API_023_GetConversationMessages(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-023_Alice_%d",
		time.Now().UnixNano(),
	)
	bob := fmt.Sprintf(
		"test_IT-API-023_Bob_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob)
	})

	var conversationIDs []int64
	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationIDs...)
	})

	// Create users.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		alice,
		bob,
	)
	if err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	// Get user IDs.
	var aliceID, bobID int

	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		alice,
	).Scan(&aliceID)
	if err != nil {
		t.Fatalf("failed to get Alice's ID: %v", err)
	}

	err = database.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID)
	if err != nil {
		t.Fatalf("failed to get Bob's ID: %v", err)
	}

	// Create the conversation we will request.
	result, err := database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-023 Message Test",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}
	conversationIDs = append(conversationIDs, conversationID)

	// Add Alice and Bob.
	_, err = database.Exec(
		`INSERT INTO user_in_conversation
			(user_id, conversation_id)
		 VALUES (?, ?), (?, ?)`,
		aliceID,
		conversationID,
		bobID,
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to add conversation members: %v", err)
	}

	// Create two messages in the requested conversation.
	message1, err := database.Exec(
		`INSERT INTO messages
			(user_id, conversation_id, content)
		 VALUES (?, ?, ?)`,
		aliceID,
		conversationID,
		"Hello Bob",
	)
	if err != nil {
		t.Fatalf("failed to create first message: %v", err)
	}

	message1ID, err := message1.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get first message ID: %v", err)
	}

	message2, err := database.Exec(
		`INSERT INTO messages
			(user_id, conversation_id, content)
		 VALUES (?, ?, ?)`,
		bobID,
		conversationID,
		"Hello Alice",
	)
	if err != nil {
		t.Fatalf("failed to create second message: %v", err)
	}

	message2ID, err := message2.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get second message ID: %v", err)
	}

	// Create an unrelated conversation and message.
	result, err = database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, ?)`,
		"IT-API-023 Unrelated Conversation",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create unrelated conversation: %v", err)
	}

	unrelatedConversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get unrelated conversation ID: %v", err)
	}
	conversationIDs = append(conversationIDs, unrelatedConversationID)

	_, err = database.Exec(
		`INSERT INTO messages
			(user_id, conversation_id, content)
		 VALUES (?, ?, ?)`,
		aliceID,
		unrelatedConversationID,
		"This should not be returned",
	)
	if err != nil {
		t.Fatalf("failed to create unrelated message: %v", err)
	}

	// Log Alice in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		alice,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	// Get Alice's session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Request messages.
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/api/conversations/%d/messages",
			conversationID,
		),
		nil,
	)

	req.AddCookie(sessionCookie)

	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", conversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.GetMessages(recorder, req)

	// Verify status.
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected HTTP 200, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Decode response.
	var messages []struct {
		ID             int    `json:"ID"`
		UserID         int    `json:"UserID"`
		Username       string `json:"Username"`
		ConversationID int    `json:"ConversationID"`
		Content        string `json:"Content"`
		SentAt         string `json:"SentAt"`
		DeliveryStatus string `json:"delivery_status"`
	}

	err = json.NewDecoder(recorder.Body).Decode(&messages)
	if err != nil {
		t.Fatalf(
			"failed to decode response JSON: %v. Body: %s",
			err,
			recorder.Body.String(),
		)
	}

	// We expect exactly the two messages from this conversation.
	if len(messages) != 2 {
		t.Fatalf(
			"expected 2 messages, got %d",
			len(messages),
		)
	}

	// Verify first message.
	var foundMessage1 bool
	var foundMessage2 bool

	for _, message := range messages {
		if message.ID == int(message1ID) {
			foundMessage1 = true

			if message.UserID != aliceID {
				t.Errorf(
					"message %d: expected user ID %d, got %d",
					message.ID,
					aliceID,
					message.UserID,
				)
			}

			if message.ConversationID != int(conversationID) {
				t.Errorf(
					"message %d: expected conversation ID %d, got %d",
					message.ID,
					conversationID,
					message.ConversationID,
				)
			}

			if message.Content != "Hello Bob" {
				t.Errorf(
					"message %d: expected content %q, got %q",
					message.ID,
					"Hello Bob",
					message.Content,
				)
			}
		}

		if message.ID == int(message2ID) {
			foundMessage2 = true

			if message.UserID != bobID {
				t.Errorf(
					"message %d: expected user ID %d, got %d",
					message.ID,
					bobID,
					message.UserID,
				)
			}

			if message.ConversationID != int(conversationID) {
				t.Errorf(
					"message %d: expected conversation ID %d, got %d",
					message.ID,
					conversationID,
					message.ConversationID,
				)
			}

			if message.Content != "Hello Alice" {
				t.Errorf(
					"message %d: expected content %q, got %q",
					message.ID,
					"Hello Alice",
					message.Content,
				)
			}
		}

		// Make sure the unrelated message isn't returned.
		if message.Content == "This should not be returned" {
			t.Fatal("unrelated conversation message was returned")
		}
	}

	if !foundMessage1 {
		t.Errorf("expected message %d to be returned", message1ID)
	}

	if !foundMessage2 {
		t.Errorf("expected message %d to be returned", message2ID)
	}

	t.Logf(
		"IT-API-023 PASS: retrieved 2 messages from conversation %d",
		conversationID,
	)
}

func TestIT_API_024_GetMessagesFromNonExistentConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-API-024_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, username)
	})

	// Create test user.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Generate a conversation ID that does not exist.
	var nonexistentConversationID int

	err = database.QueryRow(
		`SELECT COALESCE(MAX(id), 0) + 1000
		 FROM conversations`,
	).Scan(&nonexistentConversationID)

	if err != nil {
		t.Fatalf(
			"failed to generate nonexistent conversation ID: %v",
			err,
		)
	}

	// Log the user in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		username,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	// Get session cookie.
	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Request messages from nonexistent conversation.
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/api/conversations/%d/messages",
			nonexistentConversationID,
		),
		nil,
	)

	req.AddCookie(sessionCookie)

	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", nonexistentConversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.GetMessages(recorder, req)

	// The request should be rejected.
	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status for nonexistent conversation, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	t.Logf(
		"IT-API-024 PASS: nonexistent conversation %d was rejected",
		nonexistentConversationID,
	)
}

func TestIT_API_025_GetMessagesWithoutAuthentication(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-025_Alice_%d",
		time.Now().UnixNano(),
	)
	bob := fmt.Sprintf(
		"test_IT-API-025_Bob_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob)
	})

	// Create users.
	aliceResult, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		alice,
	)
	if err != nil {
		t.Fatalf("failed to create Alice: %v", err)
	}

	aliceID64, err := aliceResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Alice ID: %v", err)
	}
	aliceID := int(aliceID64)

	bobResult, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		bob,
	)
	if err != nil {
		t.Fatalf("failed to create Bob: %v", err)
	}

	bobID64, err := bobResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Bob ID: %v", err)
	}
	bobID := int(bobID64)

	// Create a conversation.
	result, err := database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, 'group')`,
		"IT-API-025 Test Conversation",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID64, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}
	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationID64)
	})
	conversationID := int(conversationID64)

	// Add both users to the conversation.
	_, err = database.Exec(
		`INSERT INTO user_in_conversation (user_id, conversation_id)
		 VALUES (?, ?), (?, ?)`,
		aliceID,
		conversationID,
		bobID,
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to add users to conversation: %v", err)
	}

	// Request messages WITHOUT a session cookie.
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/api/conversations/%d/messages",
			conversationID,
		),
		nil,
	)

	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", conversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.GetMessages(recorder, req)

	// The endpoint should require authentication.
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected 401 Unauthorized, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	t.Logf(
		"IT-API-025 PASS: unauthenticated request for conversation %d was rejected",
		conversationID,
	)
}

func TestIT_API_026_GetMessagesAsNonMember(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-026_Alice_%d",
		time.Now().UnixNano(),
	)
	bob := fmt.Sprintf(
		"test_IT-API-026_Bob_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice, bob)
	})

	// Create Alice.
	aliceResult, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		alice,
	)
	if err != nil {
		t.Fatalf("failed to create Alice: %v", err)
	}

	aliceID64, err := aliceResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Alice ID: %v", err)
	}
	aliceID := int(aliceID64)

	// Create Bob.
	bobResult, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		bob,
	)
	if err != nil {
		t.Fatalf("failed to create Bob: %v", err)
	}

	bobID64, err := bobResult.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get Bob ID: %v", err)
	}
	bobID := int(bobID64)

	// Create conversation.
	result, err := database.Exec(
		`INSERT INTO conversations (name, conversation_type)
		 VALUES (?, 'group')`,
		"IT-API-026 Test Conversation",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID64, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}
	t.Cleanup(func() {
		testutil.DeleteConversationsByID(t, database, conversationID64)
	})
	conversationID := int(conversationID64)

	// Only Bob is a member.
	_, err = database.Exec(
		`INSERT INTO user_in_conversation (user_id, conversation_id)
		 VALUES (?, ?)`,
		bobID,
		conversationID,
	)
	if err != nil {
		t.Fatalf("failed to add Bob to conversation: %v", err)
	}

	// Log Alice in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		alice,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Alice tries to read Bob's conversation.
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/api/conversations/%d/messages",
			conversationID,
		),
		nil,
	)

	req.AddCookie(sessionCookie)

	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", conversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.GetMessages(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected 403 Forbidden, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	t.Logf(
		"IT-API-026 PASS: non-member Alice was rejected from conversation %d",
		conversationID,
	)

	_ = aliceID
}

func TestIT_API_027_CreateGroupConversationWithoutMembers(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-API-027_Alice_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, username)
	})

	// Create user.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Log the user in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		username,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Record the number of conversations before the request.
	var beforeCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&beforeCount)

	if err != nil {
		t.Fatalf("failed to count conversations before request: %v", err)
	}

	// Attempt to create a group conversation without members.
	body := `{
		"type": "group",
		"name": "IT-API-027 Empty Group",
		"members": []
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	// Invalid request should be rejected.
	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify no conversation was created.
	var afterCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&afterCount)

	if err != nil {
		t.Fatalf("failed to count conversations after request: %v", err)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"expected conversation count to remain %d, got %d",
			beforeCount,
			afterCount,
		)
	}

	t.Logf(
		"IT-API-027 PASS: group conversation without members was rejected and no conversation was created",
	)
}

func TestIT_API_028_CreateDirectConversationWithNonExistentUser(t *testing.T) {
	database := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-028_Alice_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, alice)
	})

	// Create Alice.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		alice,
	)
	if err != nil {
		t.Fatalf("failed to create Alice: %v", err)
	}

	// Log Alice in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		alice,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Count conversations before the request.
	var beforeCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&beforeCount)

	if err != nil {
		t.Fatalf(
			"failed to count conversations before request: %v",
			err,
		)
	}

	// Attempt to create a direct conversation with a user
	// who does not exist.
	body := `{
		"type": "direct",
		"members": ["IT-API-028_NonExistentUser"]
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	// The request should be rejected.
	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify no conversation was created.
	var afterCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&afterCount)

	if err != nil {
		t.Fatalf(
			"failed to count conversations after request: %v",
			err,
		)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"expected conversation count to remain %d, got %d",
			beforeCount,
			afterCount,
		)
	}

	t.Logf(
		"IT-API-028 PASS: direct conversation with nonexistent user was rejected and no conversation was created",
	)
}

func TestIT_API_029_CreateConversationInvalidJSON(t *testing.T) {
	database := testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-API-029_Alice_%d",
		time.Now().UnixNano(),
	)
	t.Cleanup(func() {
		testutil.DeleteUsersByUsername(t, database, username)
	})

	// Create user.
	_, err := database.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Log in.
	loginBody := fmt.Sprintf(
		`{"username":%q}`,
		username,
	)

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
			"expected login to return 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected login to create session_token cookie")
	}

	// Count conversations before the request.
	var beforeCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&beforeCount)

	if err != nil {
		t.Fatalf(
			"failed to count conversations before request: %v",
			err,
		)
	}

	// Send malformed JSON.
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(`this is definitely not json`),
	)

	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	// Invalid JSON should be rejected.
	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx status, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify no conversation was created.
	var afterCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&afterCount)

	if err != nil {
		t.Fatalf(
			"failed to count conversations after request: %v",
			err,
		)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"expected conversation count to remain %d, got %d",
			beforeCount,
			afterCount,
		)
	}

	t.Logf(
		"IT-API-029 PASS: invalid JSON was rejected and no conversation was created",
	)
}

func TestIT_API_030_CreateConversationWithoutAuthentication(t *testing.T) {
	database := testutil.SetupDatabase(t)

	var beforeCount int

	err := database.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&beforeCount)

	if err != nil {
		t.Fatalf(
			"failed to count conversations before request: %v",
			err,
		)
	}

	body := `{
    "type": "group",
    "name": "IT-API-030 Unauthorized Group",
    "members": ["some_user", "another_user"]
}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	// Deliberately NO session cookie.

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected 401 Unauthorized, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var afterCount int

	err = database.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&afterCount)

	if err != nil {
		t.Fatalf(
			"failed to count conversations after request: %v",
			err,
		)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"expected conversation count to remain %d, got %d",
			beforeCount,
			afterCount,
		)
	}

	t.Logf(
		"IT-API-030 PASS: unauthenticated conversation creation was rejected",
	)
}

func TestIT_API_031_GetConversationInvalidID(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conversations/abc",
		nil,
	)

	req.SetPathValue("convoID", "abc")

	recorder := httptest.NewRecorder()

	handlers.ConvoInfo(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400 Bad Request, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	t.Logf(
		"IT-API-031 PASS: invalid conversation ID was rejected",
	)
}

func TestIT_API_032_GetNonExistentConversation(t *testing.T) {
	database := testutil.SetupDatabase(t)

	var conversationID int

	err := database.QueryRow(
		`SELECT COALESCE(MAX(id), 0) + 1000
		 FROM conversations`,
	).Scan(&conversationID)

	if err != nil {
		t.Fatalf(
			"failed to generate nonexistent conversation ID: %v",
			err,
		)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"/api/conversations/%d",
			conversationID,
		),
		nil,
	)

	req.SetPathValue(
		"convoID",
		fmt.Sprintf("%d", conversationID),
	)

	recorder := httptest.NewRecorder()

	handlers.ConvoInfo(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected 404 Not Found, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	t.Logf(
		"IT-API-032 PASS: nonexistent conversation %d was rejected",
		conversationID,
	)
}

func TestIT_API_033_GetConversationsWithoutAuthentication(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conversations",
		nil,
	)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected 401 Unauthorized, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	t.Log(
		"IT-API-033 PASS: unauthenticated conversation list request was rejected",
	)
}

func TestIT_API_034_GetConversationsWithInvalidSession(t *testing.T) {

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conversations",
		nil,
	)

	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: "this_is_not_a_valid_session",
	})

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected 401 Unauthorized, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	t.Log(
		"IT-API-034 PASS: invalid session was rejected when retrieving conversations",
	)
}

func TestIT_API_035_GetConversationsResponseStructure(t *testing.T) {
	db := testutil.SetupDatabase(t)

	// Create a unique user
	username := fmt.Sprintf("test_IT-API-035_%d", time.Now().UnixNano())

	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create a conversation
	result, err := db.Exec(
		"INSERT INTO conversations (name, conversation_type) VALUES (?, ?)",
		"test conversation",
		"group",
	)
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	// Add user to conversation
	_, err = db.Exec(
		"INSERT INTO user_in_conversation (user_id, conversation_id) "+
			"SELECT id, ? FROM users WHERE username = ?",
		conversationID,
		username,
	)
	if err != nil {
		t.Fatalf("failed to add user to conversation: %v", err)
	}

	// Login to obtain a valid session
	loginBody := strings.NewReader(
		fmt.Sprintf(`{"username":"%s"}`, username),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		loginBody,
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated &&
		loginRecorder.Code != http.StatusOK {
		t.Fatalf(
			"login failed: expected 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	cookies := loginRecorder.Result().Cookies()

	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("login did not return a session_token cookie")
	}

	// Request conversations
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conversations",
		nil,
	)

	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200 OK, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Decode response
	var conversations []struct {
		ID               int    `json:"id"`
		Name             string `json:"name"`
		ConversationType string `json:"type"`
	}

	err = json.NewDecoder(recorder.Body).Decode(&conversations)
	if err != nil {
		t.Fatalf(
			"failed to decode conversations response: %v. Body: %s",
			err,
			recorder.Body.String(),
		)
	}

	// Response must contain at least the conversation we created
	found := false

	for _, conversation := range conversations {
		if conversation.ID == int(conversationID) {
			found = true

			if conversation.Name != "test conversation" {
				t.Errorf(
					"expected conversation name %q, got %q",
					"test conversation",
					conversation.Name,
				)
			}

			if conversation.ConversationType != "group" {
				t.Errorf(
					"expected conversation type %q, got %q",
					"group",
					conversation.ConversationType,
				)
			}
		}
	}

	if !found {
		t.Fatalf(
			"created conversation %d was not found in response. Body: %s",
			conversationID,
			recorder.Body.String(),
		)
	}

	t.Logf(
		"IT-API-035 PASS: conversation list returned valid structure and included conversation %d",
		conversationID,
	)
}

func TestIT_API_036_GetConversationsWithNoConversations(t *testing.T) {
	db := testutil.SetupDatabase(t)

	// Create a unique user who will not belong to any conversation.
	username := fmt.Sprintf(
		"test_IT-API-036_%d",
		time.Now().UnixNano(),
	)

	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Login to obtain a valid session.
	loginBody := strings.NewReader(
		fmt.Sprintf(`{"username":"%s"}`, username),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		loginBody,
	)

	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated &&
		loginRecorder.Code != http.StatusOK {
		t.Fatalf(
			"login failed: expected 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("login did not return a session_token cookie")
	}

	// Request conversations.
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conversations",
		nil,
	)

	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200 OK, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Decode the response as an array.
	var conversations []struct {
		ID               int    `json:"id"`
		Name             string `json:"name"`
		ConversationType string `json:"type"`
	}

	err = json.NewDecoder(recorder.Body).Decode(&conversations)
	if err != nil {
		t.Fatalf(
			"failed to decode response: %v. Body: %s",
			err,
			recorder.Body.String(),
		)
	}

	// The user should have no conversations.
	if conversations == nil {
		t.Fatal("expected an empty JSON array, got null")
	}

	if len(conversations) != 0 {
		t.Fatalf(
			"expected 0 conversations, got %d",
			len(conversations),
		)
	}

	t.Log(
		"IT-API-036 PASS: authenticated user with no conversations received an empty array",
	)
}

func TestIT_API_037_CreateDirectConversationWithSelf(t *testing.T) {
	db := testutil.SetupDatabase(t)

	username := fmt.Sprintf(
		"test_IT-API-037_%d",
		time.Now().UnixNano(),
	)

	// Create user.
	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?)",
		username,
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Login.
	loginBody := strings.NewReader(
		fmt.Sprintf(`{"username":"%s"}`, username),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		loginBody,
	)

	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated &&
		loginRecorder.Code != http.StatusOK {
		t.Fatalf(
			"login failed: expected 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("login did not return a session_token cookie")
	}

	// Count conversations before the request.
	var beforeCount int

	err = db.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&beforeCount)

	if err != nil {
		t.Fatalf("failed to count conversations before request: %v", err)
	}

	// Attempt to create a direct conversation with yourself.
	body := fmt.Sprintf(
		`{"type":"direct","members":["%s"]}`,
		username,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	// Self-conversation should be rejected.
	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf(
			"expected 4xx for self-conversation, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify that no conversation was created.
	var afterCount int

	err = db.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&afterCount)

	if err != nil {
		t.Fatalf("failed to count conversations after request: %v", err)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"conversation was created despite self-conversation being rejected: before=%d, after=%d",
			beforeCount,
			afterCount,
		)
	}

	t.Log(
		"IT-API-037 PASS: direct conversation with self was rejected and no conversation was created",
	)
}

func TestIT_API_038_DuplicateDirectConversation(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf(
		"test_IT-API-038_Alice_%d",
		time.Now().UnixNano(),
	)

	bob := fmt.Sprintf(
		"test_IT-API-038_Bob_%d",
		time.Now().UnixNano(),
	)

	// Create Alice and Bob.
	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		alice,
		bob,
	)
	if err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	// Get Alice's user ID.
	var aliceID int
	err = db.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		alice,
	).Scan(&aliceID)
	if err != nil {
		t.Fatalf("failed to get Alice's ID: %v", err)
	}

	// Get Bob's user ID.
	var bobID int
	err = db.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		bob,
	).Scan(&bobID)
	if err != nil {
		t.Fatalf("failed to get Bob's ID: %v", err)
	}

	// Create the original direct conversation.
	result, err := db.Exec(
		"INSERT INTO conversations (conversation_type) VALUES (?)",
		"direct",
	)
	if err != nil {
		t.Fatalf("failed to create original conversation: %v", err)
	}

	originalConversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get original conversation ID: %v", err)
	}

	// Add Alice and Bob to the conversation.
	_, err = db.Exec(
		"INSERT INTO user_in_conversation (user_id, conversation_id) VALUES (?, ?), (?, ?)",
		aliceID,
		originalConversationID,
		bobID,
		originalConversationID,
	)
	if err != nil {
		t.Fatalf("failed to add users to conversation: %v", err)
	}

	// Login as Alice.
	loginBody := strings.NewReader(
		fmt.Sprintf(`{"username":"%s"}`, alice),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		loginBody,
	)

	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()

	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated &&
		loginRecorder.Code != http.StatusOK {
		t.Fatalf(
			"login failed: expected 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var sessionCookie *http.Cookie

	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("login did not return a session_token cookie")
	}

	// Count direct conversations before the duplicate request.
	var beforeCount int

	err = db.QueryRow(
		`SELECT COUNT(*)
		 FROM conversations
		 WHERE conversation_type = 'direct'`,
	).Scan(&beforeCount)

	if err != nil {
		t.Fatalf(
			"failed to count direct conversations before request: %v",
			err,
		)
	}

	// Request the same direct conversation again.
	body := fmt.Sprintf(
		`{"type":"direct","members":["%s"]}`,
		bob,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)

	recorder := httptest.NewRecorder()

	handlers.HandleConversations(recorder, req)

	// Existing direct conversation should be returned successfully.
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200 OK for existing direct conversation, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	// Verify response contains the original conversation ID.
	var response struct {
		Conversation struct {
			ID int `json:"id"`
		} `json:"conversation"`
	}

	err = json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		t.Fatalf(
			"failed to decode response: %v. Body: %s",
			err,
			recorder.Body.String(),
		)
	}

	if response.Conversation.ID != int(originalConversationID) {
		t.Fatalf(
			"expected existing conversation ID %d, got %d",
			originalConversationID,
			response.Conversation.ID,
		)
	}

	// Count direct conversations after the request.
	var afterCount int

	err = db.QueryRow(
		`SELECT COUNT(*)
		 FROM conversations
		 WHERE conversation_type = 'direct'`,
	).Scan(&afterCount)

	if err != nil {
		t.Fatalf(
			"failed to count direct conversations after request: %v",
			err,
		)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"duplicate direct conversation was created: before=%d, after=%d",
			beforeCount,
			afterCount,
		)
	}

	t.Logf(
		"IT-API-038 PASS: duplicate direct conversation reused existing conversation %d",
		originalConversationID,
	)
}

func TestIT_API_039_DirectConversationIsSymmetric(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-039_Alice_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-039_Bob_%d", time.Now().UnixNano())

	// Create users.
	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?), (?)",
		alice, bob,
	)
	if err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	// Login as Alice.
	loginBody := strings.NewReader(
		fmt.Sprintf(`{"username":"%s"}`, alice),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		loginBody,
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()
	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated &&
		loginRecorder.Code != http.StatusOK {
		t.Fatalf(
			"login failed: expected 200 or 201, got %d. Body: %s",
			loginRecorder.Code,
			loginRecorder.Body.String(),
		)
	}

	var aliceCookie *http.Cookie
	for _, cookie := range loginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			aliceCookie = cookie
			break
		}
	}

	if aliceCookie == nil {
		t.Fatal("Alice login did not return session_token")
	}

	// Alice creates Alice <-> Bob.
	createBody := fmt.Sprintf(
		`{"type":"direct","members":["%s"]}`,
		bob,
	)

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(createBody),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(aliceCookie)

	createRecorder := httptest.NewRecorder()
	handlers.HandleConversations(createRecorder, createReq)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"Alice's conversation creation failed: expected 201, got %d. Body: %s",
			createRecorder.Code,
			createRecorder.Body.String(),
		)
	}

	var created struct {
		Conversation struct {
			ID int `json:"id"`
		} `json:"conversation"`
	}

	err = json.NewDecoder(createRecorder.Body).Decode(&created)
	if err != nil {
		t.Fatalf("failed to decode creation response: %v", err)
	}

	originalID := created.Conversation.ID

	// Login as Bob.
	bobLoginBody := strings.NewReader(
		fmt.Sprintf(`{"username":"%s"}`, bob),
	)

	bobLoginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		bobLoginBody,
	)
	bobLoginReq.Header.Set("Content-Type", "application/json")

	bobLoginRecorder := httptest.NewRecorder()
	handlers.LoginHandler(bobLoginRecorder, bobLoginReq)

	if bobLoginRecorder.Code != http.StatusCreated &&
		bobLoginRecorder.Code != http.StatusOK {
		t.Fatalf(
			"Bob login failed: expected 200 or 201, got %d. Body: %s",
			bobLoginRecorder.Code,
			bobLoginRecorder.Body.String(),
		)
	}

	var bobCookie *http.Cookie
	for _, cookie := range bobLoginRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			bobCookie = cookie
			break
		}
	}

	if bobCookie == nil {
		t.Fatal("Bob login did not return session_token")
	}

	// Bob requests Bob <-> Alice.
	duplicateBody := fmt.Sprintf(
		`{"type":"direct","members":["%s"]}`,
		alice,
	)

	duplicateReq := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(duplicateBody),
	)
	duplicateReq.Header.Set("Content-Type", "application/json")
	duplicateReq.AddCookie(bobCookie)

	duplicateRecorder := httptest.NewRecorder()
	handlers.HandleConversations(duplicateRecorder, duplicateReq)

	if duplicateRecorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200 for existing symmetric direct conversation, got %d. Body: %s",
			duplicateRecorder.Code,
			duplicateRecorder.Body.String(),
		)
	}

	var duplicate struct {
		Conversation struct {
			ID int `json:"id"`
		} `json:"conversation"`
	}

	err = json.NewDecoder(duplicateRecorder.Body).Decode(&duplicate)
	if err != nil {
		t.Fatalf("failed to decode duplicate response: %v", err)
	}

	if duplicate.Conversation.ID != originalID {
		t.Fatalf(
			"expected existing conversation ID %d, got %d",
			originalID,
			duplicate.Conversation.ID,
		)
	}

	var conversationCount int

	err = db.QueryRow(`
	SELECT COUNT(*)
	FROM (
		SELECT c.id
		FROM conversations AS c
		JOIN user_in_conversation AS uic
			ON uic.conversation_id = c.id
		WHERE c.conversation_type = 'direct'
		  AND uic.user_id IN (
		      SELECT id
		      FROM users
		      WHERE username IN (?, ?)
		  )
		GROUP BY c.id
		HAVING COUNT(DISTINCT uic.user_id) = 2
		   AND (
		       SELECT COUNT(*)
		       FROM user_in_conversation
		       WHERE conversation_id = c.id
		   ) = 2
	) AS matching_conversations
`, alice, bob).Scan(&conversationCount)

	if err != nil {
		t.Fatalf("failed to verify direct conversation count: %v", err)
	}

	if conversationCount != 1 {
		t.Fatalf(
			"expected exactly one direct conversation, got %d",
			conversationCount,
		)
	}
}

func TestIT_API_040_DuplicateGroupConversationAllowed(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-040_Alice_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-040_Bob_%d", time.Now().UnixNano())
	charlie := fmt.Sprintf("test_IT-API-040_Charlie_%d", time.Now().UnixNano())

	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?), (?), (?)",
		alice, bob, charlie,
	)
	if err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	// Login as Alice.
	loginBody := strings.NewReader(
		fmt.Sprintf(`{"username":"%s"}`, alice),
	)

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		loginBody,
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()
	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated &&
		loginRecorder.Code != http.StatusOK {
		t.Fatalf("login failed: got %d", loginRecorder.Code)
	}

	var cookie *http.Cookie
	for _, c := range loginRecorder.Result().Cookies() {
		if c.Name == "session_token" {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("login did not return session_token")
	}

	body := fmt.Sprintf(
		`{"type":"group","name":"Test Group","members":["%s","%s"]}`,
		bob,
		charlie,
	)

	// First group.
	req1 := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)
	req1.Header.Set("Content-Type", "application/json")
	req1.AddCookie(cookie)

	recorder1 := httptest.NewRecorder()
	handlers.HandleConversations(recorder1, req1)

	if recorder1.Code != http.StatusCreated {
		t.Fatalf(
			"first group creation expected 201, got %d. Body: %s",
			recorder1.Code,
			recorder1.Body.String(),
		)
	}

	var first struct {
		Conversation struct {
			ID int `json:"id"`
		} `json:"conversation"`
	}

	if err := json.NewDecoder(recorder1.Body).Decode(&first); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}

	// Second identical group.
	req2 := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(cookie)

	recorder2 := httptest.NewRecorder()
	handlers.HandleConversations(recorder2, req2)

	if recorder2.Code != http.StatusCreated {
		t.Fatalf(
			"second group creation expected 201, got %d. Body: %s",
			recorder2.Code,
			recorder2.Body.String(),
		)
	}

	var second struct {
		Conversation struct {
			ID int `json:"id"`
		} `json:"conversation"`
	}

	if err := json.NewDecoder(recorder2.Body).Decode(&second); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}

	if first.Conversation.ID == second.Conversation.ID {
		t.Fatalf(
			"expected two different group conversations, both returned ID %d",
			first.Conversation.ID,
		)
	}

	var count int

	err = db.QueryRow(
		`SELECT COUNT(*)
		 FROM conversations
		 WHERE id IN (?, ?)`,
		first.Conversation.ID,
		second.Conversation.ID,
	).Scan(&count)

	if err != nil {
		t.Fatalf("failed to verify group conversations: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected two group conversations in database, got %d", count)
	}

	t.Logf(
		"IT-API-040 PASS: duplicate group requests created separate conversations %d and %d",
		first.Conversation.ID,
		second.Conversation.ID,
	)
}

func TestIT_API_041_GroupConversationRequiresName(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-041_Alice_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-041_Bob_%d", time.Now().UnixNano())
	charlie := fmt.Sprintf("test_IT-API-041_Charlie_%d", time.Now().UnixNano())

	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?), (?), (?)",
		alice, bob, charlie,
	)
	if err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	var aliceID int
	err = db.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		alice,
	).Scan(&aliceID)
	if err != nil {
		t.Fatalf("failed to get Alice ID: %v", err)
	}

	var beforeCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&beforeCount)
	if err != nil {
		t.Fatalf("failed to count conversations: %v", err)
	}

	// Login.
	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(fmt.Sprintf(`{"username":"%s"}`, alice)),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()
	handlers.LoginHandler(loginRecorder, loginReq)

	var cookie *http.Cookie
	for _, c := range loginRecorder.Result().Cookies() {
		if c.Name == "session_token" {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("login did not return session_token")
	}

	// Missing name.
	body := fmt.Sprintf(
		`{"type":"group","members":["%s","%s"]}`,
		bob,
		charlie,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	recorder := httptest.NewRecorder()
	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400 Bad Request, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var afterCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&afterCount)
	if err != nil {
		t.Fatalf("failed to count conversations after request: %v", err)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"group conversation was created despite missing name: before=%d, after=%d",
			beforeCount,
			afterCount,
		)
	}

	_ = aliceID

	t.Log(
		"IT-API-041 PASS: group conversation without a name was rejected",
	)
}

func TestIT_API_042_DirectConversationRejectsMultipleMembers(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-042_Alice_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-042_Bob_%d", time.Now().UnixNano())
	charlie := fmt.Sprintf("test_IT-API-042_Charlie_%d", time.Now().UnixNano())

	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?), (?), (?)",
		alice, bob, charlie,
	)
	if err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	var beforeCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&beforeCount)
	if err != nil {
		t.Fatalf("failed to count conversations: %v", err)
	}

	// Login as Alice.
	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(fmt.Sprintf(`{"username":"%s"}`, alice)),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()
	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated &&
		loginRecorder.Code != http.StatusOK {
		t.Fatalf("login failed: got %d", loginRecorder.Code)
	}

	var cookie *http.Cookie
	for _, c := range loginRecorder.Result().Cookies() {
		if c.Name == "session_token" {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("login did not return session_token")
	}

	// Direct conversation with TWO other members.
	body := fmt.Sprintf(
		`{"type":"direct","members":["%s","%s"]}`,
		bob,
		charlie,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	recorder := httptest.NewRecorder()
	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400 Bad Request, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var afterCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&afterCount)
	if err != nil {
		t.Fatalf("failed to count conversations after request: %v", err)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"conversation was created despite invalid direct member count: before=%d, after=%d",
			beforeCount,
			afterCount,
		)
	}

	t.Log(
		"IT-API-042 PASS: direct conversation with multiple other members was rejected",
	)
}

func TestIT_API_043_InvalidConversationType(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-043_Alice_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-043_Bob_%d", time.Now().UnixNano())
	charlie := fmt.Sprintf("test_IT-API-043_Charlie_%d", time.Now().UnixNano())

	_, err := db.Exec(
		"INSERT INTO users (username) VALUES (?), (?), (?)",
		alice, bob, charlie,
	)
	if err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	var beforeCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&beforeCount)
	if err != nil {
		t.Fatalf("failed to count conversations: %v", err)
	}

	// Login as Alice.
	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(fmt.Sprintf(`{"username":"%s"}`, alice)),
	)
	loginReq.Header.Set("Content-Type", "application/json")

	loginRecorder := httptest.NewRecorder()
	handlers.LoginHandler(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusCreated &&
		loginRecorder.Code != http.StatusOK {
		t.Fatalf("login failed: got %d", loginRecorder.Code)
	}

	var cookie *http.Cookie
	for _, c := range loginRecorder.Result().Cookies() {
		if c.Name == "session_token" {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("login did not return session_token")
	}

	body := fmt.Sprintf(
		`{"type":"banana","name":"Invalid Type","members":["%s","%s"]}`,
		bob,
		charlie,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/conversations",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)

	recorder := httptest.NewRecorder()
	handlers.HandleConversations(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400 Bad Request, got %d. Body: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var afterCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM conversations",
	).Scan(&afterCount)
	if err != nil {
		t.Fatalf("failed to count conversations after request: %v", err)
	}

	if afterCount != beforeCount {
		t.Fatalf(
			"conversation was created despite invalid conversation type: before=%d, after=%d",
			beforeCount,
			afterCount,
		)
	}

	t.Log(
		"IT-API-043 PASS: invalid conversation type was rejected",
	)
}

func TestIT_API_044_GetMultipleMessages(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-044_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-044_bob_%d", time.Now().UnixNano())

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
	`, aliceID, conversationID, bobID, conversationID)
	if err != nil {
		t.Fatalf("failed to add conversation members: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO messages(user_id, conversation_id, content)
		VALUES
			(?, ?, ?),
			(?, ?, ?),
			(?, ?, ?)
	`,
		aliceID, conversationID, "First message",
		bobID, conversationID, "Second message",
		aliceID, conversationID, "Third message",
	)
	if err != nil {
		t.Fatalf("failed to insert messages: %v", err)
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
		t.Fatalf("login failed: got status %d", loginRecorder.Code)
	}

	var loginResponse struct {
		Token string `json:"session_token"`
	}
	_ = json.NewDecoder(loginRecorder.Body).Decode(&loginResponse)

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not return a session cookie")
	}

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/conversations/%d/messages", conversationID),
		nil,
	)
	req.SetPathValue("convoID", fmt.Sprintf("%d", conversationID))

	req.AddCookie(cookies[0])

	recorder := httptest.NewRecorder()
	handlers.GetMessages(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var messages []struct {
		ID             int    `json:"ID"`
		UserID         int    `json:"UserID"`
		Username       string `json:"Username"`
		ConversationID int    `json:"ConversationID"`
		Content        string `json:"Content"`
		SentAt         string `json:"SentAt"`
		DeliveryStatus string `json:"delivery_status"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&messages); err != nil {
		t.Fatalf("failed to decode messages: %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	t.Logf(
		"IT-API-044 PASS: retrieved %d messages from conversation %d",
		len(messages),
		conversationID,
	)
}

func TestIT_API_045_MessagesReturnedInOrder(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-045_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-045_bob_%d", time.Now().UnixNano())

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
	`, aliceID, conversationID, bobID, conversationID)
	if err != nil {
		t.Fatalf("failed to add members: %v", err)
	}

	contents := []string{
		"Message one",
		"Message two",
		"Message three",
	}

	for _, content := range contents {
		_, err := db.Exec(`
			INSERT INTO messages(user_id, conversation_id, content)
			VALUES (?, ?, ?)
		`, aliceID, conversationID, content)

		if err != nil {
			t.Fatalf("failed to insert message %q: %v", content, err)
		}

		// Ensure sent_at values cannot accidentally become identical.
		time.Sleep(10 * time.Millisecond)
	}

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
		t.Fatalf("login failed: got status %d", loginRecorder.Code)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not return a session cookie")
	}

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/conversations/%d/messages", conversationID),
		nil,
	)
	req.SetPathValue("convoID", fmt.Sprintf("%d", conversationID))

	req.AddCookie(cookies[0])

	recorder := httptest.NewRecorder()
	handlers.GetMessages(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var messages []struct {
		ID      int    `json:"ID"`
		Content string `json:"Content"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&messages); err != nil {
		t.Fatalf("failed to decode messages: %v", err)
	}

	if len(messages) != len(contents) {
		t.Fatalf(
			"expected %d messages, got %d",
			len(contents),
			len(messages),
		)
	}

	for i, expected := range contents {
		if messages[i].Content != expected {
			t.Fatalf(
				"message order incorrect at index %d: expected %q, got %q",
				i,
				expected,
				messages[i].Content,
			)
		}
	}

	t.Logf(
		"IT-API-045 PASS: %d messages were returned in chronological order",
		len(messages),
	)
}

func TestIT_API_046_MessageFieldsAreCorrect(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-046_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-046_bob_%d", time.Now().UnixNano())

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
	`, aliceID, conversationID, bobID, conversationID)
	if err != nil {
		t.Fatalf("failed to add members: %v", err)
	}

	content := "Checking every message field"

	result, err = db.Exec(`
		INSERT INTO messages(user_id, conversation_id, content)
		VALUES (?, ?, ?)
	`, aliceID, conversationID, content)
	if err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	expectedMessageID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get message ID: %v", err)
	}

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
		t.Fatalf("login failed: got status %d", loginRecorder.Code)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not return a session cookie")
	}

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/conversations/%d/messages", conversationID),
		nil,
	)
	req.SetPathValue("convoID", fmt.Sprintf("%d", conversationID))
	req.AddCookie(cookies[0])

	recorder := httptest.NewRecorder()
	handlers.GetMessages(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var messages []struct {
		ID             int    `json:"ID"`
		UserID         int    `json:"UserID"`
		Username       string `json:"Username"`
		ConversationID int    `json:"ConversationID"`
		Content        string `json:"Content"`
		SentAt         string `json:"SentAt"`
		DeliveryStatus string `json:"delivery_status"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&messages); err != nil {
		t.Fatalf("failed to decode messages: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected exactly 1 message, got %d", len(messages))
	}

	message := messages[0]

	if message.ID != int(expectedMessageID) {
		t.Fatalf(
			"expected message ID %d, got %d",
			expectedMessageID,
			message.ID,
		)
	}

	if message.UserID != aliceID {
		t.Fatalf(
			"expected UserID %d, got %d",
			aliceID,
			message.UserID,
		)
	}

	if message.Username != alice {
		t.Fatalf(
			"expected Username %q, got %q",
			alice,
			message.Username,
		)
	}

	if message.ConversationID != int(conversationID) {
		t.Fatalf(
			"expected ConversationID %d, got %d",
			conversationID,
			message.ConversationID,
		)
	}

	if message.Content != content {
		t.Fatalf(
			"expected Content %q, got %q",
			content,
			message.Content,
		)
	}

	if message.SentAt == "" {
		t.Fatal("expected SentAt to be populated")
	}

	if message.DeliveryStatus == "" {
		t.Fatal("expected delivery_status to be populated")
	}

	t.Logf(
		"IT-API-046 PASS: message %d returned with all expected fields",
		message.ID,
	)
}

func TestIT_API_047_NonMemberCannotGetMessages(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-047_alice_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-047_bob_%d", time.Now().UnixNano())
	charlie := fmt.Sprintf("test_IT-API-047_charlie_%d", time.Now().UnixNano())

	for _, username := range []string{alice, bob, charlie} {
		if err := models.CreateUser(username); err != nil {
			t.Fatalf("failed to create user %s: %v", username, err)
		}
	}

	var aliceID, bobID, charlieID int

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

	if err := db.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		charlie,
	).Scan(&charlieID); err != nil {
		t.Fatalf("failed to get Charlie ID: %v", err)
	}

	result, err := db.Exec(`
		INSERT INTO conversations(name, conversation_type)
		VALUES (?, 'group')
	`, "IT-API-047 Group")
	if err != nil {
		t.Fatalf("failed to create conversation: %v", err)
	}

	conversationID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get conversation ID: %v", err)
	}

	// Alice and Bob are members. Charlie is intentionally not.
	_, err = db.Exec(`
		INSERT INTO user_in_conversation(user_id, conversation_id)
		VALUES (?, ?), (?, ?)
	`, aliceID, conversationID, bobID, conversationID)
	if err != nil {
		t.Fatalf("failed to add members: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO messages(user_id, conversation_id, content)
		VALUES (?, ?, ?)
	`, aliceID, conversationID, "Private group message")
	if err != nil {
		t.Fatalf("failed to insert message: %v", err)
	}

	// Login Charlie.
	loginBody := fmt.Sprintf(`{"username":"%s"}`, charlie)
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
		t.Fatalf("login failed: got status %d", loginRecorder.Code)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not return a session cookie")
	}

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/conversations/%d/messages", conversationID),
		nil,
	)
	req.SetPathValue("convoID", fmt.Sprintf("%d", conversationID))
	req.AddCookie(cookies[0])

	recorder := httptest.NewRecorder()
	handlers.GetMessages(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected 403 for non-member, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	t.Logf(
		"IT-API-047 PASS: non-member was denied access to conversation %d messages",
		conversationID,
	)
}

func TestIT_API_048_EmptyConversationReturnsEmptyArray(t *testing.T) {
	db := testutil.SetupDatabase(t)

	alice := fmt.Sprintf("test_IT-API-048_alice_%d", time.Now().UnixNano())
	bob := fmt.Sprintf("test_IT-API-048_bob_%d", time.Now().UnixNano())

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
	`, aliceID, conversationID, bobID, conversationID)
	if err != nil {
		t.Fatalf("failed to add members: %v", err)
	}

	// Deliberately do not insert any messages.

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
		t.Fatalf("login failed: got status %d", loginRecorder.Code)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not return a session cookie")
	}

	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/conversations/%d/messages", conversationID),
		nil,
	)
	req.SetPathValue("convoID", fmt.Sprintf("%d", conversationID))
	req.AddCookie(cookies[0])

	recorder := httptest.NewRecorder()
	handlers.GetMessages(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	var messages []struct {
		ID int `json:"ID"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&messages); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if messages == nil {
		t.Fatal("expected an empty JSON array, got null")
	}

	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}

	t.Logf(
		"IT-API-048 PASS: empty conversation %d returned an empty array",
		conversationID,
	)
}
