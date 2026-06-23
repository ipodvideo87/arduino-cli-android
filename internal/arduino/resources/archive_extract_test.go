package resources

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestArchiveFSLinkFallsBackToCopyOnPermissionDenied(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source.txt")
	dst := filepath.Join(root, "nested", "target.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(src), 0o755))
	require.NoError(t, os.WriteFile(src, []byte("copy me"), 0o640))
	wantTime := time.Unix(1710000000, 0)
	require.NoError(t, os.Chtimes(src, wantTime, wantTime))

	fs := archiveFS{
		link: func(oldname, newname string) error {
			return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: syscall.EPERM}
		},
	}

	require.NoError(t, fs.Link(src, dst))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "copy me", string(got))
	info, err := os.Stat(dst)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
	require.WithinDuration(t, wantTime, info.ModTime(), time.Second)
	require.False(t, os.SameFile(mustStat(t, src), info), "fallback should copy, not hard-link")
}

func TestArchiveFSLinkReturnsContextWhenFallbackCopyFails(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source-dir")
	dst := filepath.Join(root, "target")
	require.NoError(t, os.MkdirAll(src, 0o755))

	fs := archiveFS{
		link: func(oldname, newname string) error {
			return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: syscall.EPERM}
		},
	}

	err := fs.Link(src, dst)
	require.Error(t, err)
	require.Contains(t, err.Error(), "create hard link")
	require.Contains(t, err.Error(), "copy fallback failed")
}

func TestExtractArchiveWithFSFallsBackToCopyForHardLinks(t *testing.T) {
	root := t.TempDir()
	archive := createTarArchiveWithHardLink(t)

	fs := archiveFS{
		link: func(oldname, newname string) error {
			return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: syscall.EPERM}
		},
	}

	require.NoError(t, extractArchiveWithFS(context.Background(), bytes.NewReader(archive), root, fs))

	data, err := os.ReadFile(filepath.Join(root, "pkg", "avr", "bin", "avr-gcc"))
	require.NoError(t, err)
	require.Equal(t, "tool-binary", string(data))

	info, err := os.Stat(filepath.Join(root, "pkg", "avr", "bin", "avr-gcc"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

func createTarArchiveWithHardLink(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	mustWriteTarFile(t, tw, &tar.Header{Name: "pkg/", Mode: 0o755, Typeflag: tar.TypeDir})
	mustWriteTarFile(t, tw, &tar.Header{Name: "pkg/avr/", Mode: 0o755, Typeflag: tar.TypeDir})
	mustWriteTarFile(t, tw, &tar.Header{Name: "pkg/avr/bin/", Mode: 0o755, Typeflag: tar.TypeDir})
	mustWriteTarFile(t, tw, &tar.Header{
		Name:     "pkg/avr/bin/avr-gcc-7.3.0",
		Mode:     0o755,
		Size:     int64(len("tool-binary")),
		Typeflag: tar.TypeReg,
	}, bytes.NewReader([]byte("tool-binary")))
	mustWriteTarFile(t, tw, &tar.Header{
		Name:     "pkg/avr/bin/avr-gcc",
		Mode:     0o755,
		Typeflag: tar.TypeLink,
		Linkname: "pkg/avr/bin/avr-gcc-7.3.0",
	})

	require.NoError(t, tw.Close())
	return buf.Bytes()
}

func mustWriteTarFile(t *testing.T, tw *tar.Writer, hdr *tar.Header, data ...*bytes.Reader) {
	t.Helper()
	require.NoError(t, tw.WriteHeader(hdr))
	if len(data) == 0 {
		return
	}
	_, err := io.Copy(tw, data[0])
	require.NoError(t, err)
}

func mustStat(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info
}
