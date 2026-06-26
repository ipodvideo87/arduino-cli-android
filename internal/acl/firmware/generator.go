package firmware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/compatibility"
	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	paths "github.com/arduino/go-paths-helper"
	properties "github.com/arduino/go-properties-orderedmap"
)

type BuildInput struct {
	BuildPath          *paths.Path
	OutputDir          *paths.Path
	Properties         *properties.Map
	PackageMode        string
	SketchName         string
	ProjectName        string
	FQBN               string
	Board              string
	PlatformPackage    string
	PlatformVersion    string
	CoreVersion        string
	ToolchainVersion   string
	TargetChip         string
	TargetFamily       string
	Libraries          []LibraryRef
	MemoryUsage        MemoryUsage
	ExecutableSections []SectionUsage
	Compatibility      []compatibility.Decision
}

var flashPlanPattern = regexp.MustCompile(`(?i)(0x[0-9a-f]+)\s+"([^"]+)"`)

type buildMetadataResolution struct {
	Source           string
	FlashArgsPath    string
	BootloaderSource string
	BootloaderPath   string
	BootloaderOffset string
	Notes            []string
}

func BuildFirmwarePackage(input BuildInput) (FirmwarePackage, error) {
	if input.BuildPath == nil {
		return FirmwarePackage{}, fmt.Errorf("build path is required")
	}
	if input.Properties == nil {
		input.Properties = properties.NewMap()
	}
	if input.ProjectName == "" {
		input.ProjectName = firstBuildProperty(input.Properties, "build.project_name")
	}
	if input.SketchName == "" {
		input.SketchName = input.ProjectName
	}

	artifacts, resolution, err := collectBuildArtifacts(input.BuildPath, input.Properties, input.ProjectName)
	if err != nil {
		return FirmwarePackage{}, err
	}
	packageMode := packageModeForBuild(input.PackageMode, artifacts)

	manifest := BuildManifest{
		SchemaVersion:    "1",
		PackageMode:      packageMode,
		SketchName:       input.SketchName,
		ProjectName:      input.ProjectName,
		Board:            input.Board,
		FQBN:             input.FQBN,
		PlatformPackage:  input.PlatformPackage,
		PlatformVersion:  input.PlatformVersion,
		CoreVersion:      input.CoreVersion,
		Libraries:        append([]LibraryRef(nil), input.Libraries...),
		ToolchainVersion: input.ToolchainVersion,
		TargetChip:       input.TargetChip,
		TargetFamily:     input.TargetFamily,
		MemoryUsage:      input.MemoryUsage,
		Compatibility:    append([]compatibility.Decision(nil), input.Compatibility...),
	}
	if manifest.ToolchainVersion == "" {
		manifest.ToolchainVersion = firstBuildProperty(input.Properties, "compiler.path", "runtime.tools.gcc.path", "runtime.tools.xtensa-esp32-elf-gcc.path")
	}
	if manifest.BuildID == "" {
		manifest.BuildID = buildID(input.BuildPath.String(), manifest.FQBN, manifest.ProjectName)
	}
	if manifest.BuiltAt == "" {
		manifest.BuiltAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	manifest.Artifacts = artifacts

	flashPlan := buildFlashPlan(input.BuildPath, input.Properties, artifacts, manifest.TargetChip, manifest.PackageMode, resolution)
	flashPlan.PackageMode = manifest.PackageMode
	pkg := FirmwarePackage{
		Manifest:  manifest,
		FlashPlan: flashPlan,
	}
	pkg.Analysis = buildFirmwareAnalysis(input, manifest, flashPlan, resolution)
	pkg.Validation = NewBinaryValidator().Validate(pkg)
	if len(resolution.Notes) > 0 {
		pkg.Analysis.Notes = append(pkg.Analysis.Notes, resolution.Notes...)
		pkg.Validation.Warnings = appendUniqueStrings(pkg.Validation.Warnings, diagnosticWarningNotes(resolution.Notes)...)
		if pkg.Validation.Status == diagnostics.StatusPassed && len(pkg.Validation.Warnings) > 0 {
			pkg.Validation.Status = diagnostics.StatusWarning
		}
	}
	if strings.EqualFold(input.PackageMode, "") && strings.EqualFold(pkg.Manifest.PackageMode, "app-only") && !hasFullFlashArtifacts(artifacts) {
		warning := "bootloader metadata is incomplete; falling back to app-only package"
		pkg.Validation.Warnings = append(pkg.Validation.Warnings, warning)
		pkg.Validation.Status = diagnostics.StatusWarning
		pkg.Analysis.Notes = append(pkg.Analysis.Notes, warning)
	}

	if input.OutputDir != nil {
		var copyErr error
		pkg, copyErr = pkg.WriteToDir(input.OutputDir.String())
		if copyErr != nil {
			return FirmwarePackage{}, copyErr
		}
		pkg.Validation = NewBinaryValidator().Validate(pkg)
		if len(resolution.Notes) > 0 {
			pkg.Analysis.Notes = appendUniqueStrings(pkg.Analysis.Notes, resolution.Notes...)
			pkg.Validation.Warnings = appendUniqueStrings(pkg.Validation.Warnings, diagnosticWarningNotes(resolution.Notes)...)
		}
		if strings.EqualFold(input.PackageMode, "") && strings.EqualFold(pkg.Manifest.PackageMode, "app-only") && !hasFullFlashArtifacts(artifacts) {
			warning := "bootloader metadata is incomplete; falling back to app-only package"
			pkg.Analysis.Notes = appendUniqueStrings(pkg.Analysis.Notes, warning)
			pkg.Validation.Warnings = appendUniqueStrings(pkg.Validation.Warnings, warning)
			if pkg.Validation.Status == diagnostics.StatusPassed {
				pkg.Validation.Status = diagnostics.StatusWarning
			}
		}
		if err := pkg.writeMetadata(input.OutputDir.String()); err != nil {
			return FirmwarePackage{}, err
		}
	}

	return pkg, nil
}

func LoadFirmwarePackage(dir string) (FirmwarePackage, error) {
	if strings.TrimSpace(dir) == "" {
		return FirmwarePackage{}, fmt.Errorf("package directory is required")
	}
	root := filepath.Clean(dir)

	var pkg FirmwarePackage
	if err := readJSON(filepath.Join(root, "manifest.json"), &pkg.Manifest); err != nil {
		return FirmwarePackage{}, err
	}
	if err := readJSON(filepath.Join(root, "flash-plan.json"), &pkg.FlashPlan); err != nil {
		return FirmwarePackage{}, err
	}
	if err := readJSON(filepath.Join(root, "validation-report.json"), &pkg.Validation); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return FirmwarePackage{}, err
		}
		pkg.Validation = NewBinaryValidator().Validate(pkg)
	}
	if err := readJSON(filepath.Join(root, "analysis.json"), &pkg.Analysis); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return FirmwarePackage{}, err
		}
	}
	if data, err := os.ReadFile(filepath.Join(root, "README_FLASHING.txt")); err == nil {
		pkg.Readme = string(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return FirmwarePackage{}, err
	}
	if pkg.Validation.PackageName == "" {
		pkg.Validation.PackageName = firstNonEmpty(pkg.Manifest.ProjectName, pkg.Manifest.SketchName, pkg.Manifest.Board)
	}
	if pkg.Validation.Board == "" {
		pkg.Validation.Board = pkg.Manifest.Board
	}
	if pkg.Validation.FQBN == "" {
		pkg.Validation.FQBN = pkg.Manifest.FQBN
	}
	if pkg.Validation.TargetChip == "" {
		pkg.Validation.TargetChip = firstNonEmpty(pkg.Manifest.TargetChip, pkg.FlashPlan.TargetChip)
	}
	return pkg, nil
}

