package upload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/firmware"
	"github.com/arduino/arduino-cli/internal/acl/transport"
)

type PackageLoader interface {
	Load(string) (firmware.FirmwarePackage, error)
}

type PackageLoaderFunc func(string) (firmware.FirmwarePackage, error)

func (f PackageLoaderFunc) Load(dir string) (firmware.FirmwarePackage, error) {
	if f == nil {
		return firmware.FirmwarePackage{}, fmt.Errorf("package loader is not configured")
	}
	return f(dir)
}

type UploadEngine interface {
	Plan(context.Context, UploadRequest) (UploadPlan, error)
	DryRun(context.Context, UploadRequest) (UploadReport, error)
}

type Engine struct {
	Loader    PackageLoader
	Validator firmware.BinaryValidator
	Now       func() time.Time
}

func NewEngine() *Engine {
	return &Engine{
		Loader:    PackageLoaderFunc(firmware.LoadFirmwarePackage),
		Validator: firmware.NewBinaryValidator(),
		Now:       time.Now,
	}
}

type UploadRequest struct {
	PackageDir string                   `json:"package_dir,omitempty"`
	Package    firmware.FirmwarePackage `json:"package,omitempty"`
	Target     UploadTarget             `json:"target,omitempty"`
	Session    UploadSession            `json:"-"`
	Metadata   map[string]string        `json:"metadata,omitempty"`
}

type UploadTarget struct {
	Provider        string                   `json:"provider,omitempty"`
	Kind            string                   `json:"kind,omitempty"`
	Name            string                   `json:"name,omitempty"`
	Identifier      string                   `json:"identifier,omitempty"`
	TransportFamily string                   `json:"transport_family,omitempty"`
	Endpoint        transport.EndpointExport `json:"endpoint,omitempty"`
	Metadata        map[string]string        `json:"metadata,omitempty"`
}

type UploadStepKind string

const (
	UploadStepValidation UploadStepKind = "validation"
	UploadStepArtifact   UploadStepKind = "artifact"
	UploadStepSummary    UploadStepKind = "summary"
)

type UploadStep struct {
	Name        string                `json:"name,omitempty"`
	Kind        UploadStepKind        `json:"kind,omitempty"`
	Artifact    firmware.ArtifactKind `json:"artifact,omitempty"`
	Offset      uint32                `json:"offset,omitempty"`
	Path        string                `json:"path,omitempty"`
	Bytes       int64                 `json:"bytes,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Ordered     bool                  `json:"ordered,omitempty"`
	Description string                `json:"description,omitempty"`
	Warnings    []string              `json:"warnings,omitempty"`
	Limitations []string              `json:"limitations,omitempty"`
}

type UploadPlan struct {
	SchemaVersion string                   `json:"schema_version,omitempty"`
	PackageDir    string                   `json:"package_dir,omitempty"`
	PackageMode   string                   `json:"package_mode,omitempty"`
	TargetChip    string                   `json:"target_chip,omitempty"`
	Target        UploadTarget             `json:"target,omitempty"`
	Package       firmware.FirmwarePackage `json:"package,omitempty"`
	Steps         []UploadStep             `json:"steps,omitempty"`
	TotalBytes    int64                    `json:"total_bytes,omitempty"`
	Ordered       bool                     `json:"ordered,omitempty"`
	Complete      bool                     `json:"complete,omitempty"`
	Warnings      []string                 `json:"warnings,omitempty"`
	Limitations   []string                 `json:"limitations,omitempty"`
	Metadata      map[string]string        `json:"metadata,omitempty"`
}

type UploadProgressPhase string

const (
	UploadPhaseInspecting UploadProgressPhase = "inspecting"
	UploadPhaseValidating UploadProgressPhase = "validating"
	UploadPhasePlanning   UploadProgressPhase = "planning"
	UploadPhaseReady      UploadProgressPhase = "ready"
)

type UploadProgressEvent struct {
	Time     time.Time           `json:"time,omitempty"`
	Phase    UploadProgressPhase `json:"phase,omitempty"`
	Step     string              `json:"step,omitempty"`
	Status   diagnostics.Status  `json:"status,omitempty"`
	Progress int                 `json:"progress,omitempty"`
	Message  string              `json:"message,omitempty"`
	Evidence []string            `json:"evidence,omitempty"`
	Metadata map[string]string   `json:"metadata,omitempty"`
}

type UploadCheck struct {
	Name     string             `json:"name,omitempty"`
	Status   diagnostics.Status `json:"status,omitempty"`
	Message  string             `json:"message,omitempty"`
	Path     string             `json:"path,omitempty"`
	Required bool               `json:"required,omitempty"`
	Evidence []string           `json:"evidence,omitempty"`
}

type UploadDiagnostics struct {
	SchemaVersion    string             `json:"schema_version,omitempty"`
	Status           diagnostics.Status `json:"status,omitempty"`
	PackageDir       string             `json:"package_dir,omitempty"`
	PackageExists    bool               `json:"package_exists,omitempty"`
	ManifestExists   bool               `json:"manifest_exists,omitempty"`
	FlashPlanExists  bool               `json:"flash_plan_exists,omitempty"`
	ValidationExists bool               `json:"validation_exists,omitempty"`
	PackageMode      string             `json:"package_mode,omitempty"`
	TargetChip       string             `json:"target_chip,omitempty"`
	Checks           []UploadCheck      `json:"checks,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
	Limitations      []string           `json:"limitations,omitempty"`
	Beginner         string             `json:"beginner_summary,omitempty"`
	Professional     []string           `json:"professional_details,omitempty"`
	Metadata         map[string]string  `json:"metadata,omitempty"`
}

