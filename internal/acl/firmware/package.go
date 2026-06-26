package firmware

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/compatibility"
	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
)

type ArtifactKind string

const (
	ArtifactApplicationBinary    ArtifactKind = "application-binary"
	ArtifactBootloaderBinary     ArtifactKind = "bootloader-binary"
	ArtifactPartitionTableBinary ArtifactKind = "partition-table-binary"
	ArtifactBootApp0Binary       ArtifactKind = "boot-app0-binary"
	ArtifactELF                  ArtifactKind = "elf"
	ArtifactMAP                  ArtifactKind = "map"
)

var defaultFlashArtifactKinds = []ArtifactKind{
	ArtifactApplicationBinary,
	ArtifactBootloaderBinary,
	ArtifactPartitionTableBinary,
}

var defaultRequiredPackageArtifactKinds = []ArtifactKind{
	ArtifactApplicationBinary,
	ArtifactELF,
}

var defaultOptionalPackageArtifactKinds = []ArtifactKind{
	ArtifactBootloaderBinary,
	ArtifactPartitionTableBinary,
	ArtifactBootApp0Binary,
	ArtifactMAP,
}

type Artifact struct {
	Kind     ArtifactKind `json:"kind"`
	Path     string       `json:"path"`
	SHA256   string       `json:"sha256,omitempty"`
	Size     int64        `json:"size,omitempty"`
	Required bool         `json:"required,omitempty"`
}

type LibraryRef struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Path    string `json:"path,omitempty"`
}

type MemoryUsage struct {
	ProgramUsedBytes  uint64 `json:"program_used_bytes,omitempty"`
	ProgramTotalBytes uint64 `json:"program_total_bytes,omitempty"`
	ProgramPercent    int    `json:"program_percent,omitempty"`
	RAMUsedBytes      uint64 `json:"ram_used_bytes,omitempty"`
	RAMTotalBytes     uint64 `json:"ram_total_bytes,omitempty"`
	RAMPercent        int    `json:"ram_percent,omitempty"`
}

type BuildManifest struct {
	SchemaVersion    string                    `json:"schema_version,omitempty"`
	SketchName       string                    `json:"sketch_name,omitempty"`
	ProjectName      string                    `json:"project_name,omitempty"`
	Board            string                    `json:"board"`
	FQBN             string                    `json:"fqbn"`
	PlatformPackage  string                    `json:"platform_package,omitempty"`
	PlatformVersion  string                    `json:"platform_version,omitempty"`
	CoreVersion      string                    `json:"core_version"`
	Libraries        []LibraryRef              `json:"libraries,omitempty"`
	ToolchainVersion string                    `json:"toolchain_version"`
	Artifacts        map[ArtifactKind]Artifact `json:"artifacts,omitempty"`
	MemoryUsage      MemoryUsage               `json:"memory_usage,omitempty"`
	TargetChip       string                    `json:"target_chip,omitempty"`
	TargetFamily     string                    `json:"target_family,omitempty"`
	BuiltAt          string                    `json:"built_at,omitempty"`
	BuildID          string                    `json:"build_id,omitempty"`
	Compatibility    []compatibility.Decision  `json:"compatibility,omitempty"`
}

type FlashPlanEntry struct {
	Offset      uint32       `json:"offset"`
	Artifact    ArtifactKind `json:"artifact"`
	Path        string       `json:"path"`
	Required    bool         `json:"required,omitempty"`
	Description string       `json:"description,omitempty"`
}

type FlashPlan struct {
	TargetChip        string           `json:"target_chip,omitempty"`
	TransportHint     string           `json:"transport_hint,omitempty"`
	RequiredArtifacts []ArtifactKind   `json:"required_artifacts,omitempty"`
	Entries           []FlashPlanEntry `json:"entries"`
	Notes             []string         `json:"notes,omitempty"`
}

type FirmwarePackage struct {
	Manifest   BuildManifest    `json:"manifest"`
	FlashPlan  FlashPlan        `json:"flash_plan"`
	Validation ValidationReport `json:"validation_report"`
}

