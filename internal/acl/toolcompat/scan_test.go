package toolcompat

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	aclscan "github.com/arduino/arduino-cli/internal/acl/elfscan"
	"github.com/stretchr/testify/require"
)

func TestScanClassifiesScriptAndJavaAndELF(t *testing.T) {
	root := t.TempDir()
	shell := filepath.Join(root, "tool.sh")
	python := filepath.Join(root, "tool.py")
	jar := filepath.Join(root, "tool.jar")
	elf := filepath.Join(root, "tool-elf")

	require.NoError(t, os.WriteFile(shell, []byte("#!/bin/sh\necho hi\n"), 0o755))
	require.NoError(t, os.WriteFile(python, []byte("#!/usr/bin/env python3\nprint('hi')\n"), 0o755))
	writeZip(t, jar)
	require.NoError(t, os.WriteFile(elf, []byte("\x7fELFfake"), 0o755))

	scanner := NewScanner()
	scanner.inspectELF = func(path string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                 path,
			Exists:               true,
			IsELF:                true,
			Machine:              "AArch64",
			Interpreter:          "/system/bin/linker64",
			ImportedLibraries:    []string{"libc.so"},
			LooksLikeLinuxTarget: false,
		}, nil
	}

	report, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, report.Entries, 4)

	entries := map[string]Entry{}
	for _, entry := range report.Entries {
		entries[entry.RelativePath] = entry
	}
	require.Equal(t, "shell-script", entries["tool.sh"].ExecutableType)
	require.Equal(t, CategoryScript, entries["tool.sh"].CompatibilityCategory)

	require.Equal(t, "python-script", entries["tool.py"].ExecutableType)
	require.Equal(t, CategoryScript, entries["tool.py"].CompatibilityCategory)

	require.Equal(t, "java-archive", entries["tool.jar"].ExecutableType)
	require.Equal(t, CategoryUnknown, entries["tool.jar"].CompatibilityCategory)

	require.Equal(t, "elf", entries["tool-elf"].ExecutableType)
	require.Equal(t, CategoryAndroidCompatible, entries["tool-elf"].CompatibilityCategory)
}

func TestScanBuildsLinuxRuntimeCandidate(t *testing.T) {
	root := t.TempDir()
	elf := filepath.Join(root, "tool-elf")
	require.NoError(t, os.WriteFile(elf, []byte("\x7fELFfake"), 0o755))

	scanner := NewScanner()
	scanner.inspectELF = func(path string) (aclscan.Inspection, error) {
		return aclscan.Inspection{
			Path:                   path,
			Exists:                 true,
			IsELF:                  true,
			Machine:                "Advanced Micro Devices X86-64",
			Interpreter:            "/lib64/ld-linux-x86-64.so.2",
			ImportedLibraries:      []string{"libc.so.6", "libpthread.so.0"},
			RPath:                  "$ORIGIN/../lib",
			RunPath:                "$ORIGIN/runtime",
			HardcodedAbsolutePaths: []string{"/data/data/com.termux/files/usr/glibc/lib"},
			LooksLikeLinuxTarget:   true,
		}, nil
	}

	report, err := scanner.Scan(root)
	require.NoError(t, err)
	require.Len(t, report.Entries, 1)
	entry := report.Entries[0]
	require.Equal(t, CategoryLinuxGlibc, entry.CompatibilityCategory)
	require.True(t, entry.RequiresRuntime)
	require.Equal(t, "$ORIGIN/../lib", entry.RPath)
	require.Equal(t, "$ORIGIN/runtime", entry.RunPath)
	require.Contains(t, entry.SharedLibraries, "libc.so.6")
}

func TestScanSkipsNonExecutableFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "notes.txt"), []byte("plain text"), 0o644))

	report, err := NewScanner().Scan(root)
	require.NoError(t, err)
	require.Empty(t, report.Entries)
}

func TestDefaultPackagesRoot(t *testing.T) {
	root, err := DefaultPackagesRoot()
	require.NoError(t, err)
	require.Contains(t, filepath.ToSlash(root), "/.arduino15/packages")
}

func TestScanInstalledArduinoPackagesIfPresent(t *testing.T) {
	root, err := DefaultPackagesRoot()
	require.NoError(t, err)
	if _, err := os.Stat(root); err != nil {
		t.Skip("installed Arduino packages not present")
	}

	report, err := NewScanner().Scan(root)
	require.NoError(t, err)
	require.NotEmpty(t, report.Root)
}

func writeZip(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	entry, err := zipWriter.Create("META-INF/MANIFEST.MF")
	require.NoError(t, err)
	_, err = entry.Write([]byte("Manifest-Version: 1.0\n"))
	require.NoError(t, err)
	require.NoError(t, zipWriter.Close())
}
