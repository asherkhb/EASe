package auth

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// userContextKey is the context key for storing the authenticated user.
	userContextKey contextKey = "user"
)

// UserFromContext retrieves the authenticated user from the request context.
// Returns nil if no user is in the context.
func UserFromContext(ctx context.Context) *User {
	user, ok := ctx.Value(userContextKey).(*User)
	if !ok {
		return nil
	}
	return user
}

// ContextWithUser returns a new context with the user attached.
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// RequireAuth is middleware that enforces authentication.
// Unauthenticated users are redirected to the login page.
type RequireAuth struct {
	Sessions *SessionStore
	Backend  AuthBackend
	Next     http.Handler
}

// ServeHTTP implements http.Handler.
func (m *RequireAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username := m.Sessions.GetUsername(r)
	if username == "" {
		// Not authenticated, redirect to login
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get full user info from backend
	user, err := m.Backend.GetUserInfo(username)
	if err != nil {
		// User no longer exists in backend, clear session and redirect
		_ = m.Sessions.DestroySession(w, r)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Attach user to context and continue
	ctx := ContextWithUser(r.Context(), user)
	m.Next.ServeHTTP(w, r.WithContext(ctx))
}

// RequireAuthFunc wraps a handler function with authentication middleware.
func RequireAuthFunc(sessions *SessionStore, backend AuthBackend, handler http.HandlerFunc) http.Handler {
	return &RequireAuth{
		Sessions: sessions,
		Backend:  backend,
		Next:     handler,
	}
}

// OptionalAuth is middleware that attaches user to context if authenticated,
// but allows the request to proceed either way.
// Use this for pages that work differently for authenticated vs anonymous users.
type OptionalAuth struct {
	Sessions *SessionStore
	Backend  AuthBackend
	Next     http.Handler
}

// ServeHTTP implements http.Handler.
func (m *OptionalAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username := m.Sessions.GetUsername(r)
	if username != "" {
		// Try to get user info
		if user, err := m.Backend.GetUserInfo(username); err == nil {
			ctx := ContextWithUser(r.Context(), user)
			r = r.WithContext(ctx)
		}
	}

	m.Next.ServeHTTP(w, r)
}
