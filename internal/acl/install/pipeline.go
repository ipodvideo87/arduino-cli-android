package install

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/arduino/arduino-cli/internal/acl/compatibility"
	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
)

type StageName string

const (
	StageDownload               StageName = "download"
	StageExtract                StageName = "extract"
	StageAndroidPatch           StageName = "android-patch"
	StagePermissionRuntimeFixes StageName = "permission-runtime-fixes"
	StageExecutableValidation   StageName = "executable-validation"
	StageRegister               StageName = "register"
	StageSelfTest               StageName = "self-test"
	StageReady                  StageName = "ready"
)

type PatchFix struct {
	Path     string             `json:"path,omitempty"`
	Action   string             `json:"action,omitempty"`
	Reason   string             `json:"reason,omitempty"`
	Status   diagnostics.Status `json:"status"`
	Message  string             `json:"message,omitempty"`
	Evidence []string           `json:"evidence,omitempty"`
}

type PatchStage struct {
	Name       StageName          `json:"name"`
	Status     diagnostics.Status `json:"status"`
	Message    string             `json:"message,omitempty"`
	StartedAt  time.Time          `json:"started_at,omitempty"`
	FinishedAt time.Time          `json:"finished_at,omitempty"`
	Evidence   []string           `json:"evidence,omitempty"`
}

type PatchManifest struct {
	PackageName    string                           `json:"package_name,omitempty"`
	PackageVersion string                           `json:"package_version,omitempty"`
	Source         string                           `json:"source,omitempty"`
	Status         diagnostics.Status               `json:"status"`
	Stages         []PatchStage                     `json:"stages,omitempty"`
	Fixes          []PatchFix                       `json:"fixes,omitempty"`
	Compatibility  compatibility.InstallationReport `json:"compatibility,omitempty"`
	Metadata       map[string]string                `json:"metadata,omitempty"`
	UpdatedAt      time.Time                        `json:"updated_at,omitempty"`
}

type StageRequest struct {
	Stage    StageName
	Manifest PatchManifest
}

type StageResult struct {
	Status   diagnostics.Status
	Message  string
	Evidence []string
	Fixes    []PatchFix
}

type StageExecutor interface {
	Execute(context.Context, StageRequest) (StageResult, error)
}

type AndroidInstallPatchPipeline struct {
	Stages   []StageName
	Executor StageExecutor
}

func DefaultStages() []StageName {
	return []StageName{
		StageDownload,
		StageExtract,
		StageAndroidPatch,
		StagePermissionRuntimeFixes,
		StageExecutableValidation,
		StageRegister,
		StageSelfTest,
		StageReady,
	}
}

func NewAndroidInstallPatchPipeline(executor StageExecutor) *AndroidInstallPatchPipeline {
	return &AndroidInstallPatchPipeline{
		Stages:   DefaultStages(),
		Executor: executor,
	}
}

func (p *AndroidInstallPatchPipeline) Run(ctx context.Context, manifest *PatchManifest) error {
	if p == nil {
		return errors.New("pipeline is nil")
	}
	if manifest == nil {
		return errors.New("patch manifest is required")
	}
	if p.Executor == nil {
		return errors.New("stage executor is required")
	}

	for _, stage := range p.stageList() {
		manifest.setStage(stage, diagnostics.StatusRunning, "running")
		result, err := p.Executor.Execute(ctx, StageRequest{
			Stage:    stage,
			Manifest: manifest.Clone(),
		})
		if err != nil {
			manifest.setStage(stage, diagnostics.StatusFailed, err.Error())
			manifest.Status = diagnostics.StatusFailed
			manifest.UpdatedAt = time.Now().UTC()
			return err
		}
		status := result.Status
		if status == "" {
			status = diagnostics.StatusPassed
		}
		manifest.setStage(stage, status, result.Message, result.Evidence...)
		manifest.Fixes = append(manifest.Fixes, result.Fixes...)
	}

	manifest.Status = manifest.FinalStatus()
	manifest.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *PatchManifest) setStage(name StageName, status diagnostics.Status, message string, evidence ...string) {
	if m == nil {
		return
	}
	now := time.Now().UTC()
	idx := -1
	for i, stage := range m.Stages {
		if stage.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		m.Stages = append(m.Stages, PatchStage{Name: name})
		idx = len(m.Stages) - 1
	}
	stage := &m.Stages[idx]
	stage.Name = name
	stage.Status = status
	stage.Message = message
	if status == diagnostics.StatusRunning && stage.StartedAt.IsZero() {
		stage.StartedAt = now
	}
	if status.IsTerminal() {
		if stage.StartedAt.IsZero() {
			stage.StartedAt = now
		}
		stage.FinishedAt = now
	}
	stage.Evidence = append([]string(nil), evidence...)
	m.Status = m.FinalStatus()
	m.UpdatedAt = now
}