type ValidationCheck struct {
	Name     string             `json:"name"`
	Status   diagnostics.Status `json:"status"`
	Message  string             `json:"message,omitempty"`
	Path     string             `json:"path,omitempty"`
	Required bool               `json:"required,omitempty"`
	Evidence []string           `json:"evidence,omitempty"`
}

type ValidationReport struct {
	PackageName string             `json:"package_name,omitempty"`
	Board       string             `json:"board,omitempty"`
	FQBN        string             `json:"fqbn,omitempty"`
	TargetChip  string             `json:"target_chip,omitempty"`
	Status      diagnostics.Status `json:"status"`
	GeneratedAt time.Time          `json:"generated_at,omitempty"`
	Checks      []ValidationCheck  `json:"checks,omitempty"`
	Warnings    []string           `json:"warnings,omitempty"`
	Errors      []string           `json:"errors,omitempty"`
}

type BinaryValidator interface {
	Validate(FirmwarePackage) ValidationReport
}

type DefaultBinaryValidator struct {
	Now  func() time.Time
	Stat func(string) (os.FileInfo, error)
}

func NewBinaryValidator() *DefaultBinaryValidator {
	return &DefaultBinaryValidator{
		Now:  time.Now,
		Stat: os.Stat,
	}
}

func (p FirmwarePackage) Artifact(kind ArtifactKind) (Artifact, bool) {
	if p.Manifest.Artifacts == nil {
		return Artifact{}, false
	}
	artifact, ok := p.Manifest.Artifacts[kind]
	return artifact, ok
}

func (p FirmwarePackage) ArtifactKinds() []ArtifactKind {
	kinds := make([]ArtifactKind, 0, len(p.Manifest.Artifacts))
	for kind := range p.Manifest.Artifacts {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}

func (p FirmwarePackage) Validate() error {
	if err := p.Manifest.Validate(); err != nil {
		return err
	}
	if err := p.FlashPlan.Validate(); err != nil {
		return err
	}
	return nil
}

func (m BuildManifest) Validate() error {
	switch {
	case strings.TrimSpace(m.Board) == "":
		return errors.New("board is required")
	case strings.TrimSpace(m.FQBN) == "":
		return errors.New("fqbn is required")
	case strings.TrimSpace(m.CoreVersion) == "":
		return errors.New("core_version is required")
	case strings.TrimSpace(m.ToolchainVersion) == "":
		return errors.New("toolchain_version is required")
	}
	if len(m.Artifacts) == 0 {
		return errors.New("artifacts are required")
	}
	for _, kind := range defaultRequiredPackageArtifactKinds {
		artifact, ok := m.Artifacts[kind]
		if !ok {
			return fmt.Errorf("artifact %s is required", kind)
		}
		if strings.TrimSpace(artifact.Path) == "" {
			return fmt.Errorf("artifact %s path is required", kind)
		}
		if strings.TrimSpace(artifact.SHA256) == "" {
			return fmt.Errorf("artifact %s sha256 is required", kind)
		}
	}
	return nil
}

func (m *BuildManifest) AddCompatibility(decisions ...compatibility.Decision) {
	if m == nil || len(decisions) == 0 {
		return
	}
	m.Compatibility = append(m.Compatibility, decisions...)
}

func (p FlashPlan) Validate() error {
	if len(p.Entries) == 0 {
		return errors.New("flash plan entries are required")
	}
	required := p.RequiredArtifacts
	if len(required) == 0 {
		required = defaultFlashArtifactKinds
	}
	seenOffsets := map[uint32]struct{}{}
	seenArtifacts := map[ArtifactKind]struct{}{}
	for i, entry := range p.Entries {
		if strings.TrimSpace(entry.Path) == "" {
			return fmt.Errorf("flash_plan.entries[%d].path is required", i)
		}
		if strings.TrimSpace(string(entry.Artifact)) == "" {
			return fmt.Errorf("flash_plan.entries[%d].artifact is required", i)
		}
		if _, ok := seenOffsets[entry.Offset]; ok {
			return fmt.Errorf("duplicate flash offset 0x%x", entry.Offset)
		}
		seenOffsets[entry.Offset] = struct{}{}
		seenArtifacts[entry.Artifact] = struct{}{}
	}
	for _, artifact := range required {
		if _, ok := seenArtifacts[artifact]; !ok {
			return fmt.Errorf("flash plan missing required artifact %s", artifact)
		}
	}
	return nil
}

func (p FlashPlan) SortedEntries() []FlashPlanEntry {
	entries := append([]FlashPlanEntry(nil), p.Entries...)
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Offset != entries[j].Offset {
			return entries[i].Offset < entries[j].Offset
		}
		return entries[i].Artifact < entries[j].Artifact
	})
	return entries
}