func (r UploadDiagnostics) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "upload dry-run completed"
	case diagnostics.StatusWarning:
		return "upload dry-run completed with warnings"
	case diagnostics.StatusFailed:
		return "upload dry-run failed"
	default:
		return "upload dry-run pending"
	}
}

func (r UploadDiagnostics) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if strings.TrimSpace(r.PackageDir) != "" {
		details = append(details, "package dir: "+r.PackageDir)
	}
	if r.PackageExists {
		details = append(details, "package exists: true")
	}
	if r.ManifestExists {
		details = append(details, "manifest exists: true")
	}
	if r.FlashPlanExists {
		details = append(details, "flash plan exists: true")
	}
	if r.ValidationExists {
		details = append(details, "validation report exists: true")
	}
	if strings.TrimSpace(r.PackageMode) != "" {
		details = append(details, "package mode: "+r.PackageMode)
	}
	if strings.TrimSpace(r.TargetChip) != "" {
		details = append(details, "target chip: "+r.TargetChip)
	}
	for _, check := range r.Checks {
		details = append(details, fmt.Sprintf("%s: %s", check.Name, check.Message))
	}
	if len(r.Warnings) > 0 {
		details = append(details, "warnings: "+strings.Join(r.Warnings, "; "))
	}
	if len(r.Limitations) > 0 {
		details = append(details, "limitations: "+strings.Join(r.Limitations, "; "))
	}
	return dedupeStrings(details)
}

type UploadResult struct {
	DryRun       bool               `json:"dry_run,omitempty"`
	Ready        bool               `json:"ready,omitempty"`
	PlannedBytes int64              `json:"planned_bytes,omitempty"`
	StepCount    int                `json:"step_count,omitempty"`
	Status       diagnostics.Status `json:"status,omitempty"`
	Beginner     string             `json:"beginner_summary,omitempty"`
	Professional []string           `json:"professional_details,omitempty"`
}

type UploadReport struct {
	SchemaVersion string                   `json:"schema_version,omitempty"`
	Status        diagnostics.Status       `json:"status,omitempty"`
	DryRun        bool                     `json:"dry_run,omitempty"`
	Request       UploadRequest            `json:"request,omitempty"`
	Package       firmware.FirmwarePackage `json:"package,omitempty"`
	Plan          UploadPlan               `json:"plan,omitempty"`
	Diagnostics   UploadDiagnostics        `json:"diagnostics,omitempty"`
	Result        UploadResult             `json:"result,omitempty"`
	Progress      []UploadProgressEvent    `json:"progress,omitempty"`
	Warnings      []string                 `json:"warnings,omitempty"`
	Limitations   []string                 `json:"limitations,omitempty"`
	Beginner      string                   `json:"beginner_summary,omitempty"`
	Professional  []string                 `json:"professional_details,omitempty"`
}

