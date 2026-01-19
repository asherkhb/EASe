package appdiscovery

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type AppSpec struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type App struct {
	ID          string
	Name        string
	Description string
	RelativeDir string
	SpecPath    string
}

type Group struct {
	Name   string
	Path   string
	Apps   []App
	Groups []*Group
}

type ScanResult struct {
	Root        *Group
	SpecMatches int
	TotalApps   int
}

func ScanApps(appsDir string) (ScanResult, error) {
	root := &Group{Name: "Apps", Path: ""}
	groups := map[string]*Group{
		"": root,
	}
	var matches int
	var errs []string

	walkErr := filepath.WalkDir(appsDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
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
			errs = append(errs, fmt.Sprintf("rel path error for %s: %v", path, relErr))
			return nil
		}

		spec := AppSpec{}
		if data, readErr := os.ReadFile(path); readErr == nil {
			if unmarshalErr := yaml.Unmarshal(data, &spec); unmarshalErr != nil {
				errs = append(errs, fmt.Sprintf("parse error for %s: %v", path, unmarshalErr))
			}
		} else {
			errs = append(errs, fmt.Sprintf("read error for %s: %v", path, readErr))
		}

		appName := strings.TrimSpace(spec.Name)
		if appName == "" {
			appName = filepath.Base(appDir)
		}

		groupRel := filepath.Dir(relDir)
		if groupRel == "." {
			groupRel = ""
		}
		group := getOrCreateGroup(groups, groupRel)
		group.Apps = append(group.Apps, App{
			ID:          strings.ReplaceAll(relDir, string(os.PathSeparator), "/"),
			Name:        appName,
			Description: strings.TrimSpace(spec.Description),
			RelativeDir: relDir,
			SpecPath:    path,
		})
		return nil
	})

	if walkErr != nil {
		errs = append(errs, walkErr.Error())
	}

	sortGroups(root)
	totalApps := countApps(root)

	if len(errs) > 0 {
		return ScanResult{Root: root, SpecMatches: matches, TotalApps: totalApps}, errors.New(strings.Join(errs, "; "))
	}
	return ScanResult{Root: root, SpecMatches: matches, TotalApps: totalApps}, nil
}

func getOrCreateGroup(groups map[string]*Group, groupRel string) *Group {
	if groupRel == "" {
		return groups[""]
	}
	if existing, ok := groups[groupRel]; ok {
		return existing
	}
	parentPath := filepath.Dir(groupRel)
	if parentPath == "." {
		parentPath = ""
	}
	parent := getOrCreateGroup(groups, parentPath)
	group := &Group{
		Name: filepath.Base(groupRel),
		Path: groupRel,
	}
	parent.Groups = append(parent.Groups, group)
	groups[groupRel] = group
	return group
}

func sortGroups(group *Group) {
	sort.Slice(group.Groups, func(i, j int) bool {
		return strings.ToLower(group.Groups[i].Name) < strings.ToLower(group.Groups[j].Name)
	})
	sort.Slice(group.Apps, func(i, j int) bool {
		return strings.ToLower(group.Apps[i].Name) < strings.ToLower(group.Apps[j].Name)
	})
	for _, child := range group.Groups {
		sortGroups(child)
	}
}

func countApps(group *Group) int {
	total := len(group.Apps)
	for _, child := range group.Groups {
		total += countApps(child)
	}
	return total
}
