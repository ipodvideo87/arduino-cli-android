package upload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/firmware"
	"github.com/arduino/arduino-cli/internal/acl/transport"
)

type UploadExecutor interface {
	Prepare(context.Context, UploadExecutionRequest) (UploadExecutionPlan, error)
	PrepareOnly(context.Context, UploadExecutionRequest) (UploadExecutionReport, error)
}

type Executor struct {
	Planner   UploadEngine
	Validator firmware.BinaryValidator
	Now       func() time.Time
}

func NewExecutor() *Executor {
	return &Executor{
		Planner:   NewEngine(),
		Validator: firmware.NewBinaryValidator(),
		Now:       time.Now,
	}
}

type UploadExecutionRequest struct {
	PackageDir    string                   `json:"package_dir,omitempty"`
	Package       firmware.FirmwarePackage `json:"package,omitempty"`
	Plan          UploadPlan               `json:"plan,omitempty"`
	Target        UploadTarget             `json:"target,omitempty"`
	Session       UploadExecutionSession   `json:"-"`
	RequireStream bool                     `json:"require_stream,omitempty"`
	PrepareOnly   bool                     `json:"prepare_only,omitempty"`
	Metadata      map[string]string        `json:"metadata,omitempty"`
}

type UploadOperationKind string

const (
	UploadOperationArtifact UploadOperationKind = "artifact"
)

type UploadOperation struct {
	Name        string                `json:"name,omitempty"`
	Kind        UploadOperationKind   `json:"kind,omitempty"`
	Artifact    firmware.ArtifactKind `json:"artifact,omitempty"`
	Offset      uint32                `json:"offset,omitempty"`
	Path        string                `json:"path,omitempty"`
	Size        int64                 `json:"size,omitempty"`
	SHA256      string                `json:"sha256,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Ordered     bool                  `json:"ordered,omitempty"`
	Description string                `json:"description,omitempty"`
	Warnings    []string              `json:"warnings,omitempty"`
	Limitations []string              `json:"limitations,omitempty"`
}

type UploadExecutionPlan struct {
	SchemaVersion   string            `json:"schema_version,omitempty"`
	PackageDir      string            `json:"package_dir,omitempty"`
	PackageMode     string            `json:"package_mode,omitempty"`
	TargetChip      string            `json:"target_chip,omitempty"`
	Target          UploadTarget      `json:"target,omitempty"`
	Operations      []UploadOperation `json:"operations,omitempty"`
	TotalBytes      int64             `json:"total_bytes,omitempty"`
	Ordered         bool              `json:"ordered,omitempty"`
	Complete        bool              `json:"complete,omitempty"`
	StreamRequired  bool              `json:"stream_required,omitempty"`
	StreamAvailable bool              `json:"stream_available,omitempty"`
	StreamStatus    string            `json:"stream_status,omitempty"`
	Warnings        []string          `json:"warnings,omitempty"`
	Limitations     []string          `json:"limitations,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type UploadExecutionProgressPhase string

const (
	UploadExecutionPhaseInspecting           UploadExecutionProgressPhase = "package-inspection"
	UploadExecutionPhaseLoadingPlan          UploadExecutionProgressPhase = "plan-loading"
	UploadExecutionPhaseArtifactValidation   UploadExecutionProgressPhase = "artifact-validation"
	UploadExecutionPhaseHashValidation       UploadExecutionProgressPhase = "hash-validation"
	UploadExecutionPhaseOperationPreparation UploadExecutionProgressPhase = "operation-preparation"
	UploadExecutionPhaseTransportReadiness   UploadExecutionProgressPhase = "transport-readiness"
	UploadExecutionPhasePrepared             UploadExecutionProgressPhase = "execution-prepared"
	UploadExecutionPhaseByteWriteStopped     UploadExecutionProgressPhase = "stopped-before-byte-write"
)

type UploadExecutionProgressEvent struct {
	Time     time.Time                    `json:"time,omitempty"`
	Phase    UploadExecutionProgressPhase `json:"phase,omitempty"`
	Step     string                       `json:"step,omitempty"`
	Status   diagnostics.Status           `json:"status,omitempty"`
	Progress int                          `json:"progress,omitempty"`
	Message  string                       `json:"message,omitempty"`
	Evidence []string                     `json:"evidence,omitempty"`
	Metadata map[string]string            `json:"metadata,omitempty"`
}

type UploadExecutionCheck struct {
	Name     string             `json:"name,omitempty"`
	Status   diagnostics.Status `json:"status,omitempty"`
	Message  string             `json:"message,omitempty"`
	Path     string             `json:"path,omitempty"`
	Required bool               `json:"required,omitempty"`
	Evidence []string           `json:"evidence,omitempty"`
}

type UploadExecutionSessionDiagnostics struct {
	SchemaVersion   string             `json:"schema_version,omitempty"`
	Status          diagnostics.Status `json:"status,omitempty"`
	StreamSupported bool               `json:"stream_supported,omitempty"`
	StreamAvailable bool               `json:"stream_available,omitempty"`
	Beginner        string             `json:"beginner_summary,omitempty"`
	Professional    []string           `json:"professional_details,omitempty"`
	Warnings        []string           `json:"warnings,omitempty"`
	Limitations     []string           `json:"limitations,omitempty"`
	Metadata        map[string]string  `json:"metadata,omitempty"`
}

func (r UploadExecutionSessionDiagnostics) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "transport session ready"
	case diagnostics.StatusWarning:
		return "transport session ready with limitations"
	case diagnostics.StatusFailed:
		return "transport session unavailable"
	default:
		return "transport session pending"
	}
}