func (r UploadReport) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "upload dry-run ready"
	case diagnostics.StatusWarning:
		return "upload dry-run ready with warnings"
	case diagnostics.StatusFailed:
		return "upload dry-run failed"
	default:
		return "upload dry-run pending"
	}
}

func (r UploadReport) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if strings.TrimSpace(r.Plan.PackageDir) != "" {
		details = append(details, "package dir: "+r.Plan.PackageDir)
	}
	if r.Diagnostics.PackageExists {
		details = append(details, "package exists: true")
	}
	if r.Diagnostics.ManifestExists {
		details = append(details, "manifest exists: true")
	}
	if r.Diagnostics.FlashPlanExists {
		details = append(details, "flash plan exists: true")
	}
	if r.Diagnostics.ValidationExists {
		details = append(details, "validation report exists: true")
	}
	if strings.TrimSpace(r.Plan.PackageMode) != "" {
		details = append(details, "package mode: "+r.Plan.PackageMode)
	}
	if strings.TrimSpace(r.Plan.TargetChip) != "" {
		details = append(details, "target chip: "+r.Plan.TargetChip)
	}
	if r.Plan.Ordered {
		details = append(details, "flash plan ordered: true")
	} else {
		details = append(details, "flash plan ordered: false")
	}
	if r.Plan.Complete {
		details = append(details, "flash plan complete: true")
	} else {
		details = append(details, "flash plan complete: false")
	}
	if len(r.Plan.Steps) > 0 {
		details = append(details, fmt.Sprintf("upload steps: %d", len(r.Plan.Steps)))
		for _, step := range r.Plan.Steps {
			details = append(details, fmt.Sprintf("0x%x %s %s -> %s (%d bytes)", step.Offset, step.Kind, step.Artifact, step.Path, step.Bytes))
		}
	}
	if r.Plan.TotalBytes > 0 {
		details = append(details, fmt.Sprintf("estimated bytes: %d", r.Plan.TotalBytes))
	}
	if r.Diagnostics.Status != "" {
		details = append(details, "diagnostics status: "+string(r.Diagnostics.Status))
	}
	for _, check := range r.Diagnostics.Checks {
		details = append(details, fmt.Sprintf("check %s: %s", check.Name, check.Message))
	}
	for _, warning := range r.Diagnostics.Warnings {
		details = append(details, "warning: "+warning)
	}
	for _, limitation := range r.Diagnostics.Limitations {
		details = append(details, "limitation: "+limitation)
	}
	for _, warning := range r.Warnings {
		details = append(details, "warning: "+warning)
	}
	for _, limitation := range r.Limitations {
		details = append(details, "limitation: "+limitation)
	}
	if r.Result.DryRun {
		details = append(details, "dry-run: true")
	} else if r.Result.Status != "" {
		details = append(details, "dry-run: false")
	}
	if r.Result.Status != "" {
		details = append(details, "result status: "+string(r.Result.Status))
	}
	if r.Result.Ready {
		details = append(details, "ready: true")
	} else if r.Result.DryRun {
		details = append(details, "ready: false")
	}
	if r.Result.PlannedBytes > 0 {
		details = append(details, fmt.Sprintf("planned bytes: %d", r.Result.PlannedBytes))
	}
	if r.Result.StepCount > 0 {
		details = append(details, fmt.Sprintf("step count: %d", r.Result.StepCount))
	}
	return dedupeStrings(details)
}

func (r UploadResult) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	if r.DryRun {
		return "upload dry-run completed"
	}
	return "upload result available"
}

func (r UploadResult) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	details = append(details, fmt.Sprintf("dry-run: %t", r.DryRun))
	details = append(details, fmt.Sprintf("ready: %t", r.Ready))
	details = append(details, fmt.Sprintf("planned bytes: %d", r.PlannedBytes))
	details = append(details, fmt.Sprintf("step count: %d", r.StepCount))
	return dedupeStrings(details)
}

