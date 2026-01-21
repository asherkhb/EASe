package auth

import (
	"flag"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
)

// createTestStore creates a session store for testing
func createTestStore() *SessionStore {
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

func TestLoginHandler_GET(t *testing.T) {
	store := createTestStore()

	tmpl := template.Must(template.New("login.html").Parse(`
		<!DOCTYPE html>
		<html>
		<body>
			{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
			<form method="POST">
				<input type="text" name="username">
				<input type="password" name="password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`))

	backend := &mockHandlerBackend{
		users: map[string]mockUser{
			"testuser": {password: "testpass", groups: []string{"dev"}},
		},
	}

	handlers := NewHandlers(store, backend, tmpl)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	handlers.LoginHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestLoginHandler_POST_Success(t *testing.T) {
	store := createTestStore()

	tmpl := template.Must(template.New("login.html").Parse(`<html></html>`))

	backend := &mockHandlerBackend{
		users: map[string]mockUser{
			"testuser": {password: "testpass", groups: []string{"dev"}},
		},
	}

	handlers := NewHandlers(store, backend, tmpl)

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "testpass")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.LoginHandler(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusSeeOther)
	}

	location := rec.Header().Get("Location")
	if location != "/" {
		t.Errorf("got redirect to %q, want %q", location, "/")
	}

	// Check that session cookie was set
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "ease_session" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session cookie to be set")
	}
}

func TestLoginHandler_POST_InvalidCredentials(t *testing.T) {
	store := createTestStore()

	tmpl := template.Must(template.New("login.html").Parse(`
		{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
	`))

	backend := &mockHandlerBackend{
		users: map[string]mockUser{
			"testuser": {password: "testpass", groups: []string{"dev"}},
		},
	}

	handlers := NewHandlers(store, backend, tmpl)

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "wrongpassword")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.LoginHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Invalid username or password") {
		t.Error("expected error message in response")
	}
}

func TestLoginHandler_POST_MissingCredentials(t *testing.T) {
	store := createTestStore()

	tmpl := template.Must(template.New("login.html").Parse(`
		{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
	`))

	backend := &mockHandlerBackend{
		users: map[string]mockUser{},
	}

	handlers := NewHandlers(store, backend, tmpl)

	form := url.Values{}
	form.Add("username", "")
	form.Add("password", "")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handlers.LoginHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "required") {
		t.Error("expected error message about required fields")
	}
}

func TestLogoutHandler_POST(t *testing.T) {
	store := createTestStore()

	tmpl := template.Must(template.New("login.html").Parse(`<html></html>`))
	backend := &mockHandlerBackend{users: map[string]mockUser{}}

	handlers := NewHandlers(store, backend, tmpl)

	// First create a session
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	store.SetUser(rec1, req1, &User{Username: "testuser", Groups: []string{}})
	cookies := rec1.Result().Cookies()

	// Now logout
	req2 := httptest.NewRequest(http.MethodPost, "/logout", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	rec2 := httptest.NewRecorder()

	handlers.LogoutHandler(rec2, req2)

	if rec2.Code != http.StatusSeeOther {
		t.Errorf("got status %d, want %d", rec2.Code, http.StatusSeeOther)
	}

	location := rec2.Header().Get("Location")
	if location != "/login" {
		t.Errorf("got redirect to %q, want %q", location, "/login")
	}
}

func TestLogoutHandler_GET_MethodNotAllowed(t *testing.T) {
	store := createTestStore()
	tmpl := template.Must(template.New("login.html").Parse(`<html></html>`))
	backend := &mockHandlerBackend{users: map[string]mockUser{}}

	handlers := NewHandlers(store, backend, tmpl)

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	rec := httptest.NewRecorder()

	handlers.LogoutHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestNotFoundHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()

	NotFoundHandler(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestLoginHandler_WithRealTemplate(t *testing.T) {
	// Create a temporary template file
	tmpDir, err := os.MkdirTemp("", "auth_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	templateContent := `<!DOCTYPE html>
<html>
<head><title>Login</title></head>
<body>
	{{if .Error}}<p class="error">{{.Error}}</p>{{end}}
	<form method="POST" action="/login">
		<input type="text" name="username" placeholder="Username">
		<input type="password" name="password" placeholder="Password">
		<button type="submit">Sign In</button>
	</form>
</body>
</html>`

	templatePath := filepath.Join(tmpDir, "login.html")
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	store := createTestStore()

	backend := &mockHandlerBackend{
		users: map[string]mockUser{
			"testuser": {password: "testpass", groups: []string{"dev"}},
		},
	}

	handlers := NewHandlers(store, backend, tmpl)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	handlers.LoginHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Sign In") {
		t.Error("expected 'Sign In' button in template output")
	}
}

// Mock backend for handler tests
type mockUser struct {
	password string
	groups   []string
}

type mockHandlerBackend struct {
	users map[string]mockUser
}

func (m *mockHandlerBackend) Name() string                   { return "mock" }
func (m *mockHandlerBackend) RegisterFlags(fs *flag.FlagSet) {}
func (m *mockHandlerBackend) Initialize() error              { return nil }

func (m *mockHandlerBackend) Authenticate(username, password string) (*User, error) {
	if u, ok := m.users[username]; ok {
		if u.password == password {
			return &User{
				Username: username,
				Groups:   u.groups,
			}, nil
		}
	}
	return nil, ErrInvalidCredentials
}

func (m *mockHandlerBackend) GetUserInfo(username string) (*User, error) {
	if u, ok := m.users[username]; ok {
		return &User{
			Username: username,
			Groups:   u.groups,
		}, nil
	}
	return nil, ErrUserNotFound
}
