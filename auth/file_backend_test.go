package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileBackendName(t *testing.T) {
	backend := NewFileBackend()
	if backend.Name() != "file" {
		t.Errorf("Name() = %q, want %q", backend.Name(), "file")
	}
}

func TestFileBackendInitialize_MissingFile(t *testing.T) {
	backend := &FileBackend{}
	emptyPath := ""
	backend.usersFile = &emptyPath

	err := backend.Initialize()
	if err == nil {
		t.Error("Initialize() should fail when users file is not configured")
	}
}

func TestFileBackendInitialize_NonExistentFile(t *testing.T) {
	backend := &FileBackend{}
	path := "/nonexistent/path/users.txt"
	backend.usersFile = &path

	err := backend.Initialize()
	if err == nil {
		t.Error("Initialize() should fail when users file does not exist")
	}
}

func TestFileBackendLoadUsers(t *testing.T) {
	// Create a temporary users file
	content := `# This is a comment
admin adminpass admin,developers
developer devpass developers
readonly readonlypass
`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	err := backend.loadUsersFile(tmpFile)
	if err != nil {
		t.Fatalf("loadUsersFile() error: %v", err)
	}

	// Check admin user
	admin, ok := backend.users["admin"]
	if !ok {
		t.Fatal("admin user not found")
	}
	if admin.password != "adminpass" {
		t.Errorf("admin.password = %q, want %q", admin.password, "adminpass")
	}
	if len(admin.groups) != 2 || admin.groups[0] != "admin" || admin.groups[1] != "developers" {
		t.Errorf("admin.groups = %v, want [admin developers]", admin.groups)
	}

	// Check developer user
	developer, ok := backend.users["developer"]
	if !ok {
		t.Fatal("developer user not found")
	}
	if len(developer.groups) != 1 || developer.groups[0] != "developers" {
		t.Errorf("developer.groups = %v, want [developers]", developer.groups)
	}

	// Check readonly user (no groups)
	readonly, ok := backend.users["readonly"]
	if !ok {
		t.Fatal("readonly user not found")
	}
	if readonly.groups != nil {
		t.Errorf("readonly.groups = %v, want nil", readonly.groups)
	}
}

func TestFileBackendLoadUsers_EmptyFile(t *testing.T) {
	tmpFile := createTempFile(t, "# only comments\n\n")
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	err := backend.loadUsersFile(tmpFile)
	if err == nil {
		t.Error("loadUsersFile() should fail for empty file")
	}
}

func TestFileBackendLoadUsers_InvalidFormat(t *testing.T) {
	// Only one field (missing password)
	tmpFile := createTempFile(t, "usernameonly\n")
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	err := backend.loadUsersFile(tmpFile)
	if err == nil {
		t.Error("loadUsersFile() should fail for invalid format")
	}
}

func TestFileBackendLoadUsers_DuplicateUser(t *testing.T) {
	content := `user1 pass1 group1
user1 pass2 group2
`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	err := backend.loadUsersFile(tmpFile)
	if err == nil {
		t.Error("loadUsersFile() should fail for duplicate username")
	}
}

func TestFileBackendAuthenticate_ValidCredentials(t *testing.T) {
	content := `testuser testpass group1,group2`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	backend.usersFile = &tmpFile
	if err := backend.Initialize(); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	user, err := backend.Authenticate("testuser", "testpass")
	if err != nil {
		t.Fatalf("Authenticate() error: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Username = %q, want %q", user.Username, "testuser")
	}
	if len(user.Groups) != 2 {
		t.Errorf("Groups = %v, want 2 groups", user.Groups)
	}
}

func TestFileBackendAuthenticate_InvalidPassword(t *testing.T) {
	content := `testuser testpass group1`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	backend.usersFile = &tmpFile
	if err := backend.Initialize(); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	_, err := backend.Authenticate("testuser", "wrongpass")
	if err != ErrInvalidCredentials {
		t.Errorf("Authenticate() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestFileBackendAuthenticate_NonExistentUser(t *testing.T) {
	content := `testuser testpass group1`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	backend.usersFile = &tmpFile
	if err := backend.Initialize(); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	_, err := backend.Authenticate("nonexistent", "anypass")
	if err != ErrInvalidCredentials {
		t.Errorf("Authenticate() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestFileBackendGetUserInfo_Existing(t *testing.T) {
	content := `testuser testpass admin,developers`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	backend.usersFile = &tmpFile
	if err := backend.Initialize(); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	user, err := backend.GetUserInfo("testuser")
	if err != nil {
		t.Fatalf("GetUserInfo() error: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Username = %q, want %q", user.Username, "testuser")
	}
	if len(user.Groups) != 2 {
		t.Errorf("Groups = %v, want 2 groups", user.Groups)
	}
}

func TestFileBackendGetUserInfo_NonExistent(t *testing.T) {
	content := `testuser testpass admin`
	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	backend := &FileBackend{users: make(map[string]*fileUser)}
	backend.usersFile = &tmpFile
	if err := backend.Initialize(); err != nil {
		t.Fatalf("Initialize() error: %v", err)
	}

	_, err := backend.GetUserInfo("nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("GetUserInfo() error = %v, want ErrUserNotFound", err)
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantUser   string
		wantPass   string
		wantGroups []string
		wantErr    bool
	}{
		{
			name:       "full line with groups",
			line:       "admin adminpass admin,developers",
			wantUser:   "admin",
			wantPass:   "adminpass",
			wantGroups: []string{"admin", "developers"},
			wantErr:    false,
		},
		{
			name:       "no groups",
			line:       "readonly readonlypass",
			wantUser:   "readonly",
			wantPass:   "readonlypass",
			wantGroups: nil,
			wantErr:    false,
		},
		{
			name:       "single group",
			line:       "user pass group1",
			wantUser:   "user",
			wantPass:   "pass",
			wantGroups: []string{"group1"},
			wantErr:    false,
		},
		{
			name:       "extra whitespace",
			line:       "  user   pass   group1,group2  ",
			wantUser:   "user",
			wantPass:   "pass",
			wantGroups: []string{"group1", "group2"},
			wantErr:    false,
		},
		{
			name:    "only username",
			line:    "justuser",
			wantErr: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := parseLine(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Error("parseLine() should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("parseLine() error: %v", err)
			}

			if user.username != tt.wantUser {
				t.Errorf("username = %q, want %q", user.username, tt.wantUser)
			}
			if user.password != tt.wantPass {
				t.Errorf("password = %q, want %q", user.password, tt.wantPass)
			}
			if len(user.groups) != len(tt.wantGroups) {
				t.Errorf("groups = %v, want %v", user.groups, tt.wantGroups)
			}
			for i := range user.groups {
				if user.groups[i] != tt.wantGroups[i] {
					t.Errorf("groups[%d] = %q, want %q", i, user.groups[i], tt.wantGroups[i])
				}
			}
		})
	}
}

func TestFileBackendRegistered(t *testing.T) {
	// The file backend should be auto-registered via init()
	factory := GetBackend("file")
	if factory == nil {
		t.Fatal("file backend should be registered")
	}

	backend := factory()
	if backend.Name() != "file" {
		t.Errorf("backend.Name() = %q, want %q", backend.Name(), "file")
	}
}

// createTempFile creates a temporary file with the given content and returns its path
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "users.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return tmpFile
}