type UploadSession interface {
	Close() error
	Diagnostics() UploadDiagnostics
	Stream() (transport.TransportStream, error)
}

func (e *Engine) Plan(ctx context.Context, req UploadRequest) (UploadPlan, error) {
	report, err := e.plan(ctx, req)
	return report.Plan, err
}

func (e *Engine) DryRun(ctx context.Context, req UploadRequest) (UploadReport, error) {
	report, err := e.plan(ctx, req)
	report.Result = UploadResult{
		DryRun:       true,
		Ready:        report.Status != diagnostics.StatusFailed,
		PlannedBytes: report.Plan.TotalBytes,
		StepCount:    len(report.Plan.Steps),
		Status:       report.Status,
		Beginner:     report.BeginnerSummary(),
	}
	report.Beginner = report.Result.BeginnerSummary()
	report.Professional = report.ProfessionalDetails()
	if err != nil && report.Status == "" {
		report.Status = diagnostics.StatusFailed
	}
	return report, err
}

func (e *Engine) plan(ctx context.Context, req UploadRequest) (UploadReport, error) {
	report := UploadReport{
		SchemaVersion: "1",
		DryRun:        true,
		Request:       req,
		Warnings:      []string{},
		Limitations:   []string{},
		Progress: []UploadProgressEvent{
			{
				Time:     clockNow(e),
				Phase:    UploadPhaseInspecting,
				Status:   diagnostics.StatusRunning,
				Progress: 0,
				Message:  "inspecting firmware package",
			},
		},
	}

	pkg, diag, err := e.loadAndValidatePackage(ctx, req)
	if err != nil {
		report.Diagnostics = diag
		report.Status = diagnostics.StatusFailed
		report.Beginner = firstNonEmpty(diag.Beginner, err.Error(), "upload dry-run failed")
		report.Professional = append(report.Professional, diag.ProfessionalDetails()...)
		report.Professional = append(report.Professional, err.Error())
		report.Warnings = append(report.Warnings, diag.Warnings...)
		report.Limitations = append(report.Limitations, diag.Limitations...)
		report.Progress = append(report.Progress, UploadProgressEvent{
			Time:     clockNow(e),
			Phase:    UploadPhaseValidating,
			Status:   diagnostics.StatusFailed,
			Progress: 100,
			Message:  err.Error(),
		})
		return report, err
	}

	plan := buildUploadPlan(pkg, req, e)
	report.Package = pkg
	report.Plan = plan
	report.Diagnostics = diag
	report.Status = diagnostics.StatusPassed
	report.Warnings = append(report.Warnings, diag.Warnings...)
	report.Limitations = append(report.Limitations, diag.Limitations...)
	if len(plan.Warnings) > 0 {
		report.Warnings = append(report.Warnings, plan.Warnings...)
	}
	if len(plan.Limitations) > 0 {
		report.Limitations = append(report.Limitations, plan.Limitations...)
	}
	if len(report.Warnings) > 0 || report.Diagnostics.Status == diagnostics.StatusWarning || !plan.Ordered {
		report.Status = diagnostics.StatusWarning
	}
	if report.Diagnostics.Status == diagnostics.StatusFailed || !plan.Complete {
		report.Status = diagnostics.StatusFailed
	}
	report.Progress = append(report.Progress, UploadProgressEvent{
		Time:     clockNow(e),
		Phase:    UploadPhaseValidating,
		Status:   report.Diagnostics.Status,
		Progress: 50,
		Message:  "package validation completed",
	})
	for i, step := range plan.Steps {
		progress := 50
		if len(plan.Steps) > 0 {
			progress = 50 + int(((i+1)*40)/len(plan.Steps))
		}
		report.Progress = append(report.Progress, UploadProgressEvent{
			Time:     clockNow(e),
			Phase:    UploadPhasePlanning,
			Step:     step.Name,
			Status:   diagnostics.StatusPassed,
			Progress: progress,
			Message:  step.Description,
			Evidence: []string{step.Path},
			Metadata: map[string]string{
				"artifact": string(step.Artifact),
			},
		})
	}
	report.Progress = append(report.Progress, UploadProgressEvent{
		Time:     clockNow(e),
		Phase:    UploadPhaseReady,
		Status:   report.Status,
		Progress: 100,
		Message:  report.BeginnerSummary(),
	})
	report.Result = UploadResult{
		DryRun:       true,
		Ready:        report.Status != diagnostics.StatusFailed,
		PlannedBytes: plan.TotalBytes,
		StepCount:    len(plan.Steps),
		Status:       report.Status,
		Beginner:     report.BeginnerSummary(),
		Professional: report.ProfessionalDetails(),
	}
	report.Beginner = report.Result.BeginnerSummary()
	report.Professional = report.Result.ProfessionalDetails()
	if report.Status == diagnostics.StatusFailed {
		return report, errors.New(report.BeginnerSummary())
	}
	return report, nil
}

