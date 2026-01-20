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
)

const defaultScanInterval = 8 * time.Second

//go:embed templates/index.html
var templateFS embed.FS

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
	port := flag.Int("port", 8080, "Port to bind the web server")
	appsDirFlag := flag.String("apps_dir", "", "Directory containing app subdirectories with spec.yml files")
	scanIntervalFlag := flag.Duration("scan_interval", defaultScanInterval, "Interval between rescans of apps_dir")
	flag.Parse()

	appsDir := *appsDirFlag
	if appsDir == "" {
		if envDir := os.Getenv("APPS_DIR"); envDir != "" {
			appsDir = envDir
		}
	}

	state := &AppState{}
	applyScan(state, appsDir, *scanIntervalFlag)

	ticker := time.NewTicker(*scanIntervalFlag)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			applyScan(state, appsDir, *scanIntervalFlag)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", state.handleIndex)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("EASe listening on port %d", *port)
	log.Fatal(server.ListenAndServe())
}

func applyScan(state *AppState, appsDir string, scanEvery time.Duration) {
	catalog := Catalog{
		Root:      nil,
		LastScan:  time.Now(),
		AppsDir:   appsDir,
		ScanEvery: scanEvery,
	}

	if appsDir == "" {
		catalog.LastError = "apps_dir is not configured. Provide -apps_dir or set APPS_DIR."
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

func (state *AppState) handleIndex(w http.ResponseWriter, r *http.Request) {
	catalog := state.snapshot()

	funcMap := template.FuncMap{
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

	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFS(templateFS, "templates/index.html")
	if err != nil {
		log.Printf("template parse error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, catalog); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
