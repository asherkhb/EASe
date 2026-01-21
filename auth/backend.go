package auth

import (
	"errors"
	"flag"
	"fmt"
	"sort"
	"sync"
)

// User represents an authenticated user with their group memberships.
type User struct {
	Username string
	Groups   []string
}

// AnonymousUser returns a user representing an unauthenticated/public session.
func AnonymousUser() *User {
	return &User{
		Username: "Anonymous",
		Groups:   nil,
	}
}

// HasGroup checks if the user belongs to a specific group.
func (u *User) HasGroup(group string) bool {
	for _, g := range u.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// HasAnyGroup checks if the user belongs to any of the specified groups.
func (u *User) HasAnyGroup(groups []string) bool {
	for _, g := range groups {
		if u.HasGroup(g) {
			return true
		}
	}
	return false
}

// AuthBackend defines the interface that all authentication backends must implement.
type AuthBackend interface {
	// Name returns the unique identifier for this backend.
	Name() string

	// RegisterFlags registers backend-specific flags on the provided FlagSet.
	RegisterFlags(fs *flag.FlagSet)

	// Initialize is called after flags are parsed to set up the backend.
	// Returns an error if initialization fails (e.g., missing config file).
	Initialize() error

	// Authenticate validates credentials and returns the user if successful.
	// Returns an error if authentication fails.
	Authenticate(username, password string) (*User, error)

	// GetUserInfo retrieves user information by username.
	// This is used for session restoration where password isn't needed.
	GetUserInfo(username string) (*User, error)
}

// BackendFactory is a function that creates a new instance of an AuthBackend.
type BackendFactory func() AuthBackend

// ErrInvalidCredentials is returned when authentication fails due to bad username/password.
var ErrInvalidCredentials = errors.New("invalid username or password")

// ErrUserNotFound is returned when a user lookup fails.
var ErrUserNotFound = errors.New("user not found")

// Backend registry
var (
	registryMu sync.RWMutex
	registry   = make(map[string]BackendFactory)
)

// RegisterBackend registers a new authentication backend factory.
// This is typically called from an init() function in each backend implementation.
func RegisterBackend(name string, factory BackendFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// GetBackend retrieves a backend factory by name.
// Returns nil if the backend doesn't exist.
func GetBackend(name string) BackendFactory {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[name]
}

// ListBackends returns a sorted list of all registered backend names.
func ListBackends() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// NewBackendError creates an error message for when an invalid backend is specified.
func NewBackendError(requested string) error {
	available := ListBackends()
	if len(available) == 0 {
		return fmt.Errorf("unknown auth backend %q: no backends registered", requested)
	}
	return fmt.Errorf("unknown auth backend %q: available backends are: %v", requested, available)
}