func (e *Engine) loadAndValidatePackage(ctx context.Context, req UploadRequest) (firmware.FirmwarePackage, UploadDiagnostics, error) {
	_ = ctx
	diag := UploadDiagnostics{
		SchemaVersion: "1",
		Status:        diagnostics.StatusPassed,
		PackageDir:    strings.TrimSpace(req.PackageDir),
		Checks:        []UploadCheck{},
		Warnings:      []string{},
		Limitations:   []string{},
		Metadata:      cloneStringMap(req.Metadata),
	}

	var pkg firmware.FirmwarePackage
	var err error
	if req.Package.Validate() == nil && len(req.Package.Manifest.Artifacts) > 0 {
		pkg = req.Package
		diag.PackageExists = true
		diag.ManifestExists = true
		diag.FlashPlanExists = true
		diag.ValidationExists = true
	} else {
		if strings.TrimSpace(req.PackageDir) == "" {
			diag.Status = diagnostics.StatusFailed
			err = fmt.Errorf("firmware package directory is required")
			diag.Checks = append(diag.Checks, UploadCheck{
				Name:     "package-directory",
				Status:   diagnostics.StatusFailed,
				Message:  err.Error(),
				Required: true,
			})
			return firmware.FirmwarePackage{}, diag, err
		}
		if _, statErr := os.Stat(req.PackageDir); statErr != nil {
			diag.Status = diagnostics.StatusFailed
			err = fmt.Errorf("firmware package directory is not available: %w", statErr)
			diag.Checks = append(diag.Checks, UploadCheck{
				Name:     "package-directory",
				Status:   diagnostics.StatusFailed,
				Message:  err.Error(),
				Path:     req.PackageDir,
				Required: true,
			})
			return firmware.FirmwarePackage{}, diag, err
		}
		diag.PackageExists = true
		if fileExists(filepath.Join(req.PackageDir, "manifest.json")) {
			diag.ManifestExists = true
		}
		if fileExists(filepath.Join(req.PackageDir, "flash-plan.json")) {
			diag.FlashPlanExists = true
		}
		if fileExists(filepath.Join(req.PackageDir, "validation-report.json")) {
			diag.ValidationExists = true
		}
		if e == nil || e.Loader == nil {
			err = fmt.Errorf("package loader is not configured")
		} else {
			pkg, err = e.Loader.Load(req.PackageDir)
		}
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "manifest.json"):
				err = fmt.Errorf("manifest is missing: %w", err)
			case strings.Contains(err.Error(), "flash-plan.json"):
				err = fmt.Errorf("flash plan is missing: %w", err)
			case strings.Contains(err.Error(), "validation-report.json"):
				err = fmt.Errorf("validation report is missing: %w", err)
			}
			diag.Status = diagnostics.StatusFailed
			diag.Checks = append(diag.Checks, UploadCheck{
				Name:     "package-load",
				Status:   diagnostics.StatusFailed,
				Message:  err.Error(),
				Path:     req.PackageDir,
				Required: true,
			})
			return firmware.FirmwarePackage{}, diag, err
		}
	}

	if pkg.Manifest.PackageMode == "" {
		pkg.Manifest.PackageMode = pkg.FlashPlan.PackageMode
	}
	diag.PackageMode = pkg.Manifest.PackageMode
	diag.TargetChip = firstNonEmpty(pkg.Manifest.TargetChip, pkg.FlashPlan.TargetChip)
	if strings.TrimSpace(req.PackageDir) != "" && !diag.ValidationExists {
		diag.Status = diagnostics.StatusWarning
		diag.Warnings = append(diag.Warnings, "validation report is missing; validation was recomputed from the package")
		diag.Checks = append(diag.Checks, UploadCheck{
			Name:     "validation-report",
			Status:   diagnostics.StatusWarning,
			Message:  "validation report is missing; validation was recomputed from the package",
			Path:     filepath.Join(req.PackageDir, "validation-report.json"),
			Required: false,
		})
	}

	if pkg.Validate() != nil {
		diag.Status = diagnostics.StatusFailed
		diag.Checks = append(diag.Checks, UploadCheck{
			Name:     "package-structure",
			Status:   diagnostics.StatusFailed,
			Message:  "firmware package is not structurally valid",
			Required: true,
		})
		err = fmt.Errorf("firmware package is not structurally valid")
		return pkg, diag, err
	}

	validator := e.Validator
	if validator == nil {
		validator = firmware.NewBinaryValidator()
	}
	validation := validator.Validate(pkg)
	if validation.HasFailures() {
		diag.Status = diagnostics.StatusFailed
		diag.Checks = append(diag.Checks, UploadCheck{
			Name:     "validation",
			Status:   diagnostics.StatusFailed,
			Message:  validation.BeginnerSummary(),
			Required: true,
		})
		err = errors.New(validation.BeginnerSummary())
		return pkg, diag, err
	}
	if validation.HasWarnings() {
		diag.Status = diagnostics.StatusWarning
		diag.Warnings = append(diag.Warnings, validation.Warnings...)
	}
	diag.Checks = append(diag.Checks, UploadCheck{
		Name:     "validation",
		Status:   validationStatus(validation),
		Message:  validation.BeginnerSummary(),
		Required: true,
	})

	if !planIsOrdered(pkg.FlashPlan) {
		diag.Status = diagnostics.StatusWarning
		diag.Warnings = append(diag.Warnings, "flash plan entries were reordered for dry-run planning")
		diag.Checks = append(diag.Checks, UploadCheck{
			Name:     "flash-plan-order",
			Status:   diagnostics.StatusWarning,
			Message:  "flash plan entries were reordered for dry-run planning",
			Required: false,
		})
	} else {
		diag.Checks = append(diag.Checks, UploadCheck{
			Name:     "flash-plan-order",
			Status:   diagnostics.StatusPassed,
			Message:  "flash plan entries are ordered",
			Required: false,
		})
	}

	requiredKinds := pkg.FlashPlan.RequiredArtifactKinds()
	for _, kind := range requiredKinds {
		artifact, ok := pkg.Artifact(kind)
		if !ok || strings.TrimSpace(artifact.Path) == "" {
			diag.Status = diagnostics.StatusFailed
			err = fmt.Errorf("required artifact %s is missing", kind)
			diag.Checks = append(diag.Checks, UploadCheck{
				Name:     fmt.Sprintf("artifact:%s", kind),
				Status:   diagnostics.StatusFailed,
				Message:  err.Error(),
				Required: true,
			})
			return pkg, diag, err
		}
		if fileExists(artifact.Path) {
			info, statErr := os.Stat(artifact.Path)
			if statErr != nil {
				diag.Status = diagnostics.StatusFailed
				err = fmt.Errorf("verify %s: %w", artifact.Path, statErr)
				diag.Checks = append(diag.Checks, UploadCheck{
					Name:     fmt.Sprintf("artifact:%s", kind),
					Status:   diagnostics.StatusFailed,
					Message:  err.Error(),
					Path:     artifact.Path,
					Required: true,
				})
				return pkg, diag, err
			}
			if info.Size() <= 0 {
				diag.Status = diagnostics.StatusFailed
				err = fmt.Errorf("artifact %s has zero size", kind)
				diag.Checks = append(diag.Checks, UploadCheck{
					Name:     fmt.Sprintf("artifact:%s", kind),
					Status:   diagnostics.StatusFailed,
					Message:  err.Error(),
					Path:     artifact.Path,
					Required: true,
				})
				return pkg, diag, err
			}
			diag.Checks = append(diag.Checks, UploadCheck{
				Name:     fmt.Sprintf("artifact:%s", kind),
				Status:   diagnostics.StatusPassed,
				Message:  fmt.Sprintf("found %d bytes", info.Size()),
				Path:     artifact.Path,
				Required: true,
				Evidence: []string{artifact.SHA256},
			})
		}
	}

	return pkg, diag, nil
}

