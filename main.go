package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"asherkhb.com/ease/appdiscovery"
	"asherkhb.com/ease/auth"
)

const defaultScanInterval = 8 * time.Second

//go:embed templates/index.html templates/login.html
var templateFS embed.FS

// PageData contains all data passed to the index template.
type PageData struct {
	Catalog
	User *auth.User
}

type Catalog struct {
	Root        *appdiscovery.Group
	LastScan    time.Time
	LastError   string
	AppsDir     string
	ScanEvery   time.Duration
	TotalApps   int
	SpecMatches int
}

type AppState struct {
	mu      sync.RWMutex
	catalog Catalog
}

func main() {
	// Stage 1: Parse only the auth-backend flag to determine which backend to use
	authBackendName := auth.GetEnvOrDefault("AUTH_BACKEND", "")

	// Scan args for -auth-backend flag (simple parse before full flag parsing)
	for i, arg := range os.Args[1:] {
		if arg == "-auth-backend" && i+1 < len(os.Args[1:]) {
			authBackendName = os.Args[i+2]
			break
		}
		if strings.HasPrefix(arg, "-auth-backend=") {
			authBackendName = strings.TrimPrefix(arg, "-auth-backend=")
			break
		}
	}

	// Initialize the selected auth backend (if any)
	var backend auth.AuthBackend
	if authBackendName != "" {
		factory := auth.GetBackend(authBackendName)
		if factory == nil {
			log.Fatal(auth.NewBackendError(authBackendName))
		}
		backend = factory()
	}

	// Stage 2: Create full FlagSet and register all flags
	fs := flag.NewFlagSet("ease", flag.ExitOnError)

	port := fs.Int("port", 8080, "Port to bind the web server")
	appsDirFlag := auth.StringVar(fs, "apps-dir", "APPS_DIR", "", "Directory containing app subdirectories with spec.yml files")
	scanIntervalFlag := fs.Duration("scan-interval", defaultScanInterval, "Interval between rescans of apps-dir")

	// Auth backend selection flag
	_ = auth.StringVar(fs, "auth-backend", "AUTH_BACKEND", "", "Authentication backend to use (e.g., 'file'). If empty, runs in public mode.")

	// Register backend-specific flags if a backend is selected
	if backend != nil {
		backend.RegisterFlags(fs)
	}

	// Parse all flags
	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// Initialize the auth backend if selected
	if backend != nil {
		if err := backend.Initialize(); err != nil {
			log.Fatalf("Failed to initialize auth backend %q: %v", authBackendName, err)
		}
		log.Printf("Authentication enabled using %q backend", authBackendName)
	} else {
		log.Println("Running in public mode (no authentication)")
	}

	// Parse templates
	indexTmpl, err := template.New("index.html").Funcs(templateFuncs()).ParseFS(templateFS, "templates/index.html")
	if err != nil {
		log.Fatalf("Failed to parse index template: %v", err)
	}

	loginTmpl, err := template.New("login.html").ParseFS(templateFS, "templates/login.html")
	if err != nil {
		log.Fatalf("Failed to parse login template: %v", err)
	}

	// Initialize session store (only needed if auth is enabled)
	var sessions *auth.SessionStore
	var authHandlers *auth.Handlers
	if backend != nil {
		sessions = auth.NewSessionStore()
		authHandlers = auth.NewHandlers(sessions, backend, loginTmpl)
	}

	state := &AppState{}
	applyScan(state, *appsDirFlag, *scanIntervalFlag)

	ticker := time.NewTicker(*scanIntervalFlag)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			applyScan(state, *appsDirFlag, *scanIntervalFlag)
		}
	}()

	mux := http.NewServeMux()

	// Set up routes based on auth mode
	if backend != nil {
		// Auth enabled: protect index and add login/logout routes
		indexHandler := auth.RequireAuthFunc(sessions, backend, func(w http.ResponseWriter, r *http.Request) {
			handleIndex(w, r, state, indexTmpl)
		})
		mux.Handle("/", indexHandler)
		mux.HandleFunc("/login", authHandlers.LoginHandler)
		mux.HandleFunc("/logout", authHandlers.LogoutHandler)
	} else {
		// Public mode: no auth, show anonymous user
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			handleIndexPublic(w, r, state, indexTmpl)
		})
		mux.HandleFunc("/login", auth.NotFoundHandler)
		mux.HandleFunc("/logout", auth.NotFoundHandler)
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("EASe listening on port %d", *port)
	log.Fatal(server.ListenAndServe())
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"basename": func(path string) string {
			if path == "" {
				return ""
			}
			// Use forward slash since paths are normalized to forward slashes
			if idx := strings.LastIndex(path, "/"); idx != -1 {
				return path[idx+1:]
			}
			return path
		},
		// dict creates a map from key-value pairs for passing multiple values to templates
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		// add performs integer addition
		"add": func(a, b int) int {
			return a + b
		},
		// mul performs integer multiplication
		"mul": func(a, b int) int {
			return a * b
		},
	}
}

func applyScan(state *AppState, appsDir string, scanEvery time.Duration) {
	catalog := Catalog{
		Root:      nil,
		LastScan:  time.Now(),
		AppsDir:   appsDir,
		ScanEvery: scanEvery,
	}

	if appsDir == "" {
		catalog.LastError = "apps_dir is not configured. Provide -apps-dir or set APPS_DIR."
		catalog.Root = &appdiscovery.Group{Path: ""}
		state.update(catalog)
		return
	}

	result, scanErr := appdiscovery.ScanApps(appsDir)
	catalog.Root = result.Root
	catalog.SpecMatches = result.SpecMatches
	catalog.TotalApps = result.TotalApps
	if scanErr != nil {
		catalog.LastError = scanErr.Error()
	}
	state.update(catalog)
}

func (state *AppState) update(catalog Catalog) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.catalog = catalog
}

func (state *AppState) snapshot() Catalog {
	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.catalog
}

// handleIndex handles the index page for authenticated users.
func handleIndex(w http.ResponseWriter, r *http.Request, state *AppState, tmpl *template.Template) {
	catalog := state.snapshot()
	user := auth.UserFromContext(r.Context())

	// Filter catalog based on user's groups
	if user != nil {
		catalog.Root = appdiscovery.FilterForUser(catalog.Root, user)
	}

	data := PageData{
		Catalog: catalog,
		User:    user,
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// handleIndexPublic handles the index page in public mode.
func handleIndexPublic(w http.ResponseWriter, r *http.Request, state *AppState, tmpl *template.Template) {
	catalog := state.snapshot()

	// In public mode, show only apps without group restrictions
	catalog.Root = appdiscovery.FilterForAnonymous(catalog.Root)

	data := PageData{
		Catalog: catalog,
		User:    auth.AnonymousUser(),
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
