package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/codeclysm/extract/v4"
)

type archiveFS struct {
	link func(oldname, newname string) error
}

func newArchiveFS() archiveFS {
	return archiveFS{
		link: os.Link,
	}
}

func (f archiveFS) Link(oldname, newname string) error {
	if f.link == nil {
		f.link = os.Link
	}

	err := f.link(oldname, newname)
	if err == nil {
		return nil
	}
	if !isUnsupportedHardLinkError(err) {
		return err
	}

	copyErr := copyFileWithMetadata(oldname, newname)
	if copyErr == nil {
		return nil
	}

	return fmt.Errorf("create hard link %s -> %s: %w; copy fallback failed: %v", oldname, newname, err, copyErr)
}

func (f archiveFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (f archiveFS) Symlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

func (f archiveFS) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

func (f archiveFS) Remove(path string) error {
	return os.Remove(path)
}

func (f archiveFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (f archiveFS) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

func extractArchiveWithFS(ctx context.Context, body io.Reader, location string, fs archiveFS) error {
	extractor := extract.Extractor{FS: fs}
	return extractor.Archive(ctx, body, location, nil)
}

// ExtractArchive extracts an archive using the default filesystem adapter.
// It preserves Linux behavior when hard links are supported and falls back to
// copying file contents when native Android/Termux rejects hard links.
func ExtractArchive(ctx context.Context, body io.Reader, location string) error {
	return extractArchiveWithFS(ctx, body, location, newArchiveFS())
}

var extractArchive = ExtractArchive

func copyFileWithMetadata(src, dst string) error {
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
	if err := os.Chmod(dst, info.Mode()); err != nil {
		return err
	}
	modTime := info.ModTime()
	return os.Chtimes(dst, modTime, modTime)
}

func isUnsupportedHardLinkError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrPermission) ||
		errors.Is(err, syscall.EPERM) ||
		errors.Is(err, syscall.EACCES) ||
		errors.Is(err, syscall.ENOTSUP) ||
		errors.Is(err, syscall.EOPNOTSUPP) ||
		errors.Is(err, syscall.EXDEV) ||
		errors.Is(err, syscall.ENOSYS)
}
