# Elastic Application ServicE (EASe)

EASe is a lightweight Go web app that discovers sub-apps from a directory tree and renders them in a grouped catalog. It is designed to act as the front door for launching and proxying multiple containerized apps (Dash, Streamlit, Flask, etc.). This initial version focuses on discovery, grouping, a simple UI, and modular authentication.

## Features

- **Recursive discovery** of `spec.yml` files under a configurable `apps_dir`.
- **Automatic grouping** based on directory structure (nested folders become nested groups).
- **Periodic rescan** every few seconds to keep the catalog fresh.
- **Modular authentication** with pluggable backends (file-based included, AWS Cognito and Entra ID planned).
- **Group-based access control** - apps can specify required groups for visibility.
- **Public mode** - runs without authentication if no backend is configured.
- **Single static binary** build (templates are embedded with `go:embed`; the UI uses Tailwind via CDN for styling).

## Directory Structure

EASe expects an `apps_dir` where each app lives in its own folder with a `spec.yml`. The structure can be nested to form groups.

Example:

```
apps_dir/
├─ app1/spec.yml
├─ groupA/app2/spec.yml
├─ groupA/app3/spec.yml
├─ groupA/subgroupA2/app4/spec.yml
└─ groupB/app5/spec.yml
```

Apps are grouped by their parent folders, so `groupA/app2` and `groupA/app3` appear under **groupA**, and `groupA/subgroupA2/app4` appears under **groupA/subgroupA2**.

## spec.yml format

`spec.yml` supports a minimal schema:

```yaml
name: Sales Dashboard
description: Metrics for Q4 campaign performance.
groups:
  - finance
  - admin
```

- `name` defaults to the app directory name if omitted.
- `description` is optional.
- `groups` is optional. If specified, only authenticated users belonging to at least one of these groups can see the app. If omitted or empty, the app is visible to all users (including anonymous users in public mode).

## Build

```
go build -o ease .
```

The output binary (`ease`) is the only artifact you need to run.

## Run

EASe can run in two modes: **public mode** (no authentication) or **authenticated mode** (with a configured auth backend).

### Public Mode (No Authentication)

If no auth backend is specified, EASe runs in public mode. All users see "Anonymous" and can only access apps with no group restrictions.

```bash
./ease -apps-dir /path/to/apps_dir
```

Or with environment variable:

```bash
export APPS_DIR=/path/to/apps_dir
./ease
```

### Authenticated Mode

To enable authentication, specify an auth backend using the `-auth-backend` flag or `AUTH_BACKEND` environment variable.

#### File-Based Authentication

The file backend reads users from a text file with whitespace-separated columns:

**Format:** `username password group1,group2,group3`

- Username and password are required
- Groups are optional (comma-separated list)
- Lines starting with `#` are comments
- Empty lines are ignored

**Example users file:**

```
# username password groups
admin secretpass admin,developers
developer devpass developers
readonly readonlypass
finance financepass finance,admin
```

**Run with file auth:**

```bash
./ease -apps-dir /path/to/apps_dir -auth-backend file -file-auth-users /path/to/users.txt
```

Or with environment variables:

```bash
export APPS_DIR=/path/to/apps_dir
export AUTH_BACKEND=file
export FILE_AUTH_USERS=/path/to/users.txt
./ease
```

### Session Persistence

By default, session keys are generated randomly at startup. For persistent sessions across restarts, set these environment variables:

```bash
export SESSION_AUTH_KEY=$(openssl rand -base64 32)
export SESSION_ENCRYPT_KEY=$(openssl rand -base64 32)
./ease -apps-dir /path/to/apps_dir -auth-backend file -file-auth-users users.txt
```

### Optional flags

- `-port` (default: `8080`) — HTTP port for the UI.
- `-apps-dir` (env: `APPS_DIR`) — Directory containing app subdirectories with spec.yml files.
- `-scan-interval` (default: `8s`) — How often EASe rescans `apps-dir`.
- `-auth-backend` (env: `AUTH_BACKEND`) — Authentication backend to use (`file` or empty for public mode).
- `-file-auth-users` (env: `FILE_AUTH_USERS`) — Path to users file (required when using `file` backend).

Example:

```bash
./ease -apps-dir ./apps_dir -port 9090 -scan-interval 10s -auth-backend file -file-auth-users ./users.txt
```

## Run in development

