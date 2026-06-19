package android

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	properties "github.com/arduino/go-properties-orderedmap"
)

const (
	aclDirName     = ".acl"
	aclRuntimeName = "runtime"
)

//go:embed runtime/*
var runtimeFS embed.FS

// RuntimeOSSuffix maps Android to Linux so Arduino package metadata can
// select Linux-hosted tools while ACL handles Android execution details.
func RuntimeOSSuffix(goos string) string {
	if goos == "android" {
		return "linux"
	}
	return properties.GetOSSuffix()
}

func installRuntime(root string) (string, error) {
	runtimeDir, err := filepath.Abs(filepath.Join(root, aclDirName, aclRuntimeName))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return "", err
	}
	entries, err := fs.ReadDir(runtimeFS, "runtime")
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := runtimeFS.ReadFile(filepath.ToSlash(filepath.Join("runtime", entry.Name())))
		if err != nil {
			return "", err
		}
		mode := fs.FileMode(0o644)
		if info, err := entry.Info(); err == nil {
			mode = info.Mode().Perm()
		}
		if err := os.WriteFile(filepath.Join(runtimeDir, entry.Name()), data, mode); err != nil {
			return "", err
		}
	}
	return runtimeDir, nil
}
