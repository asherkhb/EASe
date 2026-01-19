package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
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
		Root:      &appdiscovery.Group{Name: "Apps", Path: ""},
		LastScan:  time.Now(),
		AppsDir:   appsDir,
		ScanEvery: scanEvery,
	}

	if appsDir == "" {
		catalog.LastError = "apps_dir is not configured. Provide -apps_dir or set APPS_DIR."
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
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
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