func (r UploadExecutionSessionDiagnostics) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	details = append(details, fmt.Sprintf("stream supported: %t", r.StreamSupported))
	details = append(details, fmt.Sprintf("stream available: %t", r.StreamAvailable))
	for _, warning := range r.Warnings {
		details = append(details, "warning: "+warning)
	}
	for _, limitation := range r.Limitations {
		details = append(details, "limitation: "+limitation)
	}
	return dedupeStrings(details)
}

type UploadExecutionSession interface {
	Close() error
	Diagnostics() UploadExecutionSessionDiagnostics
	Stream() (transport.TransportStream, error)
}

type UploadExecutionDiagnostics struct {
	SchemaVersion    string                            `json:"schema_version,omitempty"`
	Status           diagnostics.Status                `json:"status,omitempty"`
	PackageDir       string                            `json:"package_dir,omitempty"`
	PackageExists    bool                              `json:"package_exists,omitempty"`
	ManifestExists   bool                              `json:"manifest_exists,omitempty"`
	FlashPlanExists  bool                              `json:"flash_plan_exists,omitempty"`
	ValidationExists bool                              `json:"validation_exists,omitempty"`
	PlanLoaded       bool                              `json:"plan_loaded,omitempty"`
	PackageMode      string                            `json:"package_mode,omitempty"`
	TargetChip       string                            `json:"target_chip,omitempty"`
	StreamRequired   bool                              `json:"stream_required,omitempty"`
	StreamAvailable  bool                              `json:"stream_available,omitempty"`
	StreamStatus     string                            `json:"stream_status,omitempty"`
	Session          UploadExecutionSessionDiagnostics `json:"session,omitempty"`
	Checks           []UploadExecutionCheck            `json:"checks,omitempty"`
	Warnings         []string                          `json:"warnings,omitempty"`
	Limitations      []string                          `json:"limitations,omitempty"`
	Beginner         string                            `json:"beginner_summary,omitempty"`
	Professional     []string                          `json:"professional_details,omitempty"`
	Metadata         map[string]string                 `json:"metadata,omitempty"`
}

