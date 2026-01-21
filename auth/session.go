package auth

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
)

const (
	// SessionName is the name of the session cookie.
	SessionName = "ease_session"

	// sessionUserKey is the key used to store username in session.
	sessionUserKey = "username"

	// DefaultMaxAge is the default session max age in seconds (7 days).
	DefaultMaxAge = 86400 * 7
)

// SessionStore manages user sessions.
type SessionStore struct {
	store *sessions.CookieStore
}

// NewSessionStore creates a new session store.
// Keys are read from SESSION_AUTH_KEY and SESSION_ENCRYPT_KEY environment variables.
// If not provided, random keys are generated (sessions won't persist across restarts).
func NewSessionStore() *SessionStore {
	authKey := getOrGenerateKey("SESSION_AUTH_KEY", 32)
	encryptKey := getOrGenerateKey("SESSION_ENCRYPT_KEY", 32)

	store := sessions.NewCookieStore(authKey, encryptKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   DefaultMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure is set dynamically per-request based on localhost detection
	}

	return &SessionStore{store: store}
}

// getOrGenerateKey retrieves a key from environment or generates a random one.
func getOrGenerateKey(envName string, size int) []byte {
	if keyStr := os.Getenv(envName); keyStr != "" {
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			log.Printf("WARNING: %s is not valid base64, generating random key", envName)
		} else if len(key) < size {
			log.Printf("WARNING: %s is too short (got %d bytes, need %d), generating random key", envName, len(key), size)
		} else {
			return key[:size]
		}
	} else {
		log.Printf("WARNING: %s not set, generating random key (sessions will not persist across restarts)", envName)
	}

	// Generate random key
	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate random key: %v", err)
	}
	return key
}

// isLocalhost checks if the request is from localhost.
func isLocalhost(r *http.Request) bool {
	host := r.Host
	// Strip port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	return host == "localhost" ||
		host == "127.0.0.1" ||
		host == "::1" ||
		host == ""
}

// GetSession retrieves the session for the request.
func (s *SessionStore) GetSession(r *http.Request) (*sessions.Session, error) {
	return s.store.Get(r, SessionName)
}

// SetUser stores the user in the session.
func (s *SessionStore) SetUser(w http.ResponseWriter, r *http.Request, user *User) error {
	session, err := s.GetSession(r)
	if err != nil {
		return err
	}

	session.Values[sessionUserKey] = user.Username

	// Set Secure flag based on localhost detection
	session.Options.Secure = !isLocalhost(r)

	return session.Save(r, w)
}

// GetUsername retrieves the username from the session.
// Returns empty string if no user is logged in.
func (s *SessionStore) GetUsername(r *http.Request) string {
	session, err := s.GetSession(r)
	if err != nil {
		return ""
	}

	username, ok := session.Values[sessionUserKey].(string)
	if !ok {
		return ""
	}

	return username
}

// DestroySession removes the session.
func (s *SessionStore) DestroySession(w http.ResponseWriter, r *http.Request) error {
	session, err := s.GetSession(r)
	if err != nil {
		return err
	}

	// Set max age to -1 to delete the cookie
	session.Options.MaxAge = -1
	session.Values[sessionUserKey] = ""

	// Set Secure flag based on localhost detection
	session.Options.Secure = !isLocalhost(r)

	return session.Save(r, w)
}

// GenerateKeyCommand prints a command to generate session keys.
func GenerateKeyCommand() string {
	return `To generate persistent session keys, run:
  export SESSION_AUTH_KEY=$(openssl rand -base64 32)
  export SESSION_ENCRYPT_KEY=$(openssl rand -base64 32)`
}
