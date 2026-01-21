package appdiscovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanAppsBuildsGroups(t *testing.T) {
	appsDir := t.TempDir()
	writeSpec(t, appsDir, "marketing/brand-pulse", "Brand Pulse", "Campaign performance.")
	writeSpec(t, appsDir, "monitoring/infra-core", "Infra Core", "Cluster health.")
	writeSpec(t, appsDir, "monitoring/security-hub", "Security Hub", "Threat signals.")

	result, err := ScanApps(appsDir)
	require.NoError(t, err)
	require.NotNil(t, result.Root)
	require.Equal(t, 3, result.SpecMatches)
	require.Equal(t, 3, result.TotalApps)

	monitoring := findGroup(result.Root, "monitoring")
	require.NotNil(t, monitoring)
	require.Len(t, monitoring.Apps, 2)

	marketing := findGroup(result.Root, "marketing")
	require.NotNil(t, marketing)
	require.Len(t, marketing.Apps, 1)
}

func TestScanAppsDefaultsName(t *testing.T) {
	appsDir := t.TempDir()
	specDir := filepath.Join(appsDir, "analytics/insights")
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yml"), []byte("description: Default name"), 0o600))

	result, err := ScanApps(appsDir)
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalApps)

	analytics := findGroup(result.Root, "analytics")
	require.NotNil(t, analytics)
	require.Len(t, analytics.Apps, 1)
	require.Equal(t, "insights", analytics.Apps[0].Name)
}

func writeSpec(t *testing.T, root, relDir, name, description string) {
	path := filepath.Join(root, relDir)
	require.NoError(t, os.MkdirAll(path, 0o755))
	spec := []byte("name: " + name + "\n" + "description: " + description + "\n")
	require.NoError(t, os.WriteFile(filepath.Join(path, "spec.yml"), spec, 0o600))
}

func findGroup(group *Group, path string) *Group {
	if group.Path == path {
		return group
	}
	for _, child := range group.Groups {
		if found := findGroup(child, path); found != nil {
			return found
		}
	}
	return nil
}

// Tests for FilterForUser function

// mockUserChecker implements UserGroupChecker for testing
type mockUserChecker struct {
	groups []string
}

func (m *mockUserChecker) HasAnyGroup(groups []string) bool {
	for _, g := range groups {
		for _, ug := range m.groups {
			if g == ug {
				return true
			}
		}
	}
	return false
}

func TestFilterForUser_NoGroups(t *testing.T) {
	// Apps with no groups should be visible to everyone
	root := &Group{
		Path: "",
		Apps: []App{
			{Name: "Public App", Groups: nil},
			{Name: "Also Public", Groups: []string{}},
		},
	}

	user := &mockUserChecker{groups: []string{"dev"}}
	filtered := FilterForUser(root, user)

	require.NotNil(t, filtered)
	require.Len(t, filtered.Apps, 2)
}

func TestFilterForUser_WithMatchingGroup(t *testing.T) {
	root := &Group{
		Path: "",
		Apps: []App{
			{Name: "Dev App", Groups: []string{"dev"}},
			{Name: "Admin App", Groups: []string{"admin"}},
			{Name: "Dev or Admin", Groups: []string{"dev", "admin"}},
		},
	}

	user := &mockUserChecker{groups: []string{"dev"}}
	filtered := FilterForUser(root, user)

	require.NotNil(t, filtered)
	require.Len(t, filtered.Apps, 2)

	appNames := make([]string, len(filtered.Apps))
	for i, app := range filtered.Apps {
		appNames[i] = app.Name
	}
	require.Contains(t, appNames, "Dev App")
	require.Contains(t, appNames, "Dev or Admin")
}

func TestFilterForUser_MultipleUserGroups(t *testing.T) {
	root := &Group{
		Path: "",
		Apps: []App{
			{Name: "Dev App", Groups: []string{"dev"}},
			{Name: "Admin App", Groups: []string{"admin"}},
			{Name: "Finance App", Groups: []string{"finance"}},
		},
	}

	user := &mockUserChecker{groups: []string{"dev", "admin"}}
	filtered := FilterForUser(root, user)

	require.NotNil(t, filtered)
	require.Len(t, filtered.Apps, 2)
}

func TestFilterForUser_EmptyUserGroups(t *testing.T) {
	root := &Group{
		Path: "",
		Apps: []App{
			{Name: "Public App", Groups: nil},
			{Name: "Dev App", Groups: []string{"dev"}},
		},
	}

	user := &mockUserChecker{groups: []string{}}
	filtered := FilterForUser(root, user)

	require.NotNil(t, filtered)
	require.Len(t, filtered.Apps, 1)
	require.Equal(t, "Public App", filtered.Apps[0].Name)
}