func (r UploadExecutionDiagnostics) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "upload execution prepared"
	case diagnostics.StatusWarning:
		return "upload execution prepared with warnings"
	case diagnostics.StatusFailed:
		return "upload execution preparation failed"
	default:
		return "upload execution pending"
	}
}

func (r UploadExecutionDiagnostics) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if strings.TrimSpace(r.PackageDir) != "" {
		details = append(details, "package dir: "+r.PackageDir)
	}
	details = append(details, fmt.Sprintf("package exists: %t", r.PackageExists))
	details = append(details, fmt.Sprintf("manifest exists: %t", r.ManifestExists))
	details = append(details, fmt.Sprintf("flash plan exists: %t", r.FlashPlanExists))
	details = append(details, fmt.Sprintf("validation report exists: %t", r.ValidationExists))
	details = append(details, fmt.Sprintf("plan loaded: %t", r.PlanLoaded))
	if strings.TrimSpace(r.PackageMode) != "" {
		details = append(details, "package mode: "+r.PackageMode)
	}
	if strings.TrimSpace(r.TargetChip) != "" {
		details = append(details, "target chip: "+r.TargetChip)
	}
	details = append(details, fmt.Sprintf("stream required: %t", r.StreamRequired))
	details = append(details, fmt.Sprintf("stream available: %t", r.StreamAvailable))
	if strings.TrimSpace(r.StreamStatus) != "" {
		details = append(details, "stream status: "+r.StreamStatus)
	}
	details = append(details, r.Session.ProfessionalDetails()...)
	for _, check := range r.Checks {
		details = append(details, fmt.Sprintf("check %s: %s", check.Name, check.Message))
	}
	for _, warning := range r.Warnings {
		details = append(details, "warning: "+warning)
	}
	for _, limitation := range r.Limitations {
		details = append(details, "limitation: "+limitation)
	}
	return dedupeStrings(details)
}

type UploadExecutionReport struct {
	SchemaVersion string                         `json:"schema_version,omitempty"`
	Status        diagnostics.Status             `json:"status,omitempty"`
	DryRun        bool                           `json:"dry_run,omitempty"`
	PrepareOnly   bool                           `json:"prepare_only,omitempty"`
	Request       UploadExecutionRequest         `json:"request,omitempty"`
	Package       firmware.FirmwarePackage       `json:"package,omitempty"`
	Plan          UploadExecutionPlan            `json:"plan,omitempty"`
	Diagnostics   UploadExecutionDiagnostics     `json:"diagnostics,omitempty"`
	Progress      []UploadExecutionProgressEvent `json:"progress,omitempty"`
	Warnings      []string                       `json:"warnings,omitempty"`
	Limitations   []string                       `json:"limitations,omitempty"`
	NextStep      string                         `json:"next_step,omitempty"`
	Beginner      string                         `json:"beginner_summary,omitempty"`
	Professional  []string                       `json:"professional_details,omitempty"`
}

func (r UploadExecutionReport) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	switch r.Status {
	case diagnostics.StatusPassed:
		return "upload execution prepared"
	case diagnostics.StatusWarning:
		return "upload execution prepared with warnings"
	case diagnostics.StatusFailed:
		return "upload execution preparation failed"
	default:
		return "upload execution pending"
	}
}

