// This file is part of arduino-cli.
//
// Copyright 2026 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the GNU General Public License version 3,
// which covers the main part of arduino-cli.
// The terms of this license can be found at:
// https://www.gnu.org/licenses/gpl-3.0.en.html
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package commands

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/arduino/arduino-cli/internal/arduino/cores"
	"github.com/arduino/arduino-cli/internal/arduino/cores/packagemanager"
	"github.com/arduino/arduino-cli/internal/arduino/resources"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
	"github.com/arduino/go-paths-helper"
	"github.com/stretchr/testify/require"
	"go.bug.st/downloader/v3"
	semver "go.bug.st/relaxed-semver"
)

func TestInstallToolPatchesBuiltinToolsAfterInstall(t *testing.T) {
	root := t.TempDir()
	downloadDir := paths.New(filepath.Join(root, "downloads"))
	tempDir := paths.New(filepath.Join(root, "temp"))
	packagesDir := paths.New(filepath.Join(root, "packages"))
	indexDir := paths.New(filepath.Join(root, "index"))

	archiveFileName := "ctags.tar"
	archiveRoot := "ctags-5.8.0"
	archiveData := buildTarArchive(t, archiveRoot, map[string][]byte{
		filepath.Join(archiveRoot, "ctags"): []byte("#!/bin/sh\necho ctags\n"),
	})
	archivePath := filepath.Join(downloadDir.String(), archiveFileName)
	require.NoError(t, os.MkdirAll(downloadDir.String(), 0o755))
	require.NoError(t, os.WriteFile(archivePath, archiveData, 0o644))

	checksum := sha256.Sum256(archiveData)
	downloadResource := &resources.DownloadResource{
		ArchiveFileName: archiveFileName,
		Checksum:        "SHA-256:" + hex.EncodeToString(checksum[:]),
		Size:            int64(len(archiveData)),
	}

	pkg := &cores.Package{
		Name:  "builtin",
		Tools: map[string]*cores.Tool{},
	}
	tool := &cores.Tool{
		Name:     "ctags",
		Package:  pkg,
		Releases: map[semver.NormalizedString]*cores.ToolRelease{},
	}
	pkg.Tools[tool.Name] = tool
	release := tool.GetOrCreateRelease(semver.ParseRelaxed("5.8.0"))
	release.Flavors = []*cores.Flavor{
		{
			OS:       "aarch64-linux-gnu",
			Resource: downloadResource,
		},
	}

	pm := packagemanager.NewBuilder(indexDir, packagesDir, nil, downloadDir, tempDir, "test-agent", downloader.GetDefaultConfig()).Build()

	var patchedRoot string
	originalPatch := patchAndroidInstalledTool
	patchAndroidInstalledTool = func(root string) error {
		patchedRoot = root
		return nil
	}
	t.Cleanup(func() {
		patchAndroidInstalledTool = originalPatch
	})

	noOpDownload := func(*rpc.DownloadProgress) {}
	noOpTask := func(*rpc.TaskProgress) {}

	require.NoError(t, installTool(context.Background(), pm, release, noOpDownload, noOpTask, resources.IntegrityCheckFull))

	require.NotEmpty(t, patchedRoot)
	require.Equal(t, filepath.Join(packagesDir.String(), "builtin", "tools", "ctags", "5.8.0"), patchedRoot)
	require.Equal(t, patchedRoot, release.InstallDir.String())
	require.FileExists(t, filepath.Join(patchedRoot, "ctags"))
}

func TestPatchBuiltinToolsForAndroidUsesBuiltinToolsRoot(t *testing.T) {
	root := paths.New(filepath.Join(t.TempDir(), "packages", "builtin", "tools"))
	require.NoError(t, os.MkdirAll(root.String(), 0o755))

	var patchedRoot string
	originalPatch := patchAndroidInstalledTool
	patchAndroidInstalledTool = func(root string) error {
		patchedRoot = root
		return nil
	}
	t.Cleanup(func() {
		patchAndroidInstalledTool = originalPatch
	})

	require.NoError(t, patchBuiltinToolsForAndroid(root))
	require.Equal(t, root.String(), patchedRoot)
}

func buildTarArchive(t *testing.T, root string, files map[string][]byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     root + "/",
		Mode:     0o755,
		Typeflag: tar.TypeDir,
	}))
	for name, data := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name:     filepath.ToSlash(name),
			Mode:     0o755,
			Size:     int64(len(data)),
			Typeflag: tar.TypeReg,
		}))
		_, err := io.Copy(tw, bytes.NewReader(data))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	return buf.Bytes()
}