func (p FirmwarePackage) WriteToDir(dir string) (FirmwarePackage, error) {
	if strings.TrimSpace(dir) == "" {
		return p, fmt.Errorf("output directory is required")
	}
	outputRoot := filepath.Clean(dir)
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return FirmwarePackage{}, err
	}
	artifactDir := filepath.Join(outputRoot, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return FirmwarePackage{}, err
	}

	clone := p
	clone.Manifest.Artifacts = map[ArtifactKind]Artifact{}
	for _, kind := range p.ArtifactKinds() {
		artifact := p.Manifest.Artifacts[kind]
		dstPath := filepath.Join(artifactDir, stableArtifactFileName(kind, artifact.Path))
		if err := copyArtifactFile(artifact.Path, dstPath); err != nil {
			return FirmwarePackage{}, err
		}
		info, err := os.Stat(dstPath)
		if err != nil {
			return FirmwarePackage{}, err
		}
		hash, err := sha256File(dstPath)
		if err != nil {
			return FirmwarePackage{}, err
		}
		artifact.Path = dstPath
		artifact.Size = info.Size()
		artifact.SHA256 = hash
		artifact.Required = isRequiredPackageArtifact(kind, p.FlashPlan.RequiredArtifacts)
		clone.Manifest.Artifacts[kind] = artifact
	}
	clone.FlashPlan = rewriteFlashPlanPaths(p.FlashPlan, clone.Manifest.Artifacts)
	clone.Validation = p.Validation
	clone.Analysis = p.Analysis
	clone.Readme = p.Readme
	return clone, nil
}

