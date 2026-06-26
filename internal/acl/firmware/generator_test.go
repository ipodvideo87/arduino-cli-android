package firmware

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arduino/arduino-cli/internal/acl/compatibility"
	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	paths "github.com/arduino/go-paths-helper"
	properties "github.com/arduino/go-properties-orderedmap"
	"github.com/stretchr/testify/require"
)

func TestBuildFirmwarePackageGeneratesStableArtifactsAndValidation(t *testing.T) {
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
		filepath.Join(root, "bootloader.bin"):                  []byte("bootloader"),
		filepath.Join(root, "boot_app0.bin"):                   []byte("boot_app0"),
	}
	for path, data := range files {
		require.NoError(t, os.WriteFile(path, data, 0o644))
	}

	props := properties.NewMap()
	props.Set("build.project_name", projectName)
	props.Set("runtime.platform.path", root)
	props.Set("build.bootloader_addr", "0x1000")
	props.Set("recipe.hooks.objcopy.postobjcopy.3.pattern", `esptool write_flash 0x1000 "`+filepath.Join(buildDir, projectName+".bootloader.bin")+`" 0x8000 "`+filepath.Join(buildDir, projectName+".partitions.bin")+`" 0xe000 "`+filepath.Join(root, "boot_app0.bin")+`" 0x10000 "`+filepath.Join(buildDir, projectName+".bin")+`"`)

	pkg, err := BuildFirmwarePackage(BuildInput{
		BuildPath:        paths.New(buildDir),
		OutputDir:        paths.New(outputDir),
		Properties:       props,
		SketchName:       "demo",
		ProjectName:      projectName,
		FQBN:             "esp32:esp32:esp32s3",
		Board:            "esp32s3",
		PlatformPackage:  "esp32",
		PlatformVersion:  "3.3.10",
		CoreVersion:      "3.3.10",
		ToolchainVersion: "gcc-14.2.0",
		TargetChip:       "ESP32-S3",
		Libraries:        []LibraryRef{{Name: "ESPAsyncWebServer", Version: "3.3.0"}},
		Compatibility: []compatibility.Decision{{
			Scope:           compatibility.ScopeLibrary,
			Subject:         "ESPAsyncWebServer",
			Outcome:         compatibility.OutcomeSelected,
			SelectedVersion: "3.3.0",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, "full-flash", pkg.Manifest.PackageMode)
	require.Equal(t, "full-flash", pkg.FlashPlan.PackageMode)
	require.Equal(t, diagnostics.StatusPassed, pkg.Validation.Status)
	require.Contains(t, pkg.Manifest.SketchName, "demo")
	require.Contains(t, pkg.Manifest.Artifacts[ArtifactApplicationBinary].Path, filepath.Join(outputDir, "artifacts"))
	require.Contains(t, pkg.Manifest.Artifacts[ArtifactBootloaderBinary].Path, "bootloader.bin")
	require.Len(t, pkg.FlashPlan.Entries, 4)
	require.Equal(t, uint32(0x10000), pkg.FlashPlan.SortedEntries()[3].Offset)
	require.FileExists(t, filepath.Join(outputDir, "manifest.json"))
	require.FileExists(t, filepath.Join(outputDir, "flash-plan.json"))
	require.FileExists(t, filepath.Join(outputDir, "validation-report.json"))
	require.FileExists(t, filepath.Join(outputDir, "analysis.json"))
	require.FileExists(t, filepath.Join(outputDir, "README_FLASHING.txt"))
}

func TestBuildFirmwarePackageUsesFlashArgsBootloaderLayout(t *testing.T) {
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
		filepath.Join(buildDir, "boot_app0.bin"):               []byte("boot_app0"),
		filepath.Join(buildDir, "flash_args"):                  []byte("0x1000 " + filepath.Join(buildDir, projectName+".bootloader.bin") + "\n0x8000 " + filepath.Join(buildDir, projectName+".partitions.bin") + "\n0xe000 " + filepath.Join(buildDir, "boot_app0.bin") + "\n0x10000 " + filepath.Join(buildDir, projectName+".bin") + "\n"),
	}
	for path, data := range files {
		require.NoError(t, os.WriteFile(path, data, 0o644))
	}

	props := properties.NewMap()
	props.Set("build.project_name", projectName)

	pkg, err := BuildFirmwarePackage(BuildInput{
		BuildPath:        paths.New(buildDir),
		OutputDir:        paths.New(outputDir),
		Properties:       props,
		SketchName:       "demo",
		ProjectName:      projectName,
		FQBN:             "esp32:esp32:esp32s3",
		Board:            "esp32s3",
		PlatformPackage:  "esp32",
		PlatformVersion:  "3.3.10",
		CoreVersion:      "3.3.10",
		ToolchainVersion: "gcc-14.2.0",
		TargetChip:       "ESP32-S3",
	})
	require.NoError(t, err)
	require.Equal(t, "full-flash", pkg.Manifest.PackageMode)
	require.Len(t, pkg.FlashPlan.Entries, 4)
	require.Equal(t, "flash_args", pkg.Analysis.Source.MetadataSource)
	require.Contains(t, pkg.ProfessionalDetails(), "metadata source: flash_args")
	require.FileExists(t, filepath.Join(outputDir, "artifacts", "bootloader.bin"))
	require.FileExists(t, filepath.Join(outputDir, "analysis.json"))
	require.FileExists(t, filepath.Join(outputDir, "README_FLASHING.txt"))
}

func TestBuildFirmwarePackageFallsBackToAppOnlyWhenBootloaderArtifactIsMissing(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "build")
	require.NoError(t, os.MkdirAll(buildDir, 0o755))
	projectName := "sketch.ino"
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".bin"), []byte("app"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".elf"), []byte("elf"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".partitions.bin"), []byte("partitions"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "boot_app0.bin"), []byte("boot_app0"), 0o644))

	props := properties.NewMap()
	props.Set("build.project_name", projectName)
	props.Set("build.bootloader_addr", "0x1000")
	props.Set("recipe.hooks.objcopy.postobjcopy.3.pattern", `esptool write_flash 0x1000 "`+filepath.Join(buildDir, projectName+".bootloader.bin")+`" 0x8000 "`+filepath.Join(buildDir, projectName+".partitions.bin")+`" 0xe000 "`+filepath.Join(root, "boot_app0.bin")+`" 0x10000 "`+filepath.Join(buildDir, projectName+".bin")+`"`)

	pkg, err := BuildFirmwarePackage(BuildInput{
		BuildPath:        paths.New(buildDir),
		Properties:       props,
		SketchName:       "demo",
		ProjectName:      projectName,
		FQBN:             "esp32:esp32:esp32s3",
		Board:            "esp32s3",
		CoreVersion:      "3.3.10",
		ToolchainVersion: "gcc-14.2.0",
		TargetChip:       "ESP32-S3",
	})
	require.NoError(t, err)
	require.Equal(t, "app-only", pkg.Manifest.PackageMode)
	require.Equal(t, diagnostics.StatusWarning, pkg.Validation.Status)
	require.Contains(t, pkg.Validation.Warnings[0], "bootloader")
	require.Contains(t, strings.Join(pkg.Analysis.Notes, "\n"), "bootloader artifact not found in build output")
}

