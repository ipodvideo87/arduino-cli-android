package android

import (
	"debug/elf"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type runtimeClosureHooks struct {
	listPatchTargets func(root, runtimeDir string) ([]string, error)
	neededLibraries  func(path string) ([]string, error)
	findSource       func(name string, sourceRoots []string) (string, error)
	copyFile         func(src, dst string) error
}

func completeRuntimeClosure(root, runtimeDir string, sourceRoots []string) error {
	return completeRuntimeClosureWithHooks(root, runtimeDir, sourceRoots, runtimeClosureHooks{
		listPatchTargets: listRuntimePatchTargets,
		neededLibraries:  runtimeNeededLibraries,
		findSource:       findRuntimeSource,
		copyFile:         copyRuntimeFile,
	})
}

func completeRuntimeClosureWithHooks(root, runtimeDir string, sourceRoots []string, hooks runtimeClosureHooks) error {
	if hooks.listPatchTargets == nil {
		hooks.listPatchTargets = listRuntimePatchTargets
	}
	if hooks.neededLibraries == nil {
		hooks.neededLibraries = runtimeNeededLibraries
	}
	if hooks.findSource == nil {
		hooks.findSource = findRuntimeSource
	}
	if hooks.copyFile == nil {
		hooks.copyFile = copyRuntimeFile
	}

	targets, err := hooks.listPatchTargets(root, runtimeDir)
	if err != nil {
		return err
	}

	queue := append([]string(nil), targets...)
	seenFiles := map[string]struct{}{}
	seenLibs := map[string]struct{}{}
	available := map[string]struct{}{}
	missing := map[string]struct{}{}

	for {
		if len(queue) == 0 {
			break
		}
		path := queue[0]
		queue = queue[1:]
		path = filepath.Clean(path)
		if _, ok := seenFiles[path]; ok {
			continue
		}
		seenFiles[path] = struct{}{}

		libs, err := hooks.neededLibraries(path)
		if err != nil {
			return err
		}
		for _, lib := range libs {
			lib = strings.TrimSpace(lib)
			if lib == "" {
				continue
			}
			base := filepath.Base(lib)
			if _, ok := seenLibs[base]; ok {
				continue
			}
			seenLibs[base] = struct{}{}
			if _, ok := available[base]; ok {
				continue
			}

			dst := filepath.Join(runtimeDir, base)
			if info, err := os.Lstat(dst); err == nil && !info.IsDir() {
				available[base] = struct{}{}
				queue = append(queue, dst)
				continue
			}

			src, err := hooks.findSource(base, sourceRoots)
			if err != nil {
				missing[base] = struct{}{}
				continue
			}
			if err := hooks.copyFile(src, dst); err != nil {
				return fmt.Errorf("copy runtime dependency %q from %q: %w", base, src, err)
			}
			available[base] = struct{}{}
			queue = append(queue, dst)
		}
	}

	if len(missing) > 0 {
		libraries := make([]string, 0, len(missing))
		for lib := range missing {
			libraries = append(libraries, lib)
		}
		sort.Strings(libraries)
		return fmt.Errorf("runtime dependency closure incomplete for %s: missing %s", root, strings.Join(libraries, ", "))
	}

	return nil
}

func listRuntimePatchTargets(root, runtimeDir string) ([]string, error) {
	var targets []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path == runtimeDir {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		candidate := info.Mode()&0o111 != 0
		if !candidate {
			execLike, err := looksExecutableByContent(path)
			if err != nil {
				return err
			}
			candidate = execLike
		}
		if !candidate {
			return nil
		}
		isElf, err := isELF(path)
		if err != nil {
			return err
		}
		if !isElf {
			return nil
		}
		if info.Mode()&0o111 != 0 {
			targets = append(targets, path)
			return nil
		}
		targets = append(targets, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(targets)
	return targets, nil
}

func runtimeNeededLibraries(path string) ([]string, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open ELF %s: %w", path, err)
	}
	defer f.Close()

	libs, err := f.ImportedLibraries()
	if err != nil {
		return nil, fmt.Errorf("read DT_NEEDED from %s: %w", path, err)
	}
	return libs, nil
}

func findRuntimeSource(name string, sourceRoots []string) (string, error) {
	for _, root := range sourceRoots {
		candidate := filepath.Join(root, name)
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		ok, err := isELF(candidate)
		if err != nil || !ok {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("not found")
}

func copyRuntimeFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

func runtimeSourceRoots() []string {
	candidates := []string{
		"/data/data/com.termux/files/usr/glibc/lib",
	}
	for _, envKey := range []string{"TERMUX_PREFIX", "TERMUX__PREFIX", "PREFIX"} {
		if base := strings.TrimSpace(os.Getenv(envKey)); base != "" {
			candidates = append(candidates, filepath.Join(base, "glibc", "lib"))
		}
	}

	seen := map[string]struct{}{}
	roots := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if candidate == "." || candidate == string(filepath.Separator) {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue
		}
		roots = append(roots, candidate)
	}
	return roots
}
