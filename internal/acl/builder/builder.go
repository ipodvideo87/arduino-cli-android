package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
)

const (
	PackageFormatVersion = "1.0"

	ManifestFileName   = aclruntime.ManifestFileName
	MetadataFileName   = "metadata.json"
	ChecksumsFileName  = "checksums.json"
	VersionFileName    = "version"
	DefaultLoaderDir   = "loader"
	DefaultLibraryDir  = "lib"
	DefaultMetadataDir = "metadata"
)

type Builder struct{}

type PackageSpec struct {
	RuntimeName        string                     `json:"runtime_name,omitempty"`
	RuntimeID          string                     `json:"runtime_id,omitempty"`
	RuntimeVersion     string                     `json:"runtime_version"`
	Architecture       string                     `json:"architecture"`
	SupportedABIs      []string                   `json:"supported_abis"`
	CompatibilityLevel string                     `json:"compatibility_level"`
	CreatedAt          string                     `json:"created_at"`
	Loader             SourceAsset                `json:"loader"`
	Libraries          []SourceAsset              `json:"libraries,omitempty"`
	Build              aclruntime.BuildInfo       `json:"build"`
	Extensions         map[string]json.RawMessage `json:"extensions,omitempty"`
}

type SourceAsset struct {
	Name         string `json:"name"`
	SourcePath   string `json:"source_path"`
	RelativePath string `json:"relative_path"`
	SONAME       string `json:"soname,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Required     bool   `json:"required,omitempty"`
}

type PackageMetadata struct {
	PackageFormatVersion string                     `json:"package_format_version"`
	RuntimeName          string                     `json:"runtime_name,omitempty"`
	RuntimeID            string                     `json:"runtime_id"`
	RuntimeVersion       string                     `json:"runtime_version"`
	Architecture         string                     `json:"architecture"`
	SupportedABIs        []string                   `json:"supported_abis"`
	CompatibilityLevel   string                     `json:"compatibility_level"`
	CreatedAt            string                     `json:"created_at"`
	BuiltAt              string                     `json:"built_at"`
	Build                aclruntime.BuildInfo       `json:"build"`
	ManifestSHA256       string                     `json:"manifest_sha256"`
	Extensions           map[string]json.RawMessage `json:"extensions,omitempty"`
}

type PackageResult struct {
	RuntimeName  string
	RuntimeID    string
	PackageDir   string
	Manifest     aclruntime.Manifest
	Metadata     PackageMetadata
	Checksums    map[string]string
	PackageFiles []string
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Validate(spec PackageSpec) error {
	if strings.TrimSpace(spec.RuntimeVersion) == "" {
		return errors.New("runtime_version is required")
	}
	if strings.TrimSpace(spec.RuntimeID) != "" && filepath.Base(spec.RuntimeID) != spec.RuntimeID {
		return fmt.Errorf("runtime_id %q must not contain path separators", spec.RuntimeID)
	}
	if strings.TrimSpace(spec.Architecture) == "" {
		return errors.New("architecture is required")
	}
	if len(spec.SupportedABIs) == 0 {
		return errors.New("supported_abis is required")
	}
	if strings.TrimSpace(spec.CompatibilityLevel) == "" {
		return errors.New("compatibility_level is required")
	}
	if err := validateCompatibilityLevel(spec.CompatibilityLevel); err != nil {
		return err
	}
	if strings.TrimSpace(spec.CreatedAt) == "" {
		return errors.New("created_at is required")
	}
	if err := validateAsset(spec.Loader, true); err != nil {
		return fmt.Errorf("loader: %w", err)
	}
	if err := rejectSymlinkSource(spec.Loader.SourcePath); err != nil {
		return fmt.Errorf("loader: %w", err)
	}
	if _, err := os.Stat(spec.Loader.SourcePath); err != nil {
		return fmt.Errorf("loader: %w", err)
	}
	if len(spec.Libraries) == 0 {
		return errors.New("at least one library is required")
	}

	seen := map[string]struct{}{spec.Loader.RelativePath: struct{}{}}
	for i, lib := range spec.Libraries {
		if err := validateAsset(lib, false); err != nil {
			return fmt.Errorf("libraries[%d]: %w", i, err)
		}
		if err := rejectSymlinkSource(lib.SourcePath); err != nil {
			return fmt.Errorf("libraries[%d]: %w", i, err)
		}
		if _, err := os.Stat(lib.SourcePath); err != nil {
			return fmt.Errorf("libraries[%d]: %w", i, err)
		}
		if _, ok := seen[lib.RelativePath]; ok {
			return fmt.Errorf("duplicate runtime path %q", lib.RelativePath)
		}
		seen[lib.RelativePath] = struct{}{}
	}

	if err := validateABICompatibility(spec.Architecture, spec.SupportedABIs); err != nil {
		return err
	}
	return nil
}

func (b *Builder) AssignRuntimeID(spec PackageSpec) (string, error) {
	if strings.TrimSpace(spec.RuntimeID) != "" {
		return spec.RuntimeID, nil
	}
	digest, err := specDigest(spec)
	if err != nil {
		return "", err
	}
	arch := normalizeComponent(spec.Architecture)
	version := normalizeComponent(spec.RuntimeVersion)
	if arch == "" {
		arch = "unknown"
	}
	if version == "" {
		version = "0"
	}
	name := normalizeComponent(spec.RuntimeName)
	if name == "" {
		return fmt.Sprintf("acl-%s-%s-%s", arch, version, digest[:12]), nil
	}
	return fmt.Sprintf("acl-%s-%s-%s-%s", name, arch, version, digest[:12]), nil
}

func (b *Builder) GenerateManifest(spec PackageSpec) (aclruntime.Manifest, error) {
	if err := b.Validate(spec); err != nil {
		return aclruntime.Manifest{}, err
	}
	runtimeID, err := b.AssignRuntimeID(spec)
	if err != nil {
		return aclruntime.Manifest{}, err
	}

	loaderHash, err := hashFile(spec.Loader.SourcePath)
	if err != nil {
		return aclruntime.Manifest{}, err
	}

	libs := make([]aclruntime.RuntimeFile, 0, len(spec.Libraries))
	hashes := map[string]string{spec.Loader.RelativePath: loaderHash}
	for _, lib := range spec.Libraries {
		sum, err := hashFile(lib.SourcePath)
		if err != nil {
			return aclruntime.Manifest{}, err
		}
		hashes[lib.RelativePath] = sum
		libs = append(libs, aclruntime.RuntimeFile{
			Name:     lib.Name,
			Path:     lib.RelativePath,
			SONAME:   lib.SONAME,
			SHA256:   sum,
			Kind:     defaultKind(lib.Kind, "library"),
			Required: true,
		})
	}

	manifest := aclruntime.Manifest{
		SchemaVersion:  PackageFormatVersion,
		RuntimeID:      runtimeID,
		RuntimeVersion: spec.RuntimeVersion,
		Architecture:   spec.Architecture,
		SupportedABIs:  append([]string(nil), spec.SupportedABIs...),
		Loader: aclruntime.RuntimeFile{
			Name:     spec.Loader.Name,
			Path:     spec.Loader.RelativePath,
			SONAME:   spec.Loader.SONAME,
			SHA256:   loaderHash,
			Kind:     defaultKind(spec.Loader.Kind, "loader"),
			Required: true,
		},
		Libraries:          libs,
		Hashes:             hashes,
		CompatibilityLevel: spec.CompatibilityLevel,
		CreatedAt:          spec.CreatedAt,
		Build:              spec.Build,
		Extensions:         spec.Extensions,
	}
	if err := manifest.ValidateBasic(); err != nil {
		return aclruntime.Manifest{}, err
	}
	return manifest, nil
}

func (b *Builder) ComputeHashes(packageDir string) (map[string]string, error) {
	entries, err := listPackageFiles(packageDir)
	if err != nil {
		return nil, err
	}
	hashes := make(map[string]string, len(entries))
	for _, rel := range entries {
		sum, err := hashFile(filepath.Join(packageDir, rel))
		if err != nil {
			return nil, err
		}
		hashes[rel] = sum
	}
	return hashes, nil
}

func (b *Builder) Package(outputDir string, spec PackageSpec) (PackageResult, error) {
	manifest, err := b.GenerateManifest(spec)
	if err != nil {
		return PackageResult{}, err
	}
	if err := validatePackageOutput(outputDir, spec); err != nil {
		return PackageResult{}, err
	}
	if _, err := os.Stat(outputDir); err == nil {
		return PackageResult{}, fmt.Errorf("output path %q already exists", outputDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return PackageResult{}, err
	}
	if err := prepareEmptyDir(outputDir); err != nil {
		return PackageResult{}, err
	}
	if err := importAsset(outputDir, spec.Loader); err != nil {
		_ = os.RemoveAll(outputDir)
		return PackageResult{}, err
	}
	for _, lib := range spec.Libraries {
		if err := importAsset(outputDir, lib); err != nil {
			_ = os.RemoveAll(outputDir)
			return PackageResult{}, err
		}
	}

	manifestPath := filepath.Join(outputDir, ManifestFileName)
	if err := writeJSON(manifestPath, manifest); err != nil {
		_ = os.RemoveAll(outputDir)
		return PackageResult{}, err
	}

	metadata := PackageMetadata{
		PackageFormatVersion: PackageFormatVersion,
		RuntimeName:          spec.RuntimeName,
		RuntimeID:            manifest.RuntimeID,
		RuntimeVersion:       manifest.RuntimeVersion,
		Architecture:         manifest.Architecture,
		SupportedABIs:        append([]string(nil), manifest.SupportedABIs...),
		CompatibilityLevel:   manifest.CompatibilityLevel,
		CreatedAt:            manifest.CreatedAt,
		BuiltAt:              manifest.Build.BuiltAt,
		Build:                manifest.Build,
		Extensions:           manifest.Extensions,
	}
	manifestHash, err := hashFile(manifestPath)
	if err != nil {
		_ = os.RemoveAll(outputDir)
		return PackageResult{}, err
	}
	metadata.ManifestSHA256 = manifestHash
	if err := writeJSON(filepath.Join(outputDir, MetadataFileName), metadata); err != nil {
		_ = os.RemoveAll(outputDir)
		return PackageResult{}, err
	}
	versionPath := filepath.Join(outputDir, VersionFileName)
	if err := os.WriteFile(versionPath, []byte(PackageFormatVersion+"\n"), 0o644); err != nil {
		_ = os.RemoveAll(outputDir)
		return PackageResult{}, err
	}
	checksums, err := b.ComputeHashes(outputDir)
	if err != nil {
		_ = os.RemoveAll(outputDir)
		return PackageResult{}, err
	}
	if err := writeJSON(filepath.Join(outputDir, ChecksumsFileName), checksums); err != nil {
		_ = os.RemoveAll(outputDir)
		return PackageResult{}, err
	}
	packageFiles, err := listPackageFiles(outputDir)
	if err != nil {
		_ = os.RemoveAll(outputDir)
		return PackageResult{}, err
	}
	return PackageResult{
		RuntimeName:  spec.RuntimeName,
		RuntimeID:    manifest.RuntimeID,
		PackageDir:   outputDir,
		Manifest:     manifest,
		Metadata:     metadata,
		Checksums:    checksums,
		PackageFiles: packageFiles,
	}, nil
}

func (b *Builder) Verify(packageDir string) error {
	manifest, err := loadManifest(filepath.Join(packageDir, ManifestFileName))
	if err != nil {
		return err
	}
	if err := manifest.ValidateBasic(); err != nil {
		return err
	}

	metadata, err := loadMetadata(filepath.Join(packageDir, MetadataFileName))
	if err != nil {
		return err
	}
	if metadata.PackageFormatVersion != PackageFormatVersion {
		return fmt.Errorf("unexpected package_format_version %q", metadata.PackageFormatVersion)
	}
	if metadata.RuntimeID != manifest.RuntimeID {
		return fmt.Errorf("metadata runtime_id %q does not match manifest %q", metadata.RuntimeID, manifest.RuntimeID)
	}
	if metadata.RuntimeVersion != manifest.RuntimeVersion {
		return fmt.Errorf("metadata runtime_version %q does not match manifest %q", metadata.RuntimeVersion, manifest.RuntimeVersion)
	}
	manifestHash, err := hashFile(filepath.Join(packageDir, ManifestFileName))
	if err != nil {
		return err
	}
	if !strings.EqualFold(manifestHash, metadata.ManifestSHA256) {
		return fmt.Errorf("metadata manifest_sha256 mismatch: expected %s, got %s", metadata.ManifestSHA256, manifestHash)
	}

	versionData, err := os.ReadFile(filepath.Join(packageDir, VersionFileName))
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(versionData)) != PackageFormatVersion {
		return fmt.Errorf("unexpected version %q", strings.TrimSpace(string(versionData)))
	}

	checksums, err := loadChecksums(filepath.Join(packageDir, ChecksumsFileName))
	if err != nil {
		return err
	}
	for rel, want := range checksums {
		have, err := hashFile(filepath.Join(packageDir, rel))
		if err != nil {
			return err
		}
		if !strings.EqualFold(have, want) {
			return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", rel, want, have)
		}
	}
	return nil
}

func validateAsset(asset SourceAsset, loader bool) error {
	if strings.TrimSpace(asset.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(asset.SourcePath) == "" {
		return errors.New("source_path is required")
	}
	if strings.TrimSpace(asset.RelativePath) == "" {
		return errors.New("relative_path is required")
	}
	if err := validateRelativePath(asset.RelativePath); err != nil {
		return err
	}
	if loader && defaultKind(asset.Kind, "loader") != "loader" {
		return errors.New("loader kind must be loader")
	}
	return nil
}

func validateRelativePath(rel string) error {
	if filepath.IsAbs(rel) {
		return errors.New("must be relative")
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("must stay within the package root")
	}
	if clean != rel {
		return fmt.Errorf("must be clean relative path, got %q", rel)
	}
	return nil
}

func validateABICompatibility(arch string, abis []string) error {
	allowed := allowedABIsForArchitecture(arch)
	if len(allowed) == 0 {
		return fmt.Errorf("unsupported architecture %q", arch)
	}
	for _, abi := range abis {
		for _, want := range allowed {
			if strings.EqualFold(strings.TrimSpace(abi), want) {
				return nil
			}
		}
	}
	return fmt.Errorf("supported_abis %v are not compatible with architecture %q", abis, arch)
}

func validateCompatibilityLevel(level string) error {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "stable", "preview", "beta", "experimental":
		return nil
	default:
		return fmt.Errorf("unsupported compatibility_level %q", level)
	}
}

func allowedABIsForArchitecture(arch string) []string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "aarch64", "arm64":
		return []string{"arm64-v8a", "android-aarch64", "aarch64"}
	case "arm":
		return []string{"armeabi-v7a", "android-arm", "arm"}
	case "x86_64":
		return []string{"x86_64", "android-x86_64"}
	case "i386", "x86":
		return []string{"x86", "i686", "android-i686"}
	case "riscv64":
		return []string{"riscv64", "android-riscv64"}
	default:
		return nil
	}
}

func defaultKind(kind, fallback string) string {
	if strings.TrimSpace(kind) == "" {
		return fallback
	}
	return kind
}

func rejectSymlinkSource(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink source %q is not allowed", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source %q is not a regular file", path)
	}
	return nil
}

func importAsset(packageDir string, asset SourceAsset) error {
	dst := filepath.Join(packageDir, asset.RelativePath)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(asset.SourcePath)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func prepareEmptyDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func loadMetadata(path string) (PackageMetadata, error) {
	var metadata PackageMetadata
	data, err := os.ReadFile(path)
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return metadata, fmt.Errorf("decode metadata %q: %w", path, err)
	}
	return metadata, nil
}

func loadChecksums(path string) (map[string]string, error) {
	var checksums map[string]string
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &checksums); err != nil {
		return nil, fmt.Errorf("decode checksums %q: %w", path, err)
	}
	return checksums, nil
}

func loadManifest(path string) (aclruntime.Manifest, error) {
	return aclruntime.LoadManifest(path)
}

func listPackageFiles(packageDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(packageDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(packageDir, path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink package entry %q is not allowed", rel)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("package entry %q is not a regular file", rel)
		}
		switch rel {
		case ChecksumsFileName:
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func validatePackageOutput(outputDir string, spec PackageSpec) error {
	outAbs, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	checkWithin := func(sourcePath, label string) error {
		srcAbs, err := filepath.Abs(sourcePath)
		if err != nil {
			return err
		}
		root := filepath.Dir(srcAbs)
		rel, err := filepath.Rel(root, outAbs)
		if err != nil {
			return err
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return fmt.Errorf("output path %q must not be inside %s tree %q", outAbs, label, root)
		}
		return nil
	}
	if err := checkWithin(spec.Loader.SourcePath, "loader"); err != nil {
		return err
	}
	for _, lib := range spec.Libraries {
		if err := checkWithin(lib.SourcePath, "library"); err != nil {
			return err
		}
	}
	return nil
}

func specDigest(spec PackageSpec) (string, error) {
	type normalizedAsset struct {
		Name         string `json:"name"`
		RelativePath string `json:"relative_path"`
		SONAME       string `json:"soname,omitempty"`
		Kind         string `json:"kind,omitempty"`
		Required     bool   `json:"required,omitempty"`
	}
	payload := struct {
		RuntimeName        string               `json:"runtime_name"`
		RuntimeVersion     string               `json:"runtime_version"`
		Architecture       string               `json:"architecture"`
		SupportedABIs      []string             `json:"supported_abis"`
		CompatibilityLevel string               `json:"compatibility_level"`
		CreatedAt          string               `json:"created_at"`
		Loader             normalizedAsset      `json:"loader"`
		Libraries          []normalizedAsset    `json:"libraries"`
		Build              aclruntime.BuildInfo `json:"build"`
	}{
		RuntimeName:        spec.RuntimeName,
		RuntimeVersion:     spec.RuntimeVersion,
		Architecture:       spec.Architecture,
		SupportedABIs:      append([]string(nil), spec.SupportedABIs...),
		CompatibilityLevel: spec.CompatibilityLevel,
		CreatedAt:          spec.CreatedAt,
		Loader: normalizedAsset{
			Name:         spec.Loader.Name,
			RelativePath: spec.Loader.RelativePath,
			SONAME:       spec.Loader.SONAME,
			Kind:         spec.Loader.Kind,
			Required:     spec.Loader.Required,
		},
		Libraries: make([]normalizedAsset, 0, len(spec.Libraries)),
		Build:     spec.Build,
	}
	for _, lib := range spec.Libraries {
		payload.Libraries = append(payload.Libraries, normalizedAsset{
			Name:         lib.Name,
			RelativePath: lib.RelativePath,
			SONAME:       lib.SONAME,
			Kind:         lib.Kind,
			Required:     lib.Required,
		})
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func normalizeComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "_", "-")
	return value
}