func TestBuildFirmwarePackageAppOnlyWarnsWhenBootloaderIsAbsent(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "build")
	require.NoError(t, os.MkdirAll(buildDir, 0o755))
	projectName := "sketch.ino"
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".bin"), []byte("app"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".elf"), []byte("elf"), 0o644))

	props := properties.NewMap()
	props.Set("build.project_name", projectName)

	pkg, err := BuildFirmwarePackage(BuildInput{
		BuildPath:        paths.New(buildDir),
		Properties:       props,
		SketchName:       "demo",
		ProjectName:      projectName,
		FQBN:             "arduino:samd:mkr1000",
		Board:            "mkr1000",
		CoreVersion:      "1.8.13",
		ToolchainVersion: "gcc-11.3.0",
	})
	require.NoError(t, err)
	require.Equal(t, "app-only", pkg.Manifest.PackageMode)
	require.Equal(t, diagnostics.StatusWarning, pkg.Validation.Status)
	require.Contains(t, pkg.Validation.Warnings[0], "bootloader")
	require.Contains(t, pkg.Analysis.Notes, "bootloader metadata was not available; app-only package was selected")
}

func TestBuildFirmwarePackageFailsWhenExplicitFullFlashBootloaderIsMissing(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "build")
	require.NoError(t, os.MkdirAll(buildDir, 0o755))
	projectName := "sketch.ino"
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".bin"), []byte("app"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".elf"), []byte("elf"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".partitions.bin"), []byte("partitions"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "boot_app0.bin"), []byte("boot_app0"), 0o644))

	props := properties.NewMap()
	props.Set("build.project_name", projectName)
	props.Set("build.bootloader_addr", "0x1000")

	pkg, err := BuildFirmwarePackage(BuildInput{
		BuildPath:        paths.New(buildDir),
		Properties:       props,
		PackageMode:      "full-flash",
		SketchName:       "demo",
		ProjectName:      projectName,
		FQBN:             "esp32:esp32:esp32s3",
		Board:            "esp32s3",
		CoreVersion:      "3.3.10",
		ToolchainVersion: "gcc-14.2.0",
		TargetChip:       "ESP32-S3",
	})
	require.NoError(t, err)
	require.Equal(t, "full-flash", pkg.Manifest.PackageMode)
	require.Equal(t, diagnostics.StatusFailed, pkg.Validation.Status)
	require.NotEmpty(t, pkg.Validation.Errors)
	require.Contains(t, strings.ToLower(pkg.Validation.Errors[0]), "bootloader")
}

func TestBuildFirmwarePackageFailsWhenRequiredArtifactMissing(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "build")
	require.NoError(t, os.MkdirAll(buildDir, 0o755))
	projectName := "sketch.ino"
	require.NoError(t, os.WriteFile(filepath.Join(buildDir, projectName+".elf"), []byte("elf"), 0o644))

	props := properties.NewMap()
	props.Set("build.project_name", projectName)

	_, err := BuildFirmwarePackage(BuildInput{
		BuildPath:        paths.New(buildDir),
		Properties:       props,
		SketchName:       "demo",
		ProjectName:      projectName,
		FQBN:             "arduino:avr:uno",
		Board:            "uno",
		CoreVersion:      "1.8.6",
		ToolchainVersion: "gcc-7.3.0",
	})
	require.Error(t, err)
}