func (p FlashPlan) RequiredArtifactKinds() []ArtifactKind {
	if len(p.RequiredArtifacts) > 0 {
		return append([]ArtifactKind(nil), p.RequiredArtifacts...)
	}
	return append([]ArtifactKind(nil), defaultFlashArtifactKinds...)
}

func (r ValidationReport) HasFailures() bool {
	if r.Status == diagnostics.StatusFailed {
		return true
	}
	for _, check := range r.Checks {
		if check.Status == diagnostics.StatusFailed {
			return true
		}
	}
	return false
}

func (r ValidationReport) HasWarnings() bool {
	if r.Status == diagnostics.StatusWarning {
		return true
	}
	for _, check := range r.Checks {
		if check.Status == diagnostics.StatusWarning {
			return true
		}
	}
	return false
}

func (v *DefaultBinaryValidator) Validate(pkg FirmwarePackage) ValidationReport {
	now := time.Now().UTC()
	if v != nil && v.Now != nil {
		now = v.Now().UTC()
	}
	report := ValidationReport{
		PackageName: firstNonEmpty(pkg.Manifest.ProjectName, pkg.Manifest.SketchName, pkg.Manifest.Board),
		Board:       pkg.Manifest.Board,
		FQBN:        pkg.Manifest.FQBN,
		TargetChip:  firstNonEmpty(pkg.Manifest.TargetChip, pkg.FlashPlan.TargetChip),
		Status:      diagnostics.StatusRunning,
		GeneratedAt: now,
	}

	addCheck := func(check ValidationCheck) {
		if check.Status == "" {
			check.Status = diagnostics.StatusPassed
		}
		report.Checks = append(report.Checks, check)
		switch check.Status {
		case diagnostics.StatusFailed:
			report.Errors = append(report.Errors, check.Message)
		case diagnostics.StatusWarning:
			report.Warnings = append(report.Warnings, check.Message)
		}
	}

	if err := pkg.Manifest.Validate(); err != nil {
		addCheck(ValidationCheck{
			Name:     "manifest",
			Status:   diagnostics.StatusFailed,
			Message:  err.Error(),
			Required: true,
		})
	} else {
		addCheck(ValidationCheck{
			Name:     "manifest",
			Status:   diagnostics.StatusPassed,
			Message:  "build manifest is structurally valid",
			Required: true,
		})
	}

	if err := pkg.FlashPlan.Validate(); err != nil {
		addCheck(ValidationCheck{
			Name:     "flash-plan",
			Status:   diagnostics.StatusFailed,
			Message:  err.Error(),
			Required: true,
		})
	} else {
		addCheck(ValidationCheck{
			Name:     "flash-plan",
			Status:   diagnostics.StatusPassed,
			Message:  "flash plan is structurally valid",
			Required: true,
		})
	}

	requiredKinds := pkg.FlashPlan.RequiredArtifactKinds()
	for _, kind := range sortedArtifactKinds(pkg.Manifest.Artifacts) {
		artifact := pkg.Manifest.Artifacts[kind]
		if strings.TrimSpace(artifact.Path) == "" {
			addCheck(ValidationCheck{
				Name:     fmt.Sprintf("artifact:%s", kind),
				Status:   diagnostics.StatusFailed,
				Message:  "artifact path is empty",
				Required: true,
			})
			continue
		}
		info, err := statFile(v, artifact.Path)
		if err != nil {
			addCheck(ValidationCheck{
				Name:     fmt.Sprintf("artifact:%s", kind),
				Status:   diagnostics.StatusFailed,
				Message:  err.Error(),
				Path:     artifact.Path,
				Required: true,
			})
			continue
		}
		if info.Size() <= 0 {
			addCheck(ValidationCheck{
				Name:     fmt.Sprintf("artifact:%s", kind),
				Status:   diagnostics.StatusFailed,
				Message:  "artifact size is zero",
				Path:     artifact.Path,
				Required: true,
			})
			continue
		}
		addCheck(ValidationCheck{
			Name:     fmt.Sprintf("artifact:%s", kind),
			Status:   diagnostics.StatusPassed,
			Message:  fmt.Sprintf("found %d bytes", info.Size()),
			Path:     artifact.Path,
			Required: true,
			Evidence: []string{artifact.SHA256},
		})
	}

	for _, kind := range requiredKinds {
		artifact, ok := pkg.Artifact(kind)
		if !ok {
			addCheck(ValidationCheck{
				Name:     fmt.Sprintf("flash-plan:%s", kind),
				Status:   diagnostics.StatusFailed,
				Message:  "artifact is missing from manifest",
				Required: true,
			})
			continue
		}
		entry, ok := pkg.FlashPlan.entryForArtifact(kind)
		if !ok {
			addCheck(ValidationCheck{
				Name:     fmt.Sprintf("flash-plan:%s", kind),
				Status:   diagnostics.StatusFailed,
				Message:  "artifact is missing from flash plan",
				Required: true,
			})
			continue
		}
		if artifact.Path != entry.Path {
			addCheck(ValidationCheck{
				Name:     fmt.Sprintf("flash-plan:%s", kind),
				Status:   diagnostics.StatusFailed,
				Message:  "manifest path and flash plan path differ",
				Path:     entry.Path,
				Required: true,
			})
			continue
		}
		addCheck(ValidationCheck{
			Name:     fmt.Sprintf("flash-plan:%s", kind),
			Status:   diagnostics.StatusPassed,
			Message:  "flash plan entry matches manifest",
			Path:     entry.Path,
			Required: true,
		})
	}

	if pkg.Manifest.TargetChip == "" && pkg.FlashPlan.TargetChip == "" {
		addCheck(ValidationCheck{
			Name:     "target-chip",
			Status:   diagnostics.StatusWarning,
			Message:  "target chip metadata is not set",
			Required: false,
		})
	} else if pkg.Manifest.TargetChip != "" && pkg.FlashPlan.TargetChip != "" && pkg.Manifest.TargetChip != pkg.FlashPlan.TargetChip {
		addCheck(ValidationCheck{
			Name:     "target-chip",
			Status:   diagnostics.StatusFailed,
			Message:  "manifest and flash plan target chips differ",
			Required: true,
		})
	} else {
		addCheck(ValidationCheck{
			Name:     "target-chip",
			Status:   diagnostics.StatusPassed,
			Message:  "target chip metadata present",
			Required: false,
		})
	}

	report.Status = aggregateValidationStatus(report.Checks)
	return report
}