func (r UploadExecutionReport) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if strings.TrimSpace(r.Plan.PackageDir) != "" {
		details = append(details, "package dir: "+r.Plan.PackageDir)
	}
	if strings.TrimSpace(r.Plan.PackageMode) != "" {
		details = append(details, "package mode: "+r.Plan.PackageMode)
	}
	if strings.TrimSpace(r.Plan.TargetChip) != "" {
		details = append(details, "target chip: "+r.Plan.TargetChip)
	}
	details = append(details, fmt.Sprintf("dry-run: %t", r.DryRun))
	details = append(details, fmt.Sprintf("prepare-only: %t", r.PrepareOnly))
	details = append(details, fmt.Sprintf("operations: %d", len(r.Plan.Operations)))
	details = append(details, fmt.Sprintf("total bytes: %d", r.Plan.TotalBytes))
	details = append(details, fmt.Sprintf("ordered: %t", r.Plan.Ordered))
	details = append(details, fmt.Sprintf("complete: %t", r.Plan.Complete))
	details = append(details, fmt.Sprintf("stream required: %t", r.Plan.StreamRequired))
	details = append(details, fmt.Sprintf("stream available: %t", r.Plan.StreamAvailable))
	if strings.TrimSpace(r.Plan.StreamStatus) != "" {
		details = append(details, "stream status: "+r.Plan.StreamStatus)
	}
	for _, op := range r.Plan.Operations {
		details = append(details, fmt.Sprintf("0x%x %s %s -> %s (%d bytes, sha256=%s)", op.Offset, op.Name, op.Artifact, op.Path, op.Size, op.SHA256))
	}
	details = append(details, r.Diagnostics.ProfessionalDetails()...)
	for _, warning := range r.Warnings {
		details = append(details, "warning: "+warning)
	}
	for _, limitation := range r.Limitations {
		details = append(details, "limitation: "+limitation)
	}
	if strings.TrimSpace(r.NextStep) != "" {
		details = append(details, "next step: "+r.NextStep)
	}
	return dedupeStrings(details)
}

func (e *Executor) Prepare(ctx context.Context, req UploadExecutionRequest) (UploadExecutionPlan, error) {
	report, err := e.prepare(ctx, req)
	return report.Plan, err
}

func (e *Executor) PrepareOnly(ctx context.Context, req UploadExecutionRequest) (UploadExecutionReport, error) {
	return e.prepare(ctx, req)
}

