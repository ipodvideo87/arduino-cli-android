// This file is part of arduino-cli.
//
// Copyright 2024 ARDUINO SA (http://www.arduino.cc/)
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
	"os"
	"path/filepath"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/firmware"
	"github.com/arduino/arduino-cli/internal/arduino/sketch"
	paths "github.com/arduino/go-paths-helper"
	properties "github.com/arduino/go-properties-orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenBuildPath(t *testing.T) {
	srv := NewArduinoCoreServer().(*arduinoCoreServerImpl)
	want := srv.settings.GetBuildCachePath().Join("sketches", "ACBD18DB4CC2F85CEDEF654FCCC4A4D8")
	act := srv.getDefaultSketchBuildPath(&sketch.Sketch{FullPath: paths.New("foo")}, nil)
	assert.True(t, act.EquivalentTo(want))
	assert.Equal(t, "ACBD18DB4CC2F85CEDEF654FCCC4A4D8", (&sketch.Sketch{FullPath: paths.New("foo")}).Hash())
}

func TestBuildFirmwarePackageForCompilePropagatesTargetChip(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "build")
	outputDir := filepath.Join(root, "package")
	require.NoError(t, os.MkdirAll(buildDir, 0o755))

	projectName := "sketch.ino"
	files := map[string][]byte{
		filepath.Join(buildDir, projectName+".bin"):            []byte("app"),
		filepath.Join(buildDir, projectName+".elf"):            []byte("elf"),
		filepath.Join(buildDir, projectName+".map"):            []byte("map"),
		filepath.Join(buildDir, projectName+".bootloader.bin"): []byte("bootloader"),
		filepath.Join(buildDir, projectName+".partitions.bin"): []byte("partitions"),
		filepath.Join(root, "boot_app0.bin"):                   []byte("boot_app0"),
	}
	for path, data := range files {
		require.NoError(t, os.WriteFile(path, data, 0o644))
	}

	props := properties.NewMap()
	props.Set("build.project_name", projectName)
	props.Set("build.mcu", "esp32s3")
	props.Set("build.bootloader_addr", "0x1000")
	props.Set("runtime.platform.path", root)
	props.Set("recipe.hooks.objcopy.postobjcopy.3.pattern", `esptool write_flash 0x1000 "`+filepath.Join(buildDir, projectName+".bootloader.bin")+`" 0x8000 "`+filepath.Join(buildDir, projectName+".partitions.bin")+`" 0xe000 "`+filepath.Join(root, "boot_app0.bin")+`" 0x10000 "`+filepath.Join(buildDir, projectName+".bin")+`"`)

	pkg, err := buildFirmwarePackageForCompile(
		paths.New(buildDir),
		props,
		paths.New(outputDir),
		"demo",
		"esp32:esp32:esp32s3",
		"esp32s3",
		"esp32",
		"3.3.10",
		"3.3.10",
		nil,
		nil,
		"gcc-14.2.0",
	)
	require.NoError(t, err)
	require.Equal(t, "esp32s3", pkg.Manifest.TargetChip)
	require.Equal(t, "esp32s3", pkg.FlashPlan.TargetChip)
	require.Equal(t, diagnostics.StatusPassed, pkg.Validation.Status)
	require.NotContains(t, pkg.Validation.Warnings, "target chip metadata is not set")
	require.FileExists(t, filepath.Join(outputDir, "manifest.json"))
	require.FileExists(t, filepath.Join(outputDir, "flash-plan.json"))
	require.FileExists(t, filepath.Join(outputDir, "validation-report.json"))
	require.Equal(t, firmware.ResolveTargetChip(props, "esp32s3"), pkg.Manifest.TargetChip)
}
