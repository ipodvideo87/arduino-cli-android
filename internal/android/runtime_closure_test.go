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

func TestTermuxRuntimeSourceRootsPrefersBaseLibBeforeGlibcLib(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "lib"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(base, "glibc", "lib"), 0o755))

	roots := termuxRuntimeSourceRoots([]string{base})
	require.GreaterOrEqual(t, len(roots), 2)
	require.Equal(t, filepath.Join(base, "lib"), roots[0])
	require.Equal(t, filepath.Join(base, "glibc", "lib"), roots[1])
}

func TestFindRuntimeSourcePrefersApprovedBaseLibRootForLibStdCxx(t *testing.T) {
	base := t.TempDir()
	baseLib := filepath.Join(base, "lib")
	baseGlibcLib := filepath.Join(base, "glibc", "lib")
	require.NoError(t, os.MkdirAll(baseLib, 0o755))
	require.NoError(t, os.MkdirAll(baseGlibcLib, 0o755))

	hostExe, err := os.Executable()
	require.NoError(t, err)
	basePath := filepath.Join(baseLib, "libstdc++.so.6")
	glibcPath := filepath.Join(baseGlibcLib, "libstdc++.so.6")
	copyFile(t, hostExe, basePath, 0o755)
	copyFile(t, hostExe, glibcPath, 0o755)

	src, err := findRuntimeSource("libstdc++.so.6", termuxRuntimeSourceRoots([]string{base}))
	require.NoError(t, err)
	require.Equal(t, basePath, src)
}

func TestCompleteRuntimeClosureUsesApprovedSourceRootsRecursively(t *testing.T) {
	root := t.TempDir()
	runtimeDir := filepath.Join(root, ".acl", "runtime")
	require.NoError(t, os.MkdirAll(runtimeDir, 0o755))

	target := filepath.Join(root, "bin", "tool")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("target"), 0o755))

	base := t.TempDir()
	baseLib := filepath.Join(base, "lib")
	baseGlibcLib := filepath.Join(base, "glibc", "lib")
	require.NoError(t, os.MkdirAll(baseLib, 0o755))
	require.NoError(t, os.MkdirAll(baseGlibcLib, 0o755))

	hostExe, err := os.Executable()
	require.NoError(t, err)
	copyFile(t, hostExe, filepath.Join(baseLib, "libstdc++.so.6"), 0o755)
	copyFile(t, hostExe, filepath.Join(baseGlibcLib, "libgcc_s.so.1"), 0o755)
	copyFile(t, hostExe, filepath.Join(baseGlibcLib, "libm.so.6"), 0o755)

	graph := map[string][]string{
		target: {"libstdc++.so.6"},
		filepath.Join(runtimeDir, "libstdc++.so.6"): {"libgcc_s.so.1"},
		filepath.Join(runtimeDir, "libgcc_s.so.1"):  {"libm.so.6"},
		filepath.Join(runtimeDir, "libm.so.6"):      {},
	}

	closureErr := completeRuntimeClosureWithHooks(root, runtimeDir, termuxRuntimeSourceRoots([]string{base}), runtimeClosureHooks{
		listPatchTargets: func(root, runtimeDir string) ([]string, error) {
			return []string{target}, nil
		},
		neededLibraries: func(path string) ([]string, error) {
			return graph[path], nil
		},
		findSource: findRuntimeSource,
		copyFile: func(src, dst string) error {
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			return os.WriteFile(dst, data, 0o644)
		},
	})
	require.NoError(t, closureErr)
	require.FileExists(t, filepath.Join(runtimeDir, "libstdc++.so.6"))
	require.FileExists(t, filepath.Join(runtimeDir, "libgcc_s.so.1"))
	require.FileExists(t, filepath.Join(runtimeDir, "libm.so.6"))
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