func (e *Executor) prepare(ctx context.Context, req UploadExecutionRequest) (UploadExecutionReport, error) {
	plan, pkg, diag, err := e.resolvePlan(ctx, req)
	report := UploadExecutionReport{
		SchemaVersion: "1",
		DryRun:        true,
		PrepareOnly:   true,
		Request:       req,
		Package:       pkg,
		Plan:          UploadExecutionPlan{SchemaVersion: "1", Target: req.Target, Metadata: cloneStringMap(req.Metadata)},
		Diagnostics: UploadExecutionDiagnostics{
			SchemaVersion:  "1",
			Metadata:       cloneStringMap(req.Metadata),
			StreamRequired: req.RequireStream,
		},
		Warnings:    []string{},
		Limitations: []string{},
		Progress: []UploadExecutionProgressEvent{
			{
				Time:     clockNowExec(e),
				Phase:    UploadExecutionPhaseInspecting,
				Status:   diagnostics.StatusRunning,
				Progress: 0,
				Message:  "inspecting firmware package",
			},
		},
		NextStep: "open transport and begin byte write",
	}
	if err != nil {
		report.Status = diagnostics.StatusFailed
		report.Diagnostics.Status = diagnostics.StatusFailed
		report.Beginner = firstNonEmpty(diag.Beginner, err.Error(), "upload execution preparation failed")
		report.Diagnostics = mergeExecutionDiagnostics(report.Diagnostics, diag)
		report.Diagnostics.Warnings = append(report.Diagnostics.Warnings, diag.Warnings...)
		report.Diagnostics.Limitations = append(report.Diagnostics.Limitations, diag.Limitations...)
		report.Warnings = append(report.Warnings, diag.Warnings...)
		report.Limitations = append(report.Limitations, diag.Limitations...)
		report.Progress = append(report.Progress, UploadExecutionProgressEvent{
			Time:     clockNowExec(e),
			Phase:    UploadExecutionPhaseLoadingPlan,
			Status:   diagnostics.StatusFailed,
			Progress: 100,
			Message:  err.Error(),
		})
		report.NextStep = "resolve package validation errors and try again"
		report.Professional = report.ProfessionalDetails()
		return report, err
	}

	report.Package = pkg
	report.Plan.SchemaVersion = "1"
	report.Plan.PackageDir = plan.PackageDir
	report.Plan.PackageMode = plan.PackageMode
	report.Plan.TargetChip = plan.TargetChip
	report.Plan.Target = plan.Target
	report.Plan.Metadata = cloneStringMap(plan.Metadata)
	report.Diagnostics = mergeExecutionDiagnostics(report.Diagnostics, diag)
	report.Diagnostics.PlanLoaded = true
	report.Diagnostics.PackageMode = plan.PackageMode
	report.Diagnostics.TargetChip = plan.TargetChip
	report.Diagnostics.PackageExists = true
	report.Diagnostics.ManifestExists = true
	report.Diagnostics.FlashPlanExists = true
	if strings.TrimSpace(pkg.Validation.BeginnerSummary()) != "" && pkg.Validation.HasWarnings() {
		report.Warnings = append(report.Warnings, pkg.Validation.Warnings...)
		report.Diagnostics.Warnings = append(report.Diagnostics.Warnings, pkg.Validation.Warnings...)
		report.Status = diagnostics.StatusWarning
		report.Diagnostics.Status = diagnostics.StatusWarning
	}

	report.Progress = append(report.Progress, UploadExecutionProgressEvent{
		Time:     clockNowExec(e),
		Phase:    UploadExecutionPhaseLoadingPlan,
		Status:   diagnostics.StatusPassed,
		Progress: 10,
		Message:  "upload plan loaded",
	})

	if strings.EqualFold(plan.PackageMode, "app-only") {
		if _, ok := plan.Package.Artifact(firmware.ArtifactBootloaderBinary); !ok {
			report.Warnings = append(report.Warnings, "bootloader artifact is not present in app-only package mode")
			report.Diagnostics.Warnings = append(report.Diagnostics.Warnings, "bootloader artifact is not present in app-only package mode")
		}
	}

	report.Progress = append(report.Progress, UploadExecutionProgressEvent{
		Time:     clockNowExec(e),
		Phase:    UploadExecutionPhaseArtifactValidation,
		Status:   diagnostics.StatusPassed,
		Progress: 30,
		Message:  "artifacts and offsets will be validated",
	})

	operations, opWarnings, opLimitations, opErr := buildExecutionOperations(plan)
	report.Warnings = append(report.Warnings, opWarnings...)
	report.Diagnostics.Warnings = append(report.Diagnostics.Warnings, opWarnings...)
	report.Limitations = append(report.Limitations, opLimitations...)
	report.Diagnostics.Limitations = append(report.Diagnostics.Limitations, opLimitations...)
	if opErr != nil {
		report.Status = diagnostics.StatusFailed
		report.Diagnostics.Status = diagnostics.StatusFailed
		report.Beginner = firstNonEmpty(opErr.Error(), "upload execution preparation failed")
		report.Diagnostics.Checks = append(report.Diagnostics.Checks, UploadExecutionCheck{
			Name:     "artifact-validation",
			Status:   diagnostics.StatusFailed,
			Message:  opErr.Error(),
			Required: true,
		})
		report.Progress = append(report.Progress, UploadExecutionProgressEvent{
			Time:     clockNowExec(e),
			Phase:    UploadExecutionPhaseHashValidation,
			Status:   diagnostics.StatusFailed,
			Progress: 50,
			Message:  opErr.Error(),
		})
		report.NextStep = "resolve artifact validation errors and retry"
		report.Professional = report.ProfessionalDetails()
		return report, opErr
	}

	report.Progress = append(report.Progress, UploadExecutionProgressEvent{
		Time:     clockNowExec(e),
		Phase:    UploadExecutionPhaseHashValidation,
		Status:   diagnostics.StatusPassed,
		Progress: 50,
		Message:  "artifact sizes and hashes validated",
	})

	report.Progress = append(report.Progress, UploadExecutionProgressEvent{
		Time:     clockNowExec(e),
		Phase:    UploadExecutionPhaseOperationPreparation,
		Status:   diagnostics.StatusPassed,
		Progress: 70,
		Message:  "upload operations prepared",
	})

	report.Plan.Operations = operations
	report.Plan.TotalBytes = totalExecutionBytes(operations)
	report.Plan.Ordered = true
	report.Plan.Complete = true
	report.Plan.StreamRequired = req.RequireStream
	report.Plan.StreamAvailable = req.Session != nil
	report.Plan.StreamStatus = "not requested"
	report.Plan.PackageDir = firstNonEmpty(plan.PackageDir, req.PackageDir)
	report.Plan.PackageMode = plan.PackageMode
	report.Plan.TargetChip = plan.TargetChip
	report.Plan.Target = req.Target
	report.Plan.Warnings = append(report.Plan.Warnings, opWarnings...)
	report.Plan.Limitations = append(report.Plan.Limitations, opLimitations...)
	report.Plan.Metadata = cloneStringMap(req.Metadata)

	streamStatus, streamAvailable, sessionDiag, streamErr := executionStreamStatus(req)
	report.Diagnostics.Session = sessionDiag
	report.Diagnostics.StreamAvailable = streamAvailable
	report.Diagnostics.StreamRequired = req.RequireStream
	report.Diagnostics.StreamStatus = streamStatus
	report.Plan.StreamAvailable = streamAvailable
	report.Plan.StreamRequired = req.RequireStream
	report.Plan.StreamStatus = streamStatus

	if streamErr != nil {
		report.Status = diagnostics.StatusFailed
		report.Diagnostics.Status = diagnostics.StatusFailed
		report.Diagnostics.Checks = append(report.Diagnostics.Checks, UploadExecutionCheck{
			Name:     "transport-readiness",
			Status:   diagnostics.StatusFailed,
			Message:  streamErr.Error(),
			Required: req.RequireStream,
		})
		report.Progress = append(report.Progress, UploadExecutionProgressEvent{
			Time:     clockNowExec(e),
			Phase:    UploadExecutionPhaseTransportReadiness,
			Status:   diagnostics.StatusFailed,
			Progress: 85,
			Message:  streamErr.Error(),
		})
		report.NextStep = "resolve transport readiness and retry"
		report.Professional = report.ProfessionalDetails()
		return report, streamErr
	}

	report.Progress = append(report.Progress, UploadExecutionProgressEvent{
		Time:     clockNowExec(e),
		Phase:    UploadExecutionPhaseTransportReadiness,
		Status:   diagnostics.StatusPassed,
		Progress: 85,
		Message:  streamStatus,
	})

	if len(report.Warnings) > 0 || len(report.Diagnostics.Warnings) > 0 {
		report.Status = diagnostics.StatusWarning
		report.Diagnostics.Status = diagnostics.StatusWarning
	} else {
		report.Status = diagnostics.StatusPassed
		report.Diagnostics.Status = diagnostics.StatusPassed
	}

	report.Progress = append(report.Progress, UploadExecutionProgressEvent{
		Time:     clockNowExec(e),
		Phase:    UploadExecutionPhasePrepared,
		Status:   report.Status,
		Progress: 95,
		Message:  report.BeginnerSummary(),
	})
	report.Progress = append(report.Progress, UploadExecutionProgressEvent{
		Time:     clockNowExec(e),
		Phase:    UploadExecutionPhaseByteWriteStopped,
		Status:   report.Status,
		Progress: 100,
		Message:  "prepare-only execution stopped before byte write",
	})

	report.Diagnostics.Checks = append(report.Diagnostics.Checks, UploadExecutionCheck{
		Name:     "operation-preparation",
		Status:   diagnostics.StatusPassed,
		Message:  fmt.Sprintf("%d operations prepared", len(operations)),
		Required: true,
	})
	report.Diagnostics.Checks = append(report.Diagnostics.Checks, UploadExecutionCheck{
		Name:     "transport-readiness",
		Status:   diagnostics.StatusPassed,
		Message:  streamStatus,
		Required: req.RequireStream,
	})
	report.Beginner = report.BeginnerSummary()
	report.Professional = report.ProfessionalDetails()
	if report.Status == diagnostics.StatusFailed {
		return report, errors.New(report.BeginnerSummary())
	}
	return report, nil
}

