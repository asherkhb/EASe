package auth

import (
	"context"
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
)

func TestContextWithUser(t *testing.T) {
	user := &User{
		Username: "testuser",
		Groups:   []string{"dev", "admin"},
	}

	ctx := context.Background()
	ctx = ContextWithUser(ctx, user)

	got := UserFromContext(ctx)
	if got == nil {
		t.Fatal("expected user in context")
	}

	if got.Username != user.Username {
		t.Errorf("got username %q, want %q", got.Username, user.Username)
	}

	if len(got.Groups) != len(user.Groups) {
		t.Errorf("got %d groups, want %d", len(got.Groups), len(user.Groups))
	}
}

func TestUserFromContext_NoUser(t *testing.T) {
	ctx := context.Background()

	got := UserFromContext(ctx)
	if got != nil {
		t.Error("expected no user in empty context")
	}
}

// createTestStoreMiddleware creates a session store for middleware tests
func createTestStoreMiddleware() *SessionStore {
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

func TestRequireAuth_NoSession(t *testing.T) {
	// Create a simple handler that should not be reached
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	store := createTestStoreMiddleware()

	// Create a mock backend
	backend := &mockMiddlewareBackend{
		users: map[string]User{},
	}

	wrapped := RequireAuthFunc(store, backend, handler)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should redirect to login
	if rec.Code != http.StatusSeeOther {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/login" {
		t.Errorf("got redirect to %q, want /login", location)
	}

	if handlerCalled {
		t.Error("handler should not have been called")
	}
}

func TestRequireAuth_WithValidSession(t *testing.T) {
	// Create a handler that checks for user in context
	var receivedUser *User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	store := createTestStoreMiddleware()

	testUser := User{
		Username: "testuser",
		Groups:   []string{"dev"},
	}

	backend := &mockMiddlewareBackend{
		users: map[string]User{
			"testuser": testUser,
		},
	}

	wrapped := RequireAuthFunc(store, backend, handler)

	// First, create a request to set up a session
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()

	// Get session and set user
	session, _ := store.store.Get(req1, "ease_session")
	session.Values["username"] = "testuser"
	session.Save(req1, rec1)

	// Get the cookie from the response
	cookies := rec1.Result().Cookies()

	// Now make the actual request with the session cookie
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	rec2 := httptest.NewRecorder()

	wrapped.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec2.Code, http.StatusOK)
	}

	if receivedUser == nil {
		t.Fatal("expected user in context")
	}

	if receivedUser.Username != "testuser" {
		t.Errorf("got username %q, want %q", receivedUser.Username, "testuser")
	}
}

func TestRequireAuth_WithInvalidUser(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	store := createTestStoreMiddleware()

	// Backend with no users (user has been deleted)
	backend := &mockMiddlewareBackend{
		users: map[string]User{},
	}

	wrapped := RequireAuthFunc(store, backend, handler)

	// Create request with session for non-existent user
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()

	session, _ := store.store.Get(req1, "ease_session")
	session.Values["username"] = "deleteduser"
	session.Save(req1, rec1)

	cookies := rec1.Result().Cookies()

	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	rec2 := httptest.NewRecorder()

	wrapped.ServeHTTP(rec2, req2)

	// Should redirect because user no longer exists
	if rec2.Code != http.StatusSeeOther {
		t.Errorf("got status %d, want %d", rec2.Code, http.StatusSeeOther)
	}

	if handlerCalled {
		t.Error("handler should not have been called for deleted user")
	}
}

func TestOptionalAuth_WithSession(t *testing.T) {
	var receivedUser *User
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	store := createTestStoreMiddleware()

	testUser := User{
		Username: "testuser",
		Groups:   []string{"dev"},
	}

	backend := &mockMiddlewareBackend{
		users: map[string]User{
			"testuser": testUser,
		},
	}

	wrapped := &OptionalAuth{
		Sessions: store,
		Backend:  backend,
		Next:     handler,
	}

	// Create session
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	session, _ := store.store.Get(req1, "ease_session")
	session.Values["username"] = "testuser"
	session.Save(req1, rec1)
	cookies := rec1.Result().Cookies()

	// Make request with session
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	rec2 := httptest.NewRecorder()

	wrapped.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec2.Code, http.StatusOK)
	}

	if receivedUser == nil {
		t.Fatal("expected user in context")
	}

	if receivedUser.Username != "testuser" {
		t.Errorf("got username %q, want %q", receivedUser.Username, "testuser")
	}
}

func TestOptionalAuth_NoSession(t *testing.T) {
	var receivedUser *User
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		receivedUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	store := createTestStoreMiddleware()

	backend := &mockMiddlewareBackend{
		users: map[string]User{},
	}

	wrapped := &OptionalAuth{
		Sessions: store,
		Backend:  backend,
		Next:     handler,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	if !handlerCalled {
		t.Error("handler should have been called")
	}

	if receivedUser != nil {
		t.Error("expected no user in context for unauthenticated request")
	}
}

// mockMiddlewareBackend for middleware tests
type mockMiddlewareBackend struct {
	users map[string]User
}

func (m *mockMiddlewareBackend) Name() string                   { return "mock" }
func (m *mockMiddlewareBackend) RegisterFlags(fs *flag.FlagSet) {}
func (m *mockMiddlewareBackend) Initialize() error              { return nil }
func (m *mockMiddlewareBackend) Authenticate(username, password string) (*User, error) {
	if user, ok := m.users[username]; ok {
		return &user, nil
	}
	return nil, ErrInvalidCredentials
}
func (m *mockMiddlewareBackend) GetUserInfo(username string) (*User, error) {
	if user, ok := m.users[username]; ok {
		return &user, nil
	}
	return nil, ErrUserNotFound
}
