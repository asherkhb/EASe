package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

func TestSessionStore_SetAndGetUser(t *testing.T) {
	store := createTestStore()

	// Create initial request to set user
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()

	user := &User{Username: "testuser", Groups: []string{"dev"}}
	err := store.SetUser(rec1, req1, user)
	if err != nil {
		t.Fatalf("SetUser error: %v", err)
	}

	// Get cookies from response
	cookies := rec1.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}

	// Create new request with session cookie
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	username := store.GetUsername(req2)
	if username != "testuser" {
		t.Errorf("got username %q, want %q", username, "testuser")
	}
}

func TestSessionStore_GetUsername_NoSession(t *testing.T) {
	store := createTestStore()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	username := store.GetUsername(req)
	if username != "" {
		t.Errorf("expected empty username, got %q", username)
	}
}

func TestSessionStore_GetUsername_EmptyUsername(t *testing.T) {
	store := createTestStore()

	// Set up session with empty username
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()

	session, _ := store.store.Get(req1, "ease_session")
	session.Values["username"] = ""
	session.Save(req1, rec1)

	cookies := rec1.Result().Cookies()

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	username := store.GetUsername(req2)
	if username != "" {
		t.Errorf("expected empty username for empty session value, got %q", username)
	}
}

func TestSessionStore_DestroySession(t *testing.T) {
	store := createTestStore()

	// First, create a session
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()

	err := store.SetUser(rec1, req1, &User{Username: "testuser", Groups: []string{}})
	if err != nil {
		t.Fatalf("SetUser error: %v", err)
	}

	cookies := rec1.Result().Cookies()

	// Verify session exists
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	username := store.GetUsername(req2)
	if username != "testuser" {
		t.Fatal("session should exist before destroy")
	}

	// Destroy the session
	rec2 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req3.AddCookie(c)
	}

	err = store.DestroySession(rec2, req3)
	if err != nil {
		t.Fatalf("DestroySession error: %v", err)
	}

	// The response should have a cookie that clears the session
	destroyCookies := rec2.Result().Cookies()
	if len(destroyCookies) == 0 {
		t.Fatal("expected cookie in destroy response")
	}

	// Create new request with the destroy cookie
	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range destroyCookies {
		req4.AddCookie(c)
	}

	username = store.GetUsername(req4)
	if username != "" {
		t.Errorf("expected empty username after session destroy, got %q", username)
	}
}

func TestSessionStore_HttpOnlyCookie(t *testing.T) {
	store := createTestStore()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	store.SetUser(rec, req, &User{Username: "testuser", Groups: []string{}})

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie")
	}

	if !cookies[0].HttpOnly {
		t.Error("expected HttpOnly=true for session cookie")
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"localhost", true},
		{"localhost:8080", true},
		{"127.0.0.1", true},
		{"127.0.0.1:3000", true},
		{"example.com", false},
		{"example.com:443", false},
		{"192.168.1.1", false},
		{"10.0.0.1:8080", false},
		{"", true}, // empty host treated as localhost
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tt.host

			result := isLocalhost(req)
			if result != tt.expected {
				t.Errorf("isLocalhost for host %q = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

func TestGetSession(t *testing.T) {
	store := createTestStore()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	session, err := store.GetSession(req)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}

	if session == nil {
		t.Error("expected non-nil session")
	}
}

func TestNewSessionStore(t *testing.T) {
	// Test that NewSessionStore works without crashing
	// Note: This will use random keys since env vars aren't set
	store := NewSessionStore()
	if store == nil {
		t.Error("expected non-nil session store")
	}
}

// createTestStoreFromPackage creates a test store - duplicated to avoid import cycle
// This is already defined in handlers_test.go but we need it here too
func init() {
	// createTestStore is defined in handlers_test.go, so tests using it
	// will work when running all tests together
}

// Standalone createTestStoreSession for session tests
func createTestStoreSession() *SessionStore {
	store := sessions.NewCookieStore(
		[]byte("12345678901234567890123456789012"),
		[]byte("12345678901234567890123456789012"),
	)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	return &SessionStore{store: store}
}