func (e *Executor) resolvePlan(ctx context.Context, req UploadExecutionRequest) (UploadPlan, firmware.FirmwarePackage, UploadExecutionDiagnostics, error) {
	diag := UploadExecutionDiagnostics{
		SchemaVersion:  "1",
		Status:         diagnostics.StatusPassed,
		PackageDir:     strings.TrimSpace(req.PackageDir),
		Checks:         []UploadExecutionCheck{},
		Warnings:       []string{},
		Limitations:    []string{},
		Metadata:       cloneStringMap(req.Metadata),
		StreamRequired: req.RequireStream,
	}

	if isValidExecutionPlan(req.Plan) {
		plan := req.Plan
		pkg := plan.Package
		diag.PackageExists = true
		diag.ManifestExists = true
		diag.FlashPlanExists = true
		diag.ValidationExists = true
		diag.PlanLoaded = true
		return plan, pkg, diag, nil
	}

	planner := e.Planner
	if planner == nil {
		planner = NewEngine()
	}
	plan, err := planner.Plan(ctx, UploadRequest{
		PackageDir: req.PackageDir,
		Package:    req.Package,
		Target:     req.Target,
		Metadata:   req.Metadata,
	})
	if err != nil {
		diag.Status = diagnostics.StatusFailed
		diag.Checks = append(diag.Checks, UploadExecutionCheck{
			Name:     "package-load",
			Status:   diagnostics.StatusFailed,
			Message:  err.Error(),
			Path:     req.PackageDir,
			Required: true,
		})
		return UploadPlan{}, firmware.FirmwarePackage{}, diag, err
	}
	diag.PackageExists = true
	diag.ManifestExists = true
	diag.FlashPlanExists = true
	diag.ValidationExists = true
	diag.PlanLoaded = true
	return plan, plan.Package, diag, nil
}

