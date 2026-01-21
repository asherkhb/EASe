package auth

import (
	"html/template"
	"log"
	"net/http"
)

// LoginData contains data passed to the login template.
type LoginData struct {
	Error    string
	Username string
}

// Handlers contains HTTP handlers for authentication routes.
type Handlers struct {
	Sessions *SessionStore
	Backend  AuthBackend
	Template *template.Template
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(sessions *SessionStore, backend AuthBackend, tmpl *template.Template) *Handlers {
	return &Handlers{
		Sessions: sessions,
		Backend:  backend,
		Template: tmpl,
	}
}

// LoginHandler handles GET and POST requests to /login.
func (h *Handlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.showLoginForm(w, r, LoginData{})
	case http.MethodPost:
		h.handleLogin(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// showLoginForm renders the login form.
func (h *Handlers) showLoginForm(w http.ResponseWriter, r *http.Request, data LoginData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Template.Execute(w, data); err != nil {
		log.Printf("Error rendering login template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleLogin processes the login form submission.
func (h *Handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.showLoginForm(w, r, LoginData{Error: "Invalid form data"})
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		h.showLoginForm(w, r, LoginData{
			Error:    "Username and password are required",
			Username: username,
		})
		return
	}

	user, err := h.Backend.Authenticate(username, password)
	if err != nil {
		log.Printf("Authentication failed for user %q: %v", username, err)
		h.showLoginForm(w, r, LoginData{
			Error:    "Invalid username or password",
			Username: username,
		})
		return
	}

	// Create session
	if err := h.Sessions.SetUser(w, r, user); err != nil {
		log.Printf("Failed to create session for user %q: %v", username, err)
		h.showLoginForm(w, r, LoginData{
			Error:    "Failed to create session",
			Username: username,
		})
		return
	}

	log.Printf("User %q logged in successfully", username)

	// Redirect to index
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutHandler handles POST requests to /logout.
func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := h.Sessions.GetUsername(r)
	if username != "" {
		log.Printf("User %q logged out", username)
	}

	if err := h.Sessions.DestroySession(w, r); err != nil {
		log.Printf("Failed to destroy session: %v", err)
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// NotFoundHandler returns 404 for auth routes when auth is disabled (public mode).
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
