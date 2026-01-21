package auth

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
)

func init() {
	RegisterBackend("file", NewFileBackend)
}

// FileBackend implements AuthBackend using a simple text file for user storage.
// File format: whitespace-separated columns where:
//   - Column 1: username
//   - Column 2: password (plaintext, for development use only)
//   - Column 3 (optional): comma-separated list of groups
//
// Example:
//
//	admin secretpass admin,users
//	developer devpass developers
//	readonly readonlypass
type FileBackend struct {
	usersFile *string

	mu    sync.RWMutex
	users map[string]*fileUser
}

type fileUser struct {
	username string
	password string
	groups   []string
}

// NewFileBackend creates a new file-based authentication backend.
func NewFileBackend() AuthBackend {
	return &FileBackend{
		users: make(map[string]*fileUser),
	}
}

// Name returns the backend identifier.
func (f *FileBackend) Name() string {
	return "file"
}

// RegisterFlags registers file backend specific flags.
func (f *FileBackend) RegisterFlags(fs *flag.FlagSet) {
	f.usersFile = StringVar(fs, "file-auth-users", "FILE_AUTH_USERS", "",
		"Path to users file (format: username password group1,group2)")
}

// Initialize loads the users file and validates configuration.
func (f *FileBackend) Initialize() error {
	if f.usersFile == nil || *f.usersFile == "" {
		return fmt.Errorf("file auth backend requires -file-auth-users flag or FILE_AUTH_USERS env var")
	}

	return f.loadUsersFile(*f.usersFile)
}

// loadUsersFile reads and parses the users configuration file.
func (f *FileBackend) loadUsersFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open users file: %w", err)
	}
	defer file.Close()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Clear existing users
	f.users = make(map[string]*fileUser)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		user, err := parseLine(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}

		if _, exists := f.users[user.username]; exists {
			return fmt.Errorf("line %d: duplicate username %q", lineNum, user.username)
		}

		f.users[user.username] = user
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading users file: %w", err)
	}

	if len(f.users) == 0 {
		return fmt.Errorf("users file is empty or contains no valid entries")
	}

	return nil
}

// parseLine parses a single line from the users file.
// Format: username password [group1,group2,...]
func parseLine(line string) (*fileUser, error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil, fmt.Errorf("expected at least 2 fields (username password), got %d", len(fields))
	}

	user := &fileUser{
		username: fields[0],
		password: fields[1],
		groups:   nil,
	}

	// Parse optional groups (third field, comma-separated)
	if len(fields) >= 3 {
		groupStr := fields[2]
		if groupStr != "" {
			user.groups = strings.Split(groupStr, ",")
			// Trim whitespace from each group
			for i, g := range user.groups {
				user.groups[i] = strings.TrimSpace(g)
			}
		}
	}

	return user, nil
}

// Authenticate validates the username and password.
func (f *FileBackend) Authenticate(username, password string) (*User, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	user, exists := f.users[username]
	if !exists {
		return nil, ErrInvalidCredentials
	}

	if user.password != password {
		return nil, ErrInvalidCredentials
	}

	return &User{
		Username: user.username,
		Groups:   user.groups,
	}, nil
}

// GetUserInfo retrieves user information by username.
func (f *FileBackend) GetUserInfo(username string) (*User, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	user, exists := f.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}

	return &User{
		Username: user.username,
		Groups:   user.groups,
	}, nil
}
