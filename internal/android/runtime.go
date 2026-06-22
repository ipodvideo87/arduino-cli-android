package android

import (
	"embed"
	"fmt"
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
		if entry.Name() == "ld-linux-aarch64.so.1" {
			mode |= 0o111
		}
		if err := os.WriteFile(filepath.Join(runtimeDir, entry.Name()), data, mode); err != nil {
			return "", err
		}
	}

	if err := addRuntimeLinkAliases(runtimeDir); err != nil {
		return "", err
	}
	if err := completeRuntimeClosure(root, runtimeDir, runtimeSourceRoots()); err != nil {
		return "", err
	}
	if err := addRuntimeLinkAliases(runtimeDir); err != nil {
		return "", err
	}
	return runtimeDir, nil
}

func addRuntimeLinkAliases(runtimeDir string) error {
	aliases := map[string]string{
		"libc.so":       "libc.so.6",
		"libdl.so":      "libdl.so.2",
		"libpthread.so": "libpthread.so.0",
		"libm.so":       "libm.so.6",
		"librt.so":      "librt.so.1",
		"libz.so":       "libz.so.1",
	}

	for alias, target := range aliases {
		aliasPath := filepath.Join(runtimeDir, alias)
		targetPath := filepath.Join(runtimeDir, target)
		if _, err := os.Stat(targetPath); err != nil {
			continue
		}
		if err := os.Remove(aliasPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove runtime alias %q: %w", aliasPath, err)
		}
		if err := os.Symlink(target, aliasPath); err != nil && !os.IsExist(err) {
			return fmt.Errorf("create runtime alias %q -> %q: %w", aliasPath, target, err)
		}
	}

	return nil
}