func (m PatchManifest) Stage(name StageName) (PatchStage, bool) {
	for _, stage := range m.Stages {
		if stage.Name == name {
			return stage, true
		}
	}
	return PatchStage{}, false
}

func (m PatchManifest) FinalStatus() diagnostics.Status {
	if len(m.Stages) == 0 {
		return diagnostics.StatusPending
	}
	status := diagnostics.StatusPassed
	for _, stage := range m.Stages {
		switch stage.Status {
		case diagnostics.StatusFailed:
			return diagnostics.StatusFailed
		case diagnostics.StatusWarning:
			status = diagnostics.StatusWarning
		case diagnostics.StatusRunning:
			if status == diagnostics.StatusPending {
				status = diagnostics.StatusRunning
			}
		case diagnostics.StatusPending:
			if status == diagnostics.StatusPassed {
				status = diagnostics.StatusPending
			}
		}
	}
	return status
}

func (m PatchManifest) StatusCounts() map[diagnostics.Status]int {
	counts := map[diagnostics.Status]int{}
	for _, stage := range m.Stages {
		if stage.Status == "" {
			counts[diagnostics.StatusPending]++
			continue
		}
		counts[stage.Status]++
	}
	return counts
}

func (m PatchManifest) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "patch manifest: %s\n", m.PackageName)
	fmt.Fprintf(&b, "status: %s\n", m.Status)
	for _, stage := range m.Stages {
		fmt.Fprintf(&b, "- %s: %s", stage.Name, stage.Status)
		if stage.Message != "" {
			fmt.Fprintf(&b, " (%s)", stage.Message)
		}
		if len(stage.Evidence) > 0 {
			fmt.Fprintf(&b, " [%s]", strings.Join(stage.Evidence, ", "))
		}
		fmt.Fprintln(&b)
	}
	for _, fix := range m.Fixes {
		fmt.Fprintf(&b, "fix: %s %s %s\n", fix.Path, fix.Action, fix.Reason)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m PatchManifest) Clone() PatchManifest {
	clone := m
	clone.Stages = append([]PatchStage(nil), m.Stages...)
	clone.Fixes = append([]PatchFix(nil), m.Fixes...)
	clone.Compatibility = m.Compatibility
	clone.Compatibility.Report.Decisions = append([]compatibility.Decision(nil), m.Compatibility.Report.Decisions...)
	clone.Compatibility.Report.Warnings = append([]string(nil), m.Compatibility.Report.Warnings...)
	clone.Compatibility.Report.Errors = append([]string(nil), m.Compatibility.Report.Errors...)
	clone.Compatibility.Report.Notes = append([]string(nil), m.Compatibility.Report.Notes...)
	clone.Compatibility.Compatibility = append([]compatibility.CompatibilityReport(nil), m.Compatibility.Compatibility...)
	for i := range clone.Compatibility.Compatibility {
		clone.Compatibility.Compatibility[i].Decisions = append([]compatibility.Decision(nil), m.Compatibility.Compatibility[i].Decisions...)
		clone.Compatibility.Compatibility[i].Beginner = append([]string(nil), m.Compatibility.Compatibility[i].Beginner...)
		clone.Compatibility.Compatibility[i].Professional = append([]string(nil), m.Compatibility.Compatibility[i].Professional...)
	}
	clone.Metadata = cloneStringMap(m.Metadata)
	for i := range clone.Stages {
		clone.Stages[i].Evidence = append([]string(nil), clone.Stages[i].Evidence...)
	}
	for i := range clone.Fixes {
		clone.Fixes[i].Evidence = append([]string(nil), clone.Fixes[i].Evidence...)
	}
	return clone
}

func (m *PatchManifest) SetCompatibility(report compatibility.InstallationReport) {
	if m == nil {
		return
	}
	m.Compatibility = report
	m.UpdatedAt = time.Now().UTC()
}

func (p *AndroidInstallPatchPipeline) stageList() []StageName {
	if len(p.Stages) > 0 {
		return append([]StageName(nil), p.Stages...)
	}
	return DefaultStages()
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
