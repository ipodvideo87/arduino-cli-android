package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	aclbuilder "github.com/arduino/arduino-cli/internal/acl/builder"
	"github.com/stretchr/testify/require"
)

func TestRunMissingLoaderArgument(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	exitCode := run([]string{
		"--name", "fixture",
		"--version", "1.0.0",
		"--arch", "aarch64",
		"--abi", "android-aarch64",
		"--compatibility", "experimental",
		"--output", t.TempDir(),
		"--lib", "lib.so",
	}, stdout, stderr)

	require.Equal(t, 2, exitCode)
	require.Contains(t, stderr.String(), "missing --loader")
}

func TestRunMissingOutputArgument(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	exitCode := run([]string{
		"--name", "fixture",
		"--version", "1.0.0",
		"--arch", "aarch64",
		"--abi", "android-aarch64",
		"--compatibility", "experimental",
		"--loader", "loader.so",
		"--lib", "lib.so",
	}, stdout, stderr)

	require.Equal(t, 2, exitCode)
	require.Contains(t, stderr.String(), "missing --output")
}

func TestRunRejectsDuplicateLibraryArguments(t *testing.T) {
	loader, lib := fixtureAssets(t)
	output := filepath.Join(t.TempDir(), "package")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}

	exitCode := run([]string{
		"--name", "fixture",
		"--version", "1.0.0",
		"--arch", "aarch64",
		"--abi", "android-aarch64",
		"--compatibility", "experimental",
		"--loader", loader,
		"--lib", lib,
		"--lib", lib,
		"--output", output,
	}, stdout, stderr)

	require.Equal(t, 2, exitCode)
	require.Contains(t, stderr.String(), "duplicate --lib path")
}

func TestRunRejectsEmptyLibraryValue(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	exitCode := run([]string{
		"--name", "fixture",
		"--version", "1.0.0",
		"--arch", "aarch64",
		"--abi", "android-aarch64",
		"--compatibility", "experimental",
		"--loader", "loader.so",
		"--lib", "",
		"--output", t.TempDir(),
	}, stdout, stderr)

	require.Equal(t, 2, exitCode)
	require.Contains(t, stderr.String(), "empty --lib value")
}

func TestRunRejectsExistingOutputPath(t *testing.T) {
	loader, lib := fixtureAssets(t)
	output := filepath.Join(t.TempDir(), "package")
	require.NoError(t, os.MkdirAll(output, 0o755))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	exitCode := run([]string{
		"--name", "fixture",
		"--version", "1.0.0",
		"--arch", "aarch64",
		"--abi", "android-aarch64",
		"--compatibility", "experimental",
		"--loader", loader,
		"--lib", lib,
		"--output", output,
	}, stdout, stderr)

	require.Equal(t, 1, exitCode)
	require.Contains(t, stderr.String(), "already exists")
}

func TestRunRejectsOutputInsideInputTree(t *testing.T) {
	root := t.TempDir()
	loader := filepath.Join(root, "loader.txt")
	lib := filepath.Join(root, "library.txt")
	require.NoError(t, os.WriteFile(loader, []byte("loader fixture"), 0o644))
	require.NoError(t, os.WriteFile(lib, []byte("library fixture"), 0o644))
	output := filepath.Join(root, "package")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	exitCode := run([]string{
		"--name", "fixture",
		"--version", "1.0.0",
		"--arch", "aarch64",
		"--abi", "android-aarch64",
		"--compatibility", "experimental",
		"--loader", loader,
		"--lib", lib,
		"--output", output,
	}, stdout, stderr)

	require.Equal(t, 1, exitCode)
	require.Contains(t, stderr.String(), "must not be inside")
}

func TestRunBuildsPackageFromFixtureAssets(t *testing.T) {
	loader, lib := fixtureAssets(t)
	output := filepath.Join(t.TempDir(), "package")
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	t.Setenv("SOURCE_DATE_EPOCH", "1718755200")

	exitCode := run([]string{
		"--id", "acl-test-runtime-cli",
		"--name", "fixture",
		"--version", "1.0.0",
		"--arch", "aarch64",
		"--abi", "android-aarch64",
		"--compatibility", "experimental",
		"--loader", loader,
		"--lib", lib,
		"--output", output,
	}, stdout, stderr)

	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), "verified: yes")
	require.Contains(t, stdout.String(), "acl-test-runtime-cli")

	builder := aclbuilder.NewBuilder()
	require.NoError(t, builder.Verify(output))
	for _, rel := range []string{"manifest.json", "metadata.json", "checksums.json", "version", "loader/" + filepath.Base(loader), "lib/" + filepath.Base(lib)} {
		_, err := os.Stat(filepath.Join(output, rel))
		require.NoError(t, err, rel)
	}
}

func fixtureAssets(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	loader := filepath.Join(dir, "loader.txt")
	lib := filepath.Join(dir, "library.txt")
	require.NoError(t, os.WriteFile(loader, []byte("loader fixture"), 0o644))
	require.NoError(t, os.WriteFile(lib, []byte("library fixture"), 0o644))
	return loader, lib
}