func buildUploadPlan(pkg firmware.FirmwarePackage, req UploadRequest, e *Engine) UploadPlan {
	_ = e
	plan := UploadPlan{
		SchemaVersion: "1",
		PackageDir:    strings.TrimSpace(req.PackageDir),
		PackageMode:   pkg.Manifest.PackageMode,
		TargetChip:    firstNonEmpty(pkg.Manifest.TargetChip, pkg.FlashPlan.TargetChip),
		Target:        req.Target,
		Package:       pkg,
		Ordered:       planIsOrdered(pkg.FlashPlan),
		Complete:      true,
		Metadata:      cloneStringMap(req.Metadata),
	}

	for _, entry := range pkg.FlashPlan.SortedEntries() {
		artifact, ok := pkg.Artifact(entry.Artifact)
		if !ok {
			plan.Complete = false
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("missing artifact %s", entry.Artifact))
			continue
		}
		size := artifact.Size
		if size <= 0 {
			if info, err := os.Stat(artifact.Path); err == nil {
				size = info.Size()
			}
		}
		if size <= 0 {
			plan.Limitations = append(plan.Limitations, fmt.Sprintf("size unavailable for %s", artifact.Path))
		}
		plan.Steps = append(plan.Steps, UploadStep{
			Name:        artifactName(entry.Artifact),
			Kind:        UploadStepArtifact,
			Artifact:    entry.Artifact,
			Offset:      entry.Offset,
			Path:        entry.Path,
			Bytes:       size,
			Required:    entry.Required,
			Ordered:     plan.Ordered,
			Description: stepDescription(entry),
		})
		plan.TotalBytes += size
	}
	plan.Complete = plan.Complete && len(plan.Steps) == len(pkg.FlashPlan.SortedEntries())
	if !plan.Ordered {
		plan.Warnings = append(plan.Warnings, "flash plan entries were reordered for dry-run output")
	}
	if pkg.Manifest.PackageMode == "" && pkg.FlashPlan.PackageMode != "" {
		plan.PackageMode = pkg.FlashPlan.PackageMode
	}
	return plan
}

