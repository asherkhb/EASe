package auth

import (
	"flag"
	"testing"
)

// MockBackend is a test implementation of AuthBackend
type MockBackend struct {
	name  string
	users map[string]*User
}

func NewMockBackend(name string) *MockBackend {
	return &MockBackend{
		name: name,
		users: map[string]*User{
			"testuser": {Username: "testuser", Groups: []string{"group1", "group2"}},
		},
	}
}

func (m *MockBackend) Name() string {
	return m.name
}

func (m *MockBackend) RegisterFlags(fs *flag.FlagSet) {}

func (m *MockBackend) Initialize() error {
	return nil
}

func (m *MockBackend) Authenticate(username, password string) (*User, error) {
	if user, ok := m.users[username]; ok && password == "testpass" {
		return user, nil
	}
	return nil, ErrInvalidCredentials
}

func (m *MockBackend) GetUserInfo(username string) (*User, error) {
	if user, ok := m.users[username]; ok {
		return user, nil
	}
	return nil, ErrUserNotFound
}

func TestUserHasGroup(t *testing.T) {
	user := &User{
		Username: "testuser",
		Groups:   []string{"admin", "developers"},
	}

	tests := []struct {
		name     string
		group    string
		expected bool
	}{
		{"existing group admin", "admin", true},
		{"existing group developers", "developers", true},
		{"non-existing group", "finance", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := user.HasGroup(tt.group)
			if result != tt.expected {
				t.Errorf("HasGroup(%q) = %v, want %v", tt.group, result, tt.expected)
			}
		})
	}
}

func TestUserHasGroup_EmptyGroups(t *testing.T) {
	user := &User{
		Username: "testuser",
		Groups:   []string{},
	}

	if user.HasGroup("any") {
		t.Error("User with empty groups should not have any group")
	}
}

func TestUserHasGroup_NilGroups(t *testing.T) {
	user := &User{
		Username: "testuser",
		Groups:   nil,
	}

	if user.HasGroup("any") {
		t.Error("User with nil groups should not have any group")
	}
}

func TestUserHasAnyGroup(t *testing.T) {
	user := &User{
		Username: "testuser",
		Groups:   []string{"admin", "developers"},
	}

	tests := []struct {
		name     string
		groups   []string
		expected bool
	}{
		{"single match", []string{"admin"}, true},
		{"multiple with first match", []string{"admin", "hr"}, true},
		{"multiple with second match", []string{"finance", "developers"}, true},
		{"all match", []string{"admin", "developers"}, true},
		{"no match single", []string{"finance"}, false},
		{"no match multiple", []string{"finance", "hr"}, false},
		{"empty list", []string{}, false},
		{"nil list", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := user.HasAnyGroup(tt.groups)
			if result != tt.expected {
				t.Errorf("HasAnyGroup(%v) = %v, want %v", tt.groups, result, tt.expected)
			}
		})
	}
}

func TestAnonymousUser(t *testing.T) {
	user := AnonymousUser()

	if user.Username != "Anonymous" {
		t.Errorf("AnonymousUser().Username = %q, want %q", user.Username, "Anonymous")
	}

	if user.Groups != nil {
		t.Errorf("AnonymousUser().Groups = %v, want nil", user.Groups)
	}

	if user.HasGroup("any") {
		t.Error("AnonymousUser() should not have any groups")
	}

	if user.HasAnyGroup([]string{"any", "group"}) {
		t.Error("AnonymousUser() should not match any groups")
	}
}

func TestRegisterAndGetBackend(t *testing.T) {
	// Save original registry and restore after test
	registryMu.Lock()
	originalRegistry := registry
	registry = make(map[string]BackendFactory)
	registryMu.Unlock()

	defer func() {
		registryMu.Lock()
		registry = originalRegistry
		registryMu.Unlock()
	}()

	// Register a test backend
	testFactory := func() AuthBackend {
		return NewMockBackend("test")
	}
	RegisterBackend("test", testFactory)

	// Get the backend
	factory := GetBackend("test")
	if factory == nil {
		t.Fatal("GetBackend() returned nil for registered backend")
	}

	backend := factory()
	if backend.Name() != "test" {
		t.Errorf("backend.Name() = %q, want %q", backend.Name(), "test")
	}

	// Test non-existent backend
	factory = GetBackend("nonexistent")
	if factory != nil {
		t.Error("GetBackend() should return nil for non-existent backend")
	}
}

func TestListBackends(t *testing.T) {
	// Save original registry and restore after test
	registryMu.Lock()
	originalRegistry := registry
	registry = make(map[string]BackendFactory)
	registryMu.Unlock()

	defer func() {
		registryMu.Lock()
		registry = originalRegistry
		registryMu.Unlock()
	}()

	RegisterBackend("charlie", func() AuthBackend { return NewMockBackend("charlie") })
	RegisterBackend("alpha", func() AuthBackend { return NewMockBackend("alpha") })
	RegisterBackend("bravo", func() AuthBackend { return NewMockBackend("bravo") })

	backends := ListBackends()

	if len(backends) != 3 {
		t.Errorf("ListBackends() returned %d backends, want 3", len(backends))
	}

	// Should be sorted alphabetically
	expected := []string{"alpha", "bravo", "charlie"}
	for i, name := range backends {
		if name != expected[i] {
			t.Errorf("ListBackends()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestNewBackendError(t *testing.T) {
	// Save original registry and restore after test
	registryMu.Lock()
	originalRegistry := registry
	registry = make(map[string]BackendFactory)
	registryMu.Unlock()

	defer func() {
		registryMu.Lock()
		registry = originalRegistry
		registryMu.Unlock()
	}()

	RegisterBackend("available1", func() AuthBackend { return NewMockBackend("available1") })
	RegisterBackend("available2", func() AuthBackend { return NewMockBackend("available2") })

	err := NewBackendError("invalid")
	if err == nil {
		t.Fatal("NewBackendError() returned nil")
	}

	errMsg := err.Error()
	if !containsString(errMsg, "invalid") {
		t.Errorf("error message should contain requested backend name: %s", errMsg)
	}
	if !containsString(errMsg, "available1") || !containsString(errMsg, "available2") {
		t.Errorf("error message should list available backends: %s", errMsg)
	}
}

func TestNewBackendError_NoBackends(t *testing.T) {
	// Save original registry and restore after test
	registryMu.Lock()
	originalRegistry := registry
	registry = make(map[string]BackendFactory)
	registryMu.Unlock()

	defer func() {
		registryMu.Lock()
		registry = originalRegistry
		registryMu.Unlock()
	}()

	err := NewBackendError("invalid")
	if err == nil {
		t.Fatal("NewBackendError() returned nil")
	}

	errMsg := err.Error()
	if !containsString(errMsg, "no backends registered") {
		t.Errorf("error message should indicate no backends registered: %s", errMsg)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