func buildExecutionOperations(plan UploadPlan) ([]UploadOperation, []string, []string, error) {
	entries := plan.Package.FlashPlan.SortedEntries()
	ordered := planIsOrdered(plan.Package.FlashPlan)
	warnings := []string{}
	limitations := []string{}
	if !ordered {
		warnings = append(warnings, "flash plan entries were reordered for execution preparation")
	}
	operations := make([]UploadOperation, 0, len(entries))
	for _, entry := range entries {
		artifact, ok := plan.Package.Artifact(entry.Artifact)
		if !ok || strings.TrimSpace(artifact.Path) == "" {
			return nil, warnings, limitations, fmt.Errorf("required artifact %s is missing", entry.Artifact)
		}
		info, err := os.Stat(artifact.Path)
		if err != nil {
			return nil, warnings, limitations, fmt.Errorf("verify %s: %w", artifact.Path, err)
		}
		if info.Size() <= 0 {
			return nil, warnings, limitations, fmt.Errorf("artifact %s has zero size", entry.Artifact)
		}
		if artifact.Size > 0 && info.Size() != artifact.Size {
			return nil, warnings, limitations, fmt.Errorf("artifact %s size mismatch: expected %d, got %d", entry.Artifact, artifact.Size, info.Size())
		}
		sum, err := sha256Hex(artifact.Path)
		if err != nil {
			return nil, warnings, limitations, err
		}
		if strings.TrimSpace(artifact.SHA256) != "" && !strings.EqualFold(sum, artifact.SHA256) {
			return nil, warnings, limitations, fmt.Errorf("artifact %s hash mismatch: expected %s, got %s", entry.Artifact, artifact.SHA256, sum)
		}
		op := UploadOperation{
			Name:        artifactName(entry.Artifact),
			Kind:        UploadOperationArtifact,
			Artifact:    entry.Artifact,
			Offset:      entry.Offset,
			Path:        artifact.Path,
			Size:        info.Size(),
			SHA256:      sum,
			Required:    entry.Required,
			Ordered:     ordered,
			Description: stepDescription(entry),
		}
		operations = append(operations, op)
	}
	if plan.PackageMode == "app-only" {
		limitations = append(limitations, "prepare-only mode stops before transport execution")
	}
	return operations, warnings, limitations, nil
}