func TestFilterForUser_NestedGroups(t *testing.T) {
	root := &Group{
		Path: "",
		Groups: []*Group{
			{
				Path: "engineering",
				Apps: []App{
					{Name: "Dev Tool", Groups: []string{"dev"}},
				},
				Groups: []*Group{
					{
						Path: "engineering/frontend",
						Apps: []App{
							{Name: "React App", Groups: []string{"frontend", "dev"}},
						},
					},
				},
			},
			{
				Path: "finance",
				Apps: []App{
					{Name: "Budget App", Groups: []string{"finance"}},
				},
			},
		},
	}

	user := &mockUserChecker{groups: []string{"dev"}}
	filtered := FilterForUser(root, user)

	require.NotNil(t, filtered)
	require.Len(t, filtered.Groups, 1) // Only engineering should remain

	eng := filtered.Groups[0]
	require.Equal(t, "engineering", eng.Name())
	require.Len(t, eng.Apps, 1)
	require.Equal(t, "Dev Tool", eng.Apps[0].Name)

	// Frontend subgroup should also be included
	require.Len(t, eng.Groups, 1)
	require.Equal(t, "frontend", eng.Groups[0].Name())
	require.Len(t, eng.Groups[0].Apps, 1)
}

func TestFilterForUser_RemovesEmptyGroups(t *testing.T) {
	root := &Group{
		Path: "",
		Groups: []*Group{
			{
				Path: "engineering",
				Apps: []App{
					{Name: "Dev Tool", Groups: []string{"dev"}},
				},
			},
			{
				Path: "finance",
				Apps: []App{
					{Name: "Budget App", Groups: []string{"finance"}},
				},
			},
		},
	}

	user := &mockUserChecker{groups: []string{"dev"}}
	filtered := FilterForUser(root, user)

	require.NotNil(t, filtered)
	require.Len(t, filtered.Groups, 1) // Finance group should be removed
	require.Equal(t, "engineering", filtered.Groups[0].Name())
}

func TestFilterForUser_NilRoot(t *testing.T) {
	user := &mockUserChecker{groups: []string{"dev"}}
	filtered := FilterForUser(nil, user)

	require.Nil(t, filtered)
}

func TestFilterForUser_MixedPublicAndRestricted(t *testing.T) {
	root := &Group{
		Path: "",
		Groups: []*Group{
			{
				Path: "shared",
				Apps: []App{
					{Name: "Public Dashboard", Groups: nil},
					{Name: "Dev Dashboard", Groups: []string{"dev"}},
					{Name: "Admin Dashboard", Groups: []string{"admin"}},
				},
			},
		},
	}

	user := &mockUserChecker{groups: []string{"dev"}}
	filtered := FilterForUser(root, user)

	require.NotNil(t, filtered)
	require.Len(t, filtered.Groups, 1)
	require.Len(t, filtered.Groups[0].Apps, 2) // Public + Dev

	appNames := make([]string, len(filtered.Groups[0].Apps))
	for i, app := range filtered.Groups[0].Apps {
		appNames[i] = app.Name
	}
	require.Contains(t, appNames, "Public Dashboard")
	require.Contains(t, appNames, "Dev Dashboard")
}

func TestFilterForUser_PreservesAppMetadata(t *testing.T) {
	root := &Group{
		Path: "",
		Apps: []App{
			{
				Name:        "My App",
				ID:          "apps/myapp",
				Description: "A test application",
				Groups:      []string{"dev"},
			},
		},
	}

	user := &mockUserChecker{groups: []string{"dev"}}
	filtered := FilterForUser(root, user)

	require.NotNil(t, filtered)
	require.Len(t, filtered.Apps, 1)

	app := filtered.Apps[0]
	require.Equal(t, "My App", app.Name)
	require.Equal(t, "apps/myapp", app.ID)
	require.Equal(t, "A test application", app.Description)
	require.Equal(t, []string{"dev"}, app.Groups)
}

func TestScanAppsWithGroups(t *testing.T) {
	appsDir := t.TempDir()

	// Create app with groups in spec
	specDir := filepath.Join(appsDir, "engineering/dev-tool")
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	spec := []byte(`name: Dev Tool
description: A developer tool
groups:
  - dev
  - engineering
`)
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.yml"), spec, 0o600))

	result, err := ScanApps(appsDir)
	require.NoError(t, err)
	require.Equal(t, 1, result.TotalApps)

	eng := findGroup(result.Root, "engineering")
	require.NotNil(t, eng)
	require.Len(t, eng.Apps, 1)
	require.Equal(t, []string{"dev", "engineering"}, eng.Apps[0].Groups)
}