func validationStatus(report firmware.ValidationReport) diagnostics.Status {
	switch {
	case report.HasFailures():
		return diagnostics.StatusFailed
	case report.HasWarnings():
		return diagnostics.StatusWarning
	case report.Status == diagnostics.StatusPassed:
		return diagnostics.StatusPassed
	default:
		return diagnostics.StatusWarning
	}
}

func planIsOrdered(plan firmware.FlashPlan) bool {
	sorted := plan.SortedEntries()
	if len(sorted) != len(plan.Entries) {
		return false
	}
	for i := range sorted {
		if sorted[i] != plan.Entries[i] {
			return false
		}
	}
	return true
}

func artifactName(kind firmware.ArtifactKind) string {
	switch kind {
	case firmware.ArtifactApplicationBinary:
		return "application"
	case firmware.ArtifactBootloaderBinary:
		return "bootloader"
	case firmware.ArtifactPartitionTableBinary:
		return "partitions"
	case firmware.ArtifactBootApp0Binary:
		return "boot_app0"
	case firmware.ArtifactELF:
		return "firmware.elf"
	case firmware.ArtifactMAP:
		return "firmware.map"
	default:
		return string(kind)
	}
}

func stepDescription(entry firmware.FlashPlanEntry) string {
	if strings.TrimSpace(entry.Description) != "" {
		return entry.Description
	}
	return fmt.Sprintf("0x%x %s", entry.Offset, artifactName(entry.Artifact))
}

func clockNow(e *Engine) time.Time {
	if e != nil && e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