func executionStreamStatus(req UploadExecutionRequest) (string, bool, UploadExecutionSessionDiagnostics, error) {
	if req.Session == nil {
		if req.RequireStream {
			return "required stream capability missing", false, UploadExecutionSessionDiagnostics{SchemaVersion: "1", Status: diagnostics.StatusFailed}, errors.New("transport stream capability is required but no session was provided")
		}
		return "stream capability not requested", false, UploadExecutionSessionDiagnostics{SchemaVersion: "1", Status: diagnostics.StatusPassed, StreamSupported: false, StreamAvailable: false}, nil
	}
	diag := req.Session.Diagnostics()
	if !diag.StreamSupported || !diag.StreamAvailable {
		if req.RequireStream {
			return "transport stream capability unavailable", false, diag, errors.New("transport stream capability is required but unavailable")
		}
		return "transport stream capability unavailable", false, diag, nil
	}
	return "transport stream capability ready", true, diag, nil
}

func mergeExecutionDiagnostics(base, overlay UploadExecutionDiagnostics) UploadExecutionDiagnostics {
	if len(overlay.Checks) > 0 {
		base.Checks = append(base.Checks, overlay.Checks...)
	}
	if len(overlay.Warnings) > 0 {
		base.Warnings = append(base.Warnings, overlay.Warnings...)
	}
	if len(overlay.Limitations) > 0 {
		base.Limitations = append(base.Limitations, overlay.Limitations...)
	}
	if overlay.Status != "" {
		base.Status = overlay.Status
	}
	if strings.TrimSpace(overlay.PackageDir) != "" {
		base.PackageDir = overlay.PackageDir
	}
	if overlay.PackageExists {
		base.PackageExists = true
	}
	if overlay.ManifestExists {
		base.ManifestExists = true
	}
	if overlay.FlashPlanExists {
		base.FlashPlanExists = true
	}
	if overlay.ValidationExists {
		base.ValidationExists = true
	}
	if overlay.PlanLoaded {
		base.PlanLoaded = true
	}
	if strings.TrimSpace(overlay.PackageMode) != "" {
		base.PackageMode = overlay.PackageMode
	}
	if strings.TrimSpace(overlay.TargetChip) != "" {
		base.TargetChip = overlay.TargetChip
	}
	if overlay.StreamRequired {
		base.StreamRequired = true
	}
	if overlay.StreamAvailable {
		base.StreamAvailable = true
	}
	if strings.TrimSpace(overlay.StreamStatus) != "" {
		base.StreamStatus = overlay.StreamStatus
	}
	if base.Metadata == nil {
		base.Metadata = map[string]string{}
	}
	for k, v := range overlay.Metadata {
		base.Metadata[k] = v
	}
	return base
}

func totalExecutionBytes(operations []UploadOperation) int64 {
	var total int64
	for _, op := range operations {
		total += op.Size
	}
	return total
}

func isValidExecutionPlan(plan UploadPlan) bool {
	return plan.Package.Validate() == nil && len(plan.Steps) > 0 && len(plan.Package.Manifest.Artifacts) > 0
}

func sha256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func clockNowExec(e *Executor) time.Time {
	if e != nil && e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}