func statFile(v *DefaultBinaryValidator, path string) (os.FileInfo, error) {
	stat := os.Stat
	if v != nil && v.Stat != nil {
		stat = v.Stat
	}
	info, err := stat(path)
	if err != nil {
		return nil, fmt.Errorf("verify %s: %w", path, err)
	}
	return info, nil
}

func aggregateValidationStatus(checks []ValidationCheck) diagnostics.Status {
	hasWarning := false
	hasPassed := false
	for _, check := range checks {
		switch check.Status {
		case diagnostics.StatusFailed:
			return diagnostics.StatusFailed
		case diagnostics.StatusWarning:
			hasWarning = true
		case diagnostics.StatusPassed:
			hasPassed = true
		}
	}
	switch {
	case hasWarning:
		return diagnostics.StatusWarning
	case hasPassed:
		return diagnostics.StatusPassed
	default:
		return diagnostics.StatusPending
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sortedArtifactKinds(artifacts map[ArtifactKind]Artifact) []ArtifactKind {
	kinds := make([]ArtifactKind, 0, len(artifacts))
	for kind := range artifacts {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}

func (p FlashPlan) entryForArtifact(kind ArtifactKind) (FlashPlanEntry, bool) {
	for _, entry := range p.Entries {
		if entry.Artifact == kind {
			return entry, true
		}
	}
	return FlashPlanEntry{}, false
}
