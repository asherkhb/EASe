package appdiscovery

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// RootPath represents the root group's path (empty string by convention)
const RootPath = ""

type AppSpec struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type App struct {
	ID          string // Forward-slash normalized path for use as unique identifier
	Name        string
	Description string
	SpecPath    string
}

type Group struct {
	Path   string // Forward-slash normalized relative path
	Apps   []App
	Groups []*Group
}

// Name returns the basename of the group's path (the folder name)
func (g *Group) Name() string {
	if g.Path == RootPath {
		return ""
	}
	idx := strings.LastIndex(g.Path, "/")
	if idx == -1 {
		return g.Path
	}
	return g.Path[idx+1:]
}

type ScanResult struct {
	Root        *Group
	SpecMatches int
	TotalApps   int
}

// normalizePath converts OS-specific paths to forward-slash format
// and normalizes "." to empty string for root path handling
func normalizePath(path string) string {
	if path == "." || path == "" {
		return RootPath
	}
	return strings.ReplaceAll(path, string(os.PathSeparator), "/")
}

func ScanApps(appsDir string) (ScanResult, error) {
	root := &Group{Path: RootPath}
	groups := map[string]*Group{
		RootPath: root,
	}
	var matches int
	var totalApps int
	var errs []string

	walkErr := filepath.WalkDir(appsDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("walk error at %s: %v", path, err)
			errs = append(errs, fmt.Sprintf("walk error at %s: %v", path, err))
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != "spec.yml" {
			return nil
		}
		matches++
		appDir := filepath.Dir(path)
		relDir, relErr := filepath.Rel(appsDir, appDir)
		if relErr != nil {
			log.Printf("rel path error for %s: %v", path, relErr)
			errs = append(errs, fmt.Sprintf("rel path error for %s: %v", path, relErr))
			return nil
		}

		// Normalize to forward slashes for consistent cross-platform handling
		appID := normalizePath(relDir)

		spec := AppSpec{}
		if data, readErr := os.ReadFile(path); readErr == nil {
			if unmarshalErr := yaml.Unmarshal(data, &spec); unmarshalErr != nil {
				log.Printf("parse error for %s: %v", path, unmarshalErr)
				errs = append(errs, fmt.Sprintf("parse error for %s: %v", path, unmarshalErr))
			}
		} else {
			log.Printf("read error for %s: %v", path, readErr)
			errs = append(errs, fmt.Sprintf("read error for %s: %v", path, readErr))
		}

		appName := strings.TrimSpace(spec.Name)
		if appName == "" {
			appName = filepath.Base(appDir)
		}

		groupPath := normalizePath(filepath.Dir(relDir))
		group := getOrCreateGroup(groups, groupPath)
		group.Apps = append(group.Apps, App{
			ID:          appID,
			Name:        appName,
			Description: strings.TrimSpace(spec.Description),
			SpecPath:    path,
		})
		totalApps++
		return nil
	})

	if walkErr != nil {
		errs = append(errs, walkErr.Error())
	}

	sortGroups(root)

	if len(errs) > 0 {
		return ScanResult{Root: root, SpecMatches: matches, TotalApps: totalApps}, errors.New(strings.Join(errs, "; "))
	}
	return ScanResult{Root: root, SpecMatches: matches, TotalApps: totalApps}, nil
}

func getOrCreateGroup(groups map[string]*Group, groupPath string) *Group {
	if groupPath == RootPath {
		return groups[RootPath]
	}
	if existing, ok := groups[groupPath]; ok {
		return existing
	}
	// Find parent path by stripping last path component
	parentPath := groupPath
	if idx := strings.LastIndex(groupPath, "/"); idx != -1 {
		parentPath = groupPath[:idx]
	} else {
		parentPath = RootPath
	}
	parent := getOrCreateGroup(groups, parentPath)
	group := &Group{
		Path: groupPath,
	}
	parent.Groups = append(parent.Groups, group)
	groups[groupPath] = group
	return group
}

func sortGroups(group *Group) {
	sort.Slice(group.Groups, func(i, j int) bool {
		return strings.ToLower(group.Groups[i].Name()) < strings.ToLower(group.Groups[j].Name())
	})
	sort.Slice(group.Apps, func(i, j int) bool {
		return strings.ToLower(group.Apps[i].Name) < strings.ToLower(group.Apps[j].Name)
	})
	for _, child := range group.Groups {
		sortGroups(child)
	}
}
