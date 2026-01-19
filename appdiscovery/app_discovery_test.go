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