func (p FirmwarePackage) writeMetadata(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("output directory is required")
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	flashPlanPath := filepath.Join(dir, "flash-plan.json")
	validationPath := filepath.Join(dir, "validation-report.json")
	analysisPath := filepath.Join(dir, "analysis.json")
	readmePath := filepath.Join(dir, "README_FLASHING.txt")

	if err := writeJSON(manifestPath, p.Manifest); err != nil {
		return err
	}
	if err := writeJSON(flashPlanPath, p.FlashPlan); err != nil {
		return err
	}
	if err := writeJSON(validationPath, p.Validation); err != nil {
		return err
	}
	if err := writeJSON(analysisPath, p.Analysis); err != nil {
		return err
	}
	readme := strings.TrimSpace(p.Readme)
	if readme == "" {
		readme = defaultFlashingReadme(p)
	}
	if err := os.WriteFile(readmePath, []byte(readme+"\n"), 0o644); err != nil {
		return err
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func collectBuildArtifacts(buildPath *paths.Path, props *properties.Map, projectName string) (map[ArtifactKind]Artifact, buildMetadataResolution, error) {
	artifacts := map[ArtifactKind]Artifact{}
	resolution := buildMetadataResolution{}
	if buildPath == nil {
		return artifacts, resolution, fmt.Errorf("build path is required")
	}
	if projectName == "" {
		return artifacts, resolution, fmt.Errorf("build.project_name is required")
	}

	sourcePaths := map[ArtifactKind]string{
		ArtifactApplicationBinary:    buildPath.Join(projectName + ".bin").String(),
		ArtifactBootloaderBinary:     buildPath.Join(projectName + ".bootloader.bin").String(),
		ArtifactELF:                  buildPath.Join(projectName + ".elf").String(),
		ArtifactMAP:                  buildPath.Join(projectName + ".map").String(),
		ArtifactPartitionTableBinary: buildPath.Join(projectName + ".partitions.bin").String(),
	}

	if entries, flashArgsPath := parseFlashArgsEntries(buildPath); len(entries) > 0 {
		resolution.Source = "flash_args"
		resolution.FlashArgsPath = flashArgsPath
		resolution.Notes = append(resolution.Notes, "flash metadata source: flash_args")
		for _, entry := range entries {
			if entry.Artifact == "" || entry.Path == "" {
				continue
			}
			sourcePaths[entry.Artifact] = entry.Path
			if entry.Artifact == ArtifactBootloaderBinary {
				resolution.BootloaderSource = "flash_args"
				resolution.BootloaderPath = entry.Path
				resolution.BootloaderOffset = fmt.Sprintf("0x%x", entry.Offset)
			}
		}
	} else if entries := parsePatternEntries(props); len(entries) > 0 {
		resolution.Source = "build_properties"
		resolution.Notes = append(resolution.Notes, "flash metadata source: build properties")
		for _, entry := range entries {
			if entry.Artifact == "" || entry.Path == "" {
				continue
			}
			sourcePaths[entry.Artifact] = entry.Path
			if entry.Artifact == ArtifactBootloaderBinary {
				resolution.BootloaderSource = "build_properties"
				resolution.BootloaderPath = entry.Path
				resolution.BootloaderOffset = fmt.Sprintf("0x%x", entry.Offset)
			}
		}
	} else {
		resolution.Source = "filesystem"
		resolution.Notes = append(resolution.Notes, "flash metadata source: filesystem fallback")
	}

	for kind, path := range resolveArtifactSourcesFromProperties(buildPath, props) {
		if strings.TrimSpace(path) == "" {
			continue
		}
		sourcePaths[kind] = path
	}

	for _, kind := range []ArtifactKind{
		ArtifactApplicationBinary,
		ArtifactBootloaderBinary,
		ArtifactPartitionTableBinary,
		ArtifactBootApp0Binary,
		ArtifactELF,
		ArtifactMAP,
	} {
		path, ok := sourcePaths[kind]
		if !ok || strings.TrimSpace(path) == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			if kind == ArtifactMAP || kind == ArtifactBootApp0Binary || kind == ArtifactBootloaderBinary || kind == ArtifactPartitionTableBinary {
				continue
			}
			return nil, resolution, fmt.Errorf("artifact %s: %w", kind, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, resolution, err
		}
		if info.Size() <= 0 {
			return nil, resolution, fmt.Errorf("artifact %s is empty", kind)
		}
		hash, err := sha256File(path)
		if err != nil {
			return nil, resolution, err
		}
		artifacts[kind] = Artifact{
			Kind:     kind,
			Path:     path,
			SHA256:   hash,
			Size:     info.Size(),
			Required: kind == ArtifactApplicationBinary || kind == ArtifactBootloaderBinary || kind == ArtifactPartitionTableBinary || kind == ArtifactELF,
		}
	}

	if _, ok := artifacts[ArtifactApplicationBinary]; !ok {
		return nil, resolution, fmt.Errorf("artifact %s is required", ArtifactApplicationBinary)
	}
	if _, ok := artifacts[ArtifactELF]; !ok {
		return nil, resolution, fmt.Errorf("artifact %s is required", ArtifactELF)
	}
	if artifact, ok := artifacts[ArtifactBootloaderBinary]; ok {
		resolution.BootloaderPath = artifact.Path
		if resolution.BootloaderSource == "" {
			resolution.BootloaderSource = "artifact"
		}
	}
	if _, ok := artifacts[ArtifactBootloaderBinary]; !ok {
		reason := "bootloader artifact not found in build output"
		if resolution.BootloaderSource != "" {
			reason += "; metadata source was " + resolution.BootloaderSource
		} else if firstBuildProperty(props, "build.bootloader.file", "bootloader.file") == "" {
			reason += "; bootloader file metadata was not provided"
		}
		if firstBuildProperty(props, "build.bootloader_addr") == "" {
			reason += "; bootloader offset metadata was not provided"
		}
		resolution.Notes = append(resolution.Notes, reason)
	}
	return artifacts, resolution, nil
}

func buildFlashPlan(buildPath *paths.Path, props *properties.Map, artifacts map[ArtifactKind]Artifact, targetChip string, packageMode string, resolution buildMetadataResolution) FlashPlan {
	plan := FlashPlan{
		TargetChip:        targetChip,
		RequiredArtifacts: requiredFlashArtifacts(artifacts, packageMode),
	}
	if len(artifacts) == 0 {
		return plan
	}

	if strings.EqualFold(packageMode, "app-only") {
		plan.Entries = append(plan.Entries, defaultFlashPlanEntries(artifacts, packageMode)...)
	} else {
		if entries, _ := parseFlashArgsEntries(buildPath); len(entries) > 0 {
			plan.Entries = append(plan.Entries, entries...)
		} else if entries := parsePatternEntries(props); len(entries) > 0 {
			plan.Entries = append(plan.Entries, entries...)
		} else {
			plan.Entries = append(plan.Entries, defaultFlashPlanEntries(artifacts, packageMode)...)
		}
	}
	sort.SliceStable(plan.Entries, func(i, j int) bool {
		if plan.Entries[i].Offset != plan.Entries[j].Offset {
			return plan.Entries[i].Offset < plan.Entries[j].Offset
		}
		return plan.Entries[i].Artifact < plan.Entries[j].Artifact
	})
	if len(resolution.Notes) > 0 {
		plan.Notes = append(plan.Notes, resolution.Notes...)
	}
	return plan
}

func parsePatternEntries(props *properties.Map) []FlashPlanEntry {
	if props == nil {
		return nil
	}
	keys := props.Keys()
	sort.Strings(keys)
	for _, key := range keys {
		if !strings.HasSuffix(key, ".pattern") {
			continue
		}
		value := strings.TrimSpace(props.ExpandPropsInString(props.Get(key)))
		if value == "" {
			continue
		}
		matches := flashPlanPattern.FindAllStringSubmatch(value, -1)
		if len(matches) == 0 {
			continue
		}
		entries := make([]FlashPlanEntry, 0, len(matches))
		for _, match := range matches {
			offset, err := parseHexOffset(match[1])
			if err != nil {
				continue
			}
			path := strings.TrimSpace(match[2])
			if path == "" {
				continue
			}
			entries = append(entries, FlashPlanEntry{
				Offset:      offset,
				Artifact:    inferArtifactKind(path, offset),
				Path:        filepath.Clean(path),
				Required:    true,
				Description: fmt.Sprintf("derived from %s", key),
			})
		}
		if len(entries) > 0 {
			return entries
		}
	}
	return nil
}

func parseFlashArgsEntries(buildPath *paths.Path) ([]FlashPlanEntry, string) {
	if buildPath == nil {
		return nil, ""
	}
	flashArgsPath := buildPath.Join("flash_args").String()
	data, err := os.ReadFile(flashArgsPath)
	if err != nil {
		return nil, ""
	}
	lines := strings.Split(string(data), "\n")
	entries := make([]FlashPlanEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		offset, err := parseHexOffset(fields[0])
		if err != nil {
			continue
		}
		path := strings.Trim(fields[1], `"'`)
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			path = buildPath.Join(path).String()
		}
		entries = append(entries, FlashPlanEntry{
			Offset:      offset,
			Artifact:    inferArtifactKind(path, offset),
			Path:        filepath.Clean(path),
			Required:    true,
			Description: "derived from flash_args",
		})
	}
	if len(entries) == 0 {
		return nil, flashArgsPath
	}
	return entries, flashArgsPath
}

func defaultFlashPlanEntries(artifacts map[ArtifactKind]Artifact, packageMode string) []FlashPlanEntry {
	entries := []FlashPlanEntry{}
	add := func(offset uint32, kind ArtifactKind, required bool) {
		artifact, ok := artifacts[kind]
		if !ok || strings.TrimSpace(artifact.Path) == "" {
			return
		}
		entries = append(entries, FlashPlanEntry{
			Offset:   offset,
			Artifact: kind,
			Path:     artifact.Path,
			Required: required,
		})
	}
	add(0x10000, ArtifactApplicationBinary, true)
	if packageMode == "full-flash" {
		add(0x1000, ArtifactBootloaderBinary, true)
		add(0x8000, ArtifactPartitionTableBinary, true)
		add(0xE000, ArtifactBootApp0Binary, true)
	}
	return entries
}

func resolveArtifactSourcesFromProperties(buildPath *paths.Path, props *properties.Map) map[ArtifactKind]string {
	resolved := map[ArtifactKind]string{}
	for _, entry := range parsePatternEntries(props) {
		if entry.Artifact == "" || entry.Path == "" {
			continue
		}
		resolved[entry.Artifact] = entry.Path
	}
	if buildPath == nil || props == nil {
		return resolved
	}
	platformPath := props.Get("runtime.platform.path")
	if platformPath != "" {
		if _, ok := resolved[ArtifactBootApp0Binary]; !ok {
			if path := filepath.Join(platformPath, "tools", "partitions", "boot_app0.bin"); fileExists(path) {
				resolved[ArtifactBootApp0Binary] = path
			}
		}
		if _, ok := resolved[ArtifactBootloaderBinary]; !ok {
			if bootloader := firstBuildProperty(props, "build.bootloader.file", "bootloader.file"); bootloader != "" {
				candidates := []string{
					filepath.Join(platformPath, "bootloaders", bootloader),
					filepath.Join(platformPath, "tools", "sdk", "bin", bootloader),
				}
				for _, candidate := range candidates {
					if fileExists(candidate) {
						resolved[ArtifactBootloaderBinary] = candidate
						break
					}
				}
			}
		}
	}
	return resolved
}

func packageModeForBuild(requested string, artifacts map[ArtifactKind]Artifact) string {
	mode := strings.ToLower(strings.TrimSpace(requested))
	if mode != "" {
		return mode
	}
	if hasFullFlashArtifacts(artifacts) {
		return "full-flash"
	}
	return "app-only"
}

func hasFullFlashArtifacts(artifacts map[ArtifactKind]Artifact) bool {
	if len(artifacts) == 0 {
		return false
	}
	required := []ArtifactKind{ArtifactApplicationBinary, ArtifactBootloaderBinary, ArtifactPartitionTableBinary, ArtifactBootApp0Binary}
	for _, kind := range required {
		artifact, ok := artifacts[kind]
		if !ok || strings.TrimSpace(artifact.Path) == "" {
			return false
		}
	}
	return true
}

func buildFirmwareAnalysis(input BuildInput, manifest BuildManifest, flashPlan FlashPlan, resolution buildMetadataResolution) FirmwareAnalysis {
	analysis := FirmwareAnalysis{
		SchemaVersion:    "1",
		PackageMode:      manifest.PackageMode,
		ProjectName:      manifest.ProjectName,
		SketchName:       manifest.SketchName,
		Board:            manifest.Board,
		FQBN:             manifest.FQBN,
		PlatformPackage:  manifest.PlatformPackage,
		PlatformVersion:  manifest.PlatformVersion,
		CoreVersion:      manifest.CoreVersion,
		ToolchainVersion: manifest.ToolchainVersion,
		TargetChip:       manifest.TargetChip,
		TargetFamily:     manifest.TargetFamily,
		BuildID:          manifest.BuildID,
		BuiltAt:          manifest.BuiltAt,
		Usage: FirmwareAnalysisUsage{
			ProgramUsedBytes:  manifest.MemoryUsage.ProgramUsedBytes,
			ProgramTotalBytes: manifest.MemoryUsage.ProgramTotalBytes,
			ProgramPercent:    manifest.MemoryUsage.ProgramPercent,
			RAMUsedBytes:      manifest.MemoryUsage.RAMUsedBytes,
			RAMTotalBytes:     manifest.MemoryUsage.RAMTotalBytes,
			RAMPercent:        manifest.MemoryUsage.RAMPercent,
		},
		Source: FirmwareAnalysisSource{
			MetadataSource:   resolution.Source,
			BootloaderSource: resolution.BootloaderSource,
			BootloaderPath:   resolution.BootloaderPath,
			BootloaderOffset: resolution.BootloaderOffset,
			FlashArgsPath:    resolution.FlashArgsPath,
			Notes:            append([]string(nil), resolution.Notes...),
		},
		LinkerSections:   append([]SectionUsage(nil), input.ExecutableSections...),
		LargestFunctions: FirmwareAnalysisBucket{Status: "unavailable", Reason: "largest functions analysis is not implemented yet"},
		LargestLibraries: FirmwareAnalysisBucket{Status: "unavailable", Reason: "largest libraries analysis is not implemented yet"},
		Symbols: FirmwareAnalysisSymbols{
			Status: "unavailable",
			Reason: "symbol analysis is not implemented yet",
		},
		Optimization: FirmwareAnalysisBucket{Status: "partial", Reason: "optimization metadata is derived from build properties only"},
		CallGraph:    FirmwareAnalysisBucket{Status: "unavailable", Reason: "call graph analysis is not implemented yet"},
		Extensions:   map[string]any{"schema_family": "firmware-package-analysis"},
	}
	if len(analysis.LinkerSections) == 0 {
		analysis.LinkerSections = []SectionUsage{
			{Name: "text", Size: int64(manifest.MemoryUsage.ProgramUsedBytes), MaxSize: int64(manifest.MemoryUsage.ProgramTotalBytes)},
			{Name: "data", Size: int64(manifest.MemoryUsage.RAMUsedBytes), MaxSize: int64(manifest.MemoryUsage.RAMTotalBytes)},
		}
	}
	if len(flashPlan.Notes) > 0 {
		analysis.Notes = append(analysis.Notes, flashPlan.Notes...)
	}
	if manifest.PackageMode == "app-only" && resolution.BootloaderPath == "" {
		analysis.Notes = append(analysis.Notes, "bootloader metadata was not available; app-only package was selected")
	}
	return analysis
}

func defaultFlashingReadme(pkg FirmwarePackage) string {
	packageMode := strings.TrimSpace(pkg.Manifest.PackageMode)
	if packageMode == "" {
		packageMode = "app-only"
	}
	validationStatus := string(pkg.Validation.Status)
	if validationStatus == "" {
		validationStatus = "pending"
	}
	lines := []string{
		"Firmware Package",
		"",
		"This package is the canonical compile output for flashing and build review.",
		"",
		"Package mode: " + packageMode,
		"Validation status: " + validationStatus,
		"",
		"Artifact guide:",
		"- application.bin: the sketch image to flash at 0x10000",
		"- firmware.elf: raw ELF for analysis and debugging",
		"- firmware.map: linker map for size and symbol inspection",
	}
	if packageMode == "full-flash" {
		lines = append(lines,
			"- bootloader.bin: flash at 0x1000",
			"- partitions.bin: flash at 0x8000",
			"- boot_app0.bin: flash at 0xE000",
		)
	}
	lines = append(lines, "", "Artifacts:")
	for _, kind := range pkg.ArtifactKinds() {
		artifact := pkg.Manifest.Artifacts[kind]
		lines = append(lines, fmt.Sprintf("- %s -> %s", kind, filepath.Base(artifact.Path)))
	}
	if len(pkg.FlashPlan.Entries) > 0 {
		lines = append(lines, "", "Flash plan:")
		for _, entry := range pkg.FlashPlan.SortedEntries() {
			lines = append(lines, fmt.Sprintf("- 0x%x %s", entry.Offset, entry.Artifact))
		}
	}
	if len(pkg.Validation.Warnings) > 0 {
		lines = append(lines, "", "Warnings:")
		for _, warning := range pkg.Validation.Warnings {
			lines = append(lines, "- "+warning)
		}
	}
	lines = append(lines,
		"",
		"Beginner flashing:",
		"1. Open manifest.json to confirm the package contents.",
		"2. Review validation-report.json for any warnings or errors.",
		"3. If the package is full-flash, use the flash plan offsets in flash-plan.json.",
		"",
		"Professional flashing:",
		"1. Use flash-plan.json as the authoritative offset map.",
		"2. Use analysis.json for build-size and memory context.",
		"3. Treat firmware.elf and firmware.map as the raw reference artifacts.",
	)
	return strings.Join(lines, "\n")
}

func appendUniqueStrings(dst []string, values ...string) []string {
	seen := map[string]struct{}{}
	for _, value := range dst {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		dst = append(dst, value)
	}
	return dst
}

func diagnosticWarningNotes(notes []string) []string {
	warnings := make([]string, 0, len(notes))
	for _, note := range notes {
		lower := strings.ToLower(note)
		if strings.Contains(lower, "warning") || strings.Contains(lower, "missing") || strings.Contains(lower, "falling back") {
			warnings = append(warnings, note)
		}
	}
	return warnings
}

func rewriteFlashPlanPaths(plan FlashPlan, artifacts map[ArtifactKind]Artifact) FlashPlan {
	clone := plan
	clone.Entries = append([]FlashPlanEntry(nil), plan.Entries...)
	for i := range clone.Entries {
		if artifact, ok := artifacts[clone.Entries[i].Artifact]; ok && strings.TrimSpace(artifact.Path) != "" {
			clone.Entries[i].Path = artifact.Path
		}
	}
	return clone
}

func stableArtifactFileName(kind ArtifactKind, original string) string {
	switch kind {
	case ArtifactApplicationBinary:
		return "application.bin"
	case ArtifactBootloaderBinary:
		return "bootloader.bin"
	case ArtifactPartitionTableBinary:
		return "partitions.bin"
	case ArtifactBootApp0Binary:
		return "boot_app0.bin"
	case ArtifactELF:
		return "firmware.elf"
	case ArtifactMAP:
		return "firmware.map"
	default:
		if base := filepath.Base(original); base != "." && base != string(filepath.Separator) {
			return base
		}
		return string(kind)
	}
}

func copyArtifactFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func parseHexOffset(value string) (uint32, error) {
	offset, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(strings.ToLower(value), "0x")), 16, 32)
	if err != nil {
		return 0, err
	}
	return uint32(offset), nil
}

func inferArtifactKind(path string, offset uint32) ArtifactKind {
	lower := strings.ToLower(filepath.Base(path))
	switch {
	case strings.Contains(lower, "boot_app0"):
		return ArtifactBootApp0Binary
	case strings.Contains(lower, "bootloader"):
		return ArtifactBootloaderBinary
	case strings.Contains(lower, "partitions"):
		return ArtifactPartitionTableBinary
	case strings.HasSuffix(lower, ".elf"):
		return ArtifactELF
	case strings.HasSuffix(lower, ".map"):
		return ArtifactMAP
	case offset == 0x1000:
		return ArtifactBootloaderBinary
	case offset == 0x8000:
		return ArtifactPartitionTableBinary
	case offset == 0xE000:
		return ArtifactBootApp0Binary
	default:
		return ArtifactApplicationBinary
	}
}

func isRequiredPackageArtifact(kind ArtifactKind, required []ArtifactKind) bool {
	for _, candidate := range required {
		if candidate == kind {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func firstBuildProperty(props *properties.Map, keys ...string) string {
	if props == nil {
		return ""
	}
	for _, key := range keys {
		if value := strings.TrimSpace(props.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func buildID(parts ...string) string {
	payload := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:8])
}

func requiredFlashArtifacts(artifacts map[ArtifactKind]Artifact, packageMode string) []ArtifactKind {
	required := []ArtifactKind{ArtifactApplicationBinary}
	if strings.EqualFold(packageMode, "full-flash") {
		required = append(required, ArtifactBootloaderBinary, ArtifactPartitionTableBinary, ArtifactBootApp0Binary)
	}
	return required
}