```
go run ./... -apps_dir ./apps_dir
```

Then open <http://localhost:8080> to view the catalog.

## Sample apps for screenshots

Use the helper script to create a realistic app tree for demoing the UI or capturing screenshots:

```bash
./scripts/create_sample_apps.sh /tmp/ease_apps
./ease -apps-dir /tmp/ease_apps -auth-backend file -file-auth-users /tmp/ease_apps/users.txt
```

The script creates sample apps with group restrictions and an example users file to test authentication and access control.

## Authentication

### Access Control

EASe uses group-based access control:

1. **Apps without groups** are visible to everyone (including anonymous users in public mode)
2. **Apps with groups** are only visible to authenticated users who belong to at least one of those groups
3. **Anonymous users** (public mode) can only see apps without group restrictions

### Security Features

- **Session cookies** with HttpOnly flag to prevent XSS attacks
- **Secure cookie flag** automatically enabled for non-localhost requests
- **SameSite=Lax** cookie policy to prevent CSRF attacks
- **Automatic session validation** - invalid sessions redirect to login
- **Localhost detection** - Secure flag disabled for local development

### Adding New Authentication Backends

EASe uses a modular authentication system that makes it easy to add new backends (e.g., AWS Cognito, Entra ID).

#### Backend Interface

All backends must implement the `auth.AuthBackend` interface:

```go
type AuthBackend interface {
    // Name returns the unique identifier for this backend
    Name() string

    // RegisterFlags registers backend-specific flags on the provided FlagSet
    RegisterFlags(fs *flag.FlagSet)

    // Initialize is called after flags are parsed to set up the backend
    Initialize() error

    // Authenticate validates credentials and returns the user if successful
    Authenticate(username, password string) (*User, error)

    // GetUserInfo retrieves user information by username
    GetUserInfo(username string) (*User, error)
}
```

#### Creating a New Backend

1. **Create a new file** in the `auth/` package (e.g., `auth/cognito_backend.go`)

2. **Implement the interface:**

```go
package auth

import "flag"

type CognitoBackend struct {
    userPoolID *string
    clientID   *string
    region     *string
}

func NewCognitoBackend() AuthBackend {
    return &CognitoBackend{}
}

func (c *CognitoBackend) Name() string {
    return "cognito"
}

func (c *CognitoBackend) RegisterFlags(fs *flag.FlagSet) {
    c.userPoolID = StringVar(fs, "cognito-user-pool-id", "COGNITO_USER_POOL_ID", "", 
        "AWS Cognito User Pool ID")
    c.clientID = StringVar(fs, "cognito-client-id", "COGNITO_CLIENT_ID", "", 
        "AWS Cognito App Client ID")
    c.region = StringVar(fs, "cognito-region", "COGNITO_REGION", "us-east-1", 
        "AWS Region for Cognito")
}

func (c *CognitoBackend) Initialize() error {
    // Validate configuration and set up AWS SDK client
    // Return error if required config is missing
    return nil
}

func (c *CognitoBackend) Authenticate(username, password string) (*User, error) {
    // Call AWS Cognito to authenticate user
    // Retrieve user groups from Cognito
    // Return User{Username: username, Groups: groups}
    return nil, nil
}

func (c *CognitoBackend) GetUserInfo(username string) (*User, error) {
    // Retrieve user info from Cognito by username
    return nil, nil
}
```

3. **Register the backend** in an `init()` function:

```go
func init() {
    RegisterBackend("cognito", NewCognitoBackend)
}
```

4. **Use the new backend:**

```bash
./ease -auth-backend cognito -cognito-user-pool-id us-east-1_ABC123 -cognito-client-id xyz789
```

Or with environment variables:

```bash
export AUTH_BACKEND=cognito
export COGNITO_USER_POOL_ID=us-east-1_ABC123
export COGNITO_CLIENT_ID=xyz789
./ease -apps-dir /path/to/apps
```

#### Backend Configuration Guidelines

- Use `auth.StringVar()`, `auth.IntVar()`, `auth.BoolVar()` for flags to ensure env var fallback
- Prefix flag names with backend name (e.g., `-cognito-*`, `-entra-*`)
- Validate required configuration in `Initialize()` and return helpful error messages
- Return `auth.ErrInvalidCredentials` for failed authentication
- Return `auth.ErrUserNotFound` when user doesn't exist

## Testing

Run the test suite:

```
go test ./...
```
