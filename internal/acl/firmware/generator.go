package firmware

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	paths "github.com/arduino/go-paths-helper"
	properties "github.com/arduino/go-properties-orderedmap"
)

type BuildInput struct {
	BuildPath        *paths.Path
	OutputDir        *paths.Path
	Properties       *properties.Map
	SketchName       string
	ProjectName      string
	FQBN             string
	Board            string
	PlatformPackage  string
	PlatformVersion  string
	CoreVersion      string
	ToolchainVersion string
	TargetChip       string
	TargetFamily     string
	Libraries        []LibraryRef
	MemoryUsage      MemoryUsage
	Compatibility    []compatibility.Decision
}

var flashPlanPattern = regexp.MustCompile(`(?i)(0x[0-9a-f]+)\s+"([^"]+)"`)

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

	manifest := BuildManifest{
		SchemaVersion:    "1",
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

	artifacts, err := collectBuildArtifacts(input.BuildPath, input.Properties, manifest.ProjectName)
	if err != nil {
		return FirmwarePackage{}, err
	}
	manifest.Artifacts = artifacts

	flashPlan := buildFlashPlan(input.BuildPath, input.Properties, artifacts, manifest.TargetChip)
	pkg := FirmwarePackage{
		Manifest:  manifest,
		FlashPlan: flashPlan,
	}
	pkg.Validation = NewBinaryValidator().Validate(pkg)

	if input.OutputDir != nil {
		var copyErr error
		pkg, copyErr = pkg.WriteToDir(input.OutputDir.String())
		if copyErr != nil {
			return FirmwarePackage{}, copyErr
		}
		pkg.Validation = NewBinaryValidator().Validate(pkg)
		if err := pkg.writeMetadata(input.OutputDir.String()); err != nil {
			return FirmwarePackage{}, err
		}
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
	return clone, nil
}

func (p FirmwarePackage) writeMetadata(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("output directory is required")
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	flashPlanPath := filepath.Join(dir, "flash-plan.json")
	validationPath := filepath.Join(dir, "validation-report.json")

	if err := writeJSON(manifestPath, p.Manifest); err != nil {
		return err
	}
	if err := writeJSON(flashPlanPath, p.FlashPlan); err != nil {
		return err
	}
	if err := writeJSON(validationPath, p.Validation); err != nil {
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

func collectBuildArtifacts(buildPath *paths.Path, props *properties.Map, projectName string) (map[ArtifactKind]Artifact, error) {
	artifacts := map[ArtifactKind]Artifact{}
	if buildPath == nil {
		return artifacts, fmt.Errorf("build path is required")
	}
	if projectName == "" {
		return artifacts, fmt.Errorf("build.project_name is required")
	}

	sourcePaths := map[ArtifactKind]string{
		ArtifactApplicationBinary:    buildPath.Join(projectName + ".bin").String(),
		ArtifactELF:                  buildPath.Join(projectName + ".elf").String(),
		ArtifactMAP:                  buildPath.Join(projectName + ".map").String(),
		ArtifactPartitionTableBinary: buildPath.Join(projectName + ".partitions.bin").String(),
	}

	for kind, path := range resolveArtifactSourcesFromUploadPattern(buildPath, props) {
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
			if kind == ArtifactMAP || kind == ArtifactBootApp0Binary {
				continue
			}
			return nil, fmt.Errorf("artifact %s: %w", kind, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.Size() <= 0 {
			return nil, fmt.Errorf("artifact %s is empty", kind)
		}
		hash, err := sha256File(path)
		if err != nil {
			return nil, err
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
		return nil, fmt.Errorf("artifact %s is required", ArtifactApplicationBinary)
	}
	if _, ok := artifacts[ArtifactELF]; !ok {
		return nil, fmt.Errorf("artifact %s is required", ArtifactELF)
	}
	return artifacts, nil
}

func buildFlashPlan(buildPath *paths.Path, props *properties.Map, artifacts map[ArtifactKind]Artifact, targetChip string) FlashPlan {
	plan := FlashPlan{
		TargetChip:        targetChip,
		RequiredArtifacts: requiredFlashArtifacts(artifacts),
	}
	if len(artifacts) == 0 {
		return plan
	}

	if entries := parseUploadPatternEntries(props); len(entries) > 0 {
		plan.Entries = append(plan.Entries, entries...)
	} else {
		plan.Entries = append(plan.Entries, defaultFlashPlanEntries(artifacts)...)
	}
	sort.SliceStable(plan.Entries, func(i, j int) bool {
		if plan.Entries[i].Offset != plan.Entries[j].Offset {
			return plan.Entries[i].Offset < plan.Entries[j].Offset
		}
		return plan.Entries[i].Artifact < plan.Entries[j].Artifact
	})
	return plan
}

func parseUploadPatternEntries(props *properties.Map) []FlashPlanEntry {
	if props == nil {
		return nil
	}
	keys := props.Keys()
	sort.Strings(keys)
	for _, key := range keys {
		if !strings.HasSuffix(key, ".upload.pattern") && key != "upload.pattern" {
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

func defaultFlashPlanEntries(artifacts map[ArtifactKind]Artifact) []FlashPlanEntry {
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
	add(0x1000, ArtifactBootloaderBinary, true)
	add(0x8000, ArtifactPartitionTableBinary, true)
	if _, ok := artifacts[ArtifactBootApp0Binary]; ok {
		add(0xE000, ArtifactBootApp0Binary, false)
	}
	return entries
}

func resolveArtifactSourcesFromUploadPattern(buildPath *paths.Path, props *properties.Map) map[ArtifactKind]string {
	resolved := map[ArtifactKind]string{}
	for _, entry := range parseUploadPatternEntries(props) {
		if entry.Artifact == "" || entry.Path == "" {
			continue
		}
		resolved[entry.Artifact] = entry.Path
	}
	if len(resolved) > 0 {
		return resolved
	}
	if buildPath == nil || props == nil {
		return resolved
	}
	platformPath := props.Get("runtime.platform.path")
	if platformPath != "" {
		if path := filepath.Join(platformPath, "tools", "partitions", "boot_app0.bin"); fileExists(path) {
			resolved[ArtifactBootApp0Binary] = path
		}
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
	return resolved
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

func requiredFlashArtifacts(artifacts map[ArtifactKind]Artifact) []ArtifactKind {
	required := []ArtifactKind{ArtifactApplicationBinary}
	for _, kind := range []ArtifactKind{ArtifactBootloaderBinary, ArtifactPartitionTableBinary, ArtifactBootApp0Binary} {
		if _, ok := artifacts[kind]; ok {
			required = append(required, kind)
		}
	}
	return required
}
