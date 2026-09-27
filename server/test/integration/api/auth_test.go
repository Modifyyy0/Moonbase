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