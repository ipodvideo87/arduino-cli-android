package android

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompleteRuntimeClosureCopiesRecursiveDependencies(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, ".acl", "runtime")
	require.NoError(t, os.MkdirAll(runtimeDir, 0o755))

	target := filepath.Join(root, "bin", "tool")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("target"), 0o755))

	sourceDir := filepath.Join(root, "sources")
	require.NoError(t, os.MkdirAll(sourceDir, 0o755))
	for _, name := range []string{"libA.so", "libB.so", "libC.so"} {
		require.NoError(t, os.WriteFile(filepath.Join(sourceDir, name), []byte(name), 0o644))
	}

	graph := map[string][]string{
		target:                               {"libA.so"},
		filepath.Join(runtimeDir, "libA.so"): {"libB.so"},
		filepath.Join(runtimeDir, "libB.so"): {"libC.so"},
		filepath.Join(runtimeDir, "libC.so"): {},
	}

	err := completeRuntimeClosureWithHooks(root, runtimeDir, []string{sourceDir}, runtimeClosureHooks{
		listPatchTargets: func(root, runtimeDir string) ([]string, error) {
			return []string{target}, nil
		},
		neededLibraries: func(path string) ([]string, error) {
			return graph[path], nil
		},
		findSource: func(name string, sourceRoots []string) (string, error) {
			for _, sourceRoot := range sourceRoots {
				candidate := filepath.Join(sourceRoot, name)
				if _, err := os.Stat(candidate); err == nil {
					return candidate, nil
				}
			}
			return "", os.ErrNotExist
		},
		copyFile: func(src, dst string) error {
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			return os.WriteFile(dst, data, 0o644)
		},
	})
	require.NoError(t, err)
	for _, name := range []string{"libA.so", "libB.so", "libC.so"} {
		require.FileExists(t, filepath.Join(runtimeDir, name))
	}
}

func TestCompleteRuntimeClosureReportsMissingDependency(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, ".acl", "runtime")
	require.NoError(t, os.MkdirAll(runtimeDir, 0o755))

	target := filepath.Join(root, "bin", "tool")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("target"), 0o755))

	err := completeRuntimeClosureWithHooks(root, runtimeDir, nil, runtimeClosureHooks{
		listPatchTargets: func(root, runtimeDir string) ([]string, error) {
			return []string{target}, nil
		},
		neededLibraries: func(path string) ([]string, error) {
			return []string{"libMissing.so"}, nil
		},
		findSource: func(name string, sourceRoots []string) (string, error) {
			return "", os.ErrNotExist
		},
		copyFile: func(src, dst string) error {
			return nil
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "libMissing.so")
}
