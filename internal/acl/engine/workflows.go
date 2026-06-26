package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/compatibility"
	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/firmware"
	aclinstall "github.com/arduino/arduino-cli/internal/acl/install"
	aclpatcher "github.com/arduino/arduino-cli/internal/acl/patcher"
	aclscanner "github.com/arduino/arduino-cli/internal/acl/scanner"
	aclverifier "github.com/arduino/arduino-cli/internal/acl/verifier"
)

type BootstrapWorkflowReport struct {
	Root           string                      `json:"root,omitempty"`
	RuntimeRoot    string                      `json:"runtime_root,omitempty"`
	Scan           aclscanner.Report           `json:"scan,omitempty"`
	ScanValidation aclscanner.ValidationReport `json:"scan_validation,omitempty"`
	Verifier       aclverifier.Report          `json:"verifier,omitempty"`
	PatchPreview   aclpatcher.Report           `json:"patch_preview,omitempty"`
	Beginner       string                      `json:"beginner_summary,omitempty"`
	Professional   []string                    `json:"professional_details,omitempty"`
}

type FlashWorkflowReport struct {
	PackageValidation any      `json:"package_validation,omitempty"`
	Transport         any      `json:"transport,omitempty"`
	Bridge            any      `json:"bridge,omitempty"`
	Flash             any      `json:"flash,omitempty"`
	Verify            any      `json:"verify,omitempty"`
	Beginner          string   `json:"beginner_summary,omitempty"`
	Professional      []string `json:"professional_details,omitempty"`
}

type DiagnosticsWorkflowReport struct {
	Scanner      aclscanner.Report           `json:"scanner,omitempty"`
	Validation   aclscanner.ValidationReport `json:"validation,omitempty"`
	Verifier     aclverifier.Report          `json:"verifier,omitempty"`
	Environment  map[string]string           `json:"environment,omitempty"`
	Toolchain    map[string]string           `json:"toolchain,omitempty"`
	Beginner     string                      `json:"beginner_summary,omitempty"`
	Professional []string                    `json:"professional_details,omitempty"`
}

func BootstrapWorkflow() Workflow {
	return Workflow{
		Name: "bootstrap",
		Jobs: []Job{
			{
				Name: "bootstrap",
				Steps: []Step{
					{Name: "scan", Critical: true, Execute: runBootstrapScan},
					{Name: "verify", Critical: true, Execute: runBootstrapVerify},
					{Name: "patch-preview", Execute: runBootstrapPatchPreview},
					{Name: "bootstrap report", Critical: true, Execute: runBootstrapReport},
				},
			},
		},
	}
}

func CompileWorkflow() Workflow {
	return Workflow{
		Name: "compile",
		Jobs: []Job{
			{
				Name: "compile",
				Steps: []Step{
					{Name: "preflight", Critical: true, Execute: runCompilePreflight},
					{Name: "compatibility check", Execute: runCompileCompatibility},
					{Name: "compile", Critical: true, Execute: runCompileHook},
					{Name: "firmware package generation", Critical: true, Execute: runFirmwarePackageHook},
					{Name: "flash plan generation", Execute: runFlashPlanHook},
					{Name: "binary validation", Critical: true, Execute: runBinaryValidationHook},
					{Name: "workflow report", Critical: true, Execute: runCompileReport},
				},
			},
		},
	}
}

func FlashWorkflow() Workflow {
	return Workflow{
		Name: "flash",
		Jobs: []Job{
			{
				Name: "flash",
				Steps: []Step{
					{Name: "package validation", Critical: true, Execute: runFlashPackageValidation},
					{Name: "transport selection", Execute: runTransportSelection},
					{Name: "usb bridge placeholder", Optional: true, Execute: runUSBBridgePlaceholder},
					{Name: "flash placeholder", Optional: true, Execute: runFlashPlaceholder},
					{Name: "verify placeholder", Optional: true, Execute: runFlashVerifyPlaceholder},
				},
			},
		},
	}
}

func DiagnosticsWorkflow() Workflow {
	return Workflow{
		Name: "diagnostics",
		Jobs: []Job{
			{
				Name: "diagnostics",
				Steps: []Step{
					{Name: "scanner", Execute: runDiagnosticsScanner},
					{Name: "verifier", Execute: runDiagnosticsVerifier},
					{Name: "android environment checks", Execute: runDiagnosticsEnvironment},
					{Name: "toolchain checks", Execute: runDiagnosticsToolchain},
					{Name: "diagnostics report", Critical: true, Execute: runDiagnosticsReport},
				},
			},
		},
	}
}

func runBootstrapScan(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	if strings.TrimSpace(wctx.Root) == "" {
		return StepResult{Status: StepStatusFailed, Message: "scan root is required", Beginner: "scan root is required", Critical: true}, fmt.Errorf("scan root is required")
	}
	report, err := aclscanner.New().Scan(wctx.Root)
	if err != nil {
		return StepResult{Status: StepStatusFailed, Message: err.Error(), Beginner: err.Error(), Critical: true}, err
	}
	validation := aclscanner.Validate(report)
	wctx.Set("scan", report)
	wctx.Set("scan_validation", validation)
	return StepResult{
		Status:       StepStatusPassed,
		Message:      validation.BeginnerSummary(),
		Beginner:     validation.BeginnerSummary(),
		Professional: append(report.ProfessionalDetails(), validation.FindingsDetails()...),
		Data: struct {
			Scan       aclscanner.Report
			Validation aclscanner.ValidationReport
		}{Scan: report, Validation: validation},
	}, nil
}

func runBootstrapVerify(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	report, err := aclverifier.New(wctx.RuntimeRoot).Verify(aclverifier.Request{
		Root:        wctx.Root,
		RuntimeRoot: wctx.RuntimeRoot,
		TargetPath:  wctx.TargetPath,
	})
	if err != nil {
		return StepResult{Status: StepStatusFailed, Message: err.Error(), Beginner: err.Error(), Critical: true}, err
	}
	wctx.Set("verifier", report)
	return StepResult{
		Status:       report.Status,
		Message:      report.Beginner,
		Beginner:     report.Beginner,
		Professional: append([]string(nil), report.Professional...),
		Data:         report,
	}, nil
}

func runBootstrapPatchPreview(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	if strings.TrimSpace(wctx.Root) == "" {
		return StepResult{Status: StepStatusSkipped, Message: "scan root is not set", Beginner: "patch preview skipped", Skipped: true}, nil
	}
	report, err := aclpatcher.DryRun(wctx.Root)
	if err != nil {
		return StepResult{Status: StepStatusFailed, Message: err.Error(), Beginner: err.Error()}, err
	}
	wctx.Set("patch_preview", report)
	return StepResult{
		Status:       StepStatusPassed,
		Message:      report.BeginnerSummary(),
		Beginner:     report.BeginnerSummary(),
		Professional: report.ProfessionalDetails(),
		Data:         report,
	}, nil
}

func runBootstrapReport(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	scan, _ := wctx.Get("scan")
	validation, _ := wctx.Get("scan_validation")
	verifier, _ := wctx.Get("verifier")
	preview, _ := wctx.Get("patch_preview")

	report := BootstrapWorkflowReport{
		Root:        wctx.Root,
		RuntimeRoot: wctx.RuntimeRoot,
	}
	if v, ok := scan.(aclscanner.Report); ok {
		report.Scan = v
	}
	if v, ok := validation.(aclscanner.ValidationReport); ok {
		report.ScanValidation = v
	}
	if v, ok := verifier.(aclverifier.Report); ok {
		report.Verifier = v
	}
	if v, ok := preview.(aclpatcher.Report); ok {
		report.PatchPreview = v
	}

	manifest := aclinstall.PatchManifest{
		PackageName:    "acl-bootstrap",
		PackageVersion: "1",
		Source:         "engine",
		Metadata: map[string]string{
			"root":        wctx.Root,
			"runtimeRoot": wctx.RuntimeRoot,
		},
	}
	manifest.Stages = append(manifest.Stages, aclinstall.PatchStage{
		Name:    aclinstall.StageExecutableValidation,
		Status:  statusFromValidation(report.ScanValidation),
		Message: report.ScanValidation.BeginnerSummary(),
	})
	manifest.Stages = append(manifest.Stages, aclinstall.PatchStage{
		Name:    aclinstall.StageSelfTest,
		Status:  statusFromVerifier(report.Verifier),
		Message: report.Verifier.Beginner,
	})
	report.Beginner = bootstrapBeginner(report)
	report.Professional = bootstrapProfessional(report)
	wctx.Set("bootstrap_report", report)
	return StepResult{
		Status:       reportStatus(report),
		Message:      report.Beginner,
		Beginner:     report.Beginner,
		Professional: report.Professional,
		Data: struct {
			Report   BootstrapWorkflowReport
			Manifest aclinstall.PatchManifest
		}{Report: report, Manifest: manifest},
	}, nil
}

func runCompilePreflight(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	if err := validateCompileRequest(wctx.CompileRequest); err != nil {
		return StepResult{Status: StepStatusFailed, Message: err.Error(), Beginner: err.Error(), Critical: true}, err
	}
	if wctx.CompileRunner == nil {
		return StepResult{Status: StepStatusFailed, Message: "compile runner is not configured", Beginner: "compile runner is not configured", Critical: true}, fmt.Errorf("compile runner is not configured")
	}
	message := fmt.Sprintf("compile preflight ready for %s", filepath.Base(wctx.CompileRequest.SketchPath))
	return StepResult{Status: StepStatusPassed, Message: message, Beginner: message}, nil
}

func runCompileCompatibility(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	report, ok := wctx.Get("compatibility_report")
	if !ok {
		message := "compatibility check completed"
		return StepResult{Status: StepStatusPassed, Message: message, Beginner: message}, nil
	}
	switch value := report.(type) {
	case compatibility.Report:
		wctx.Set("compatibility_report", value)
		return StepResult{Status: value.Status, Message: compatibilityBeginner(value), Beginner: compatibilityBeginner(value), Professional: compatibilityProfessional(value), Data: value}, nil
	case compatibility.InstallationReport:
		wctx.Set("compatibility_report", value)
		return StepResult{Status: value.Report.Status, Message: compatibilityInstallationBeginner(value), Beginner: compatibilityInstallationBeginner(value), Professional: compatibilityInstallationProfessional(value), Data: value}, nil
	default:
		message := "compatibility report attached"
		return StepResult{Status: StepStatusWarning, Message: message, Beginner: message, Professional: []string{fmt.Sprintf("compatibility report type: %T", report)}, Data: report}, nil
	}
}

func runCompileHook(ctx context.Context, wctx *WorkflowContext) (StepResult, error) {
	result, err := wctx.CompileRunner.Run(ctx, wctx.CompileRequest, func(event Event) {
		event.Workflow = "compile"
		event.Job = "compile"
		if event.Step == "" {
			event.Step = "compile"
		}
		_ = wctx.Publish(event)
	})
	if err != nil {
		return StepResult{Status: StepStatusFailed, Message: err.Error(), Beginner: err.Error(), Critical: true}, err
	}
	wctx.Set("compile_execution", result)
	message := "compile completed"
	if strings.TrimSpace(result.BuildPath) != "" {
		message = "compile completed at " + result.BuildPath
	}
	return StepResult{Status: StepStatusPassed, Message: message, Beginner: message, Data: result}, nil
}

func runFirmwarePackageHook(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	execution, ok := wctx.Get("compile_execution")
	if !ok {
		return StepResult{Status: StepStatusFailed, Message: "compile execution not available", Beginner: "compile execution not available", Critical: true}, fmt.Errorf("compile execution not available")
	}
	exec, ok := execution.(CompileExecution)
	if !ok {
		return StepResult{Status: StepStatusFailed, Message: "compile execution has an unsupported type", Beginner: "compile execution is invalid", Critical: true}, fmt.Errorf("compile execution has an unsupported type")
	}
	if strings.TrimSpace(exec.PackageDir) == "" {
		exec.PackageDir = compileWorkflowPackageDir(wctx.CompileRequest)
	}
	if strings.TrimSpace(exec.PackageDir) == "" {
		return StepResult{Status: StepStatusFailed, Message: "package directory is not available", Beginner: "package directory is not available", Critical: true}, fmt.Errorf("package directory is not available")
	}
	pkg, err := firmware.LoadFirmwarePackage(exec.PackageDir)
	if err != nil {
		return StepResult{Status: StepStatusFailed, Message: err.Error(), Beginner: err.Error(), Critical: true}, err
	}
	wctx.Set("firmware_package", pkg)
	wctx.Set("compile_execution", exec)
	message := "firmware package generated at " + exec.PackageDir
	professional := []string{"package path: " + exec.PackageDir}
	if pkg.Manifest.MemoryUsage.ProgramTotalBytes > 0 || pkg.Manifest.MemoryUsage.RAMTotalBytes > 0 {
		professional = append(professional, fmt.Sprintf("memory usage: flash %d/%d (%d%%), ram %d/%d (%d%%)",
			pkg.Manifest.MemoryUsage.ProgramUsedBytes, pkg.Manifest.MemoryUsage.ProgramTotalBytes, pkg.Manifest.MemoryUsage.ProgramPercent,
			pkg.Manifest.MemoryUsage.RAMUsedBytes, pkg.Manifest.MemoryUsage.RAMTotalBytes, pkg.Manifest.MemoryUsage.RAMPercent,
		))
	}
	return StepResult{Status: StepStatusPassed, Message: message, Beginner: "firmware package generated", Professional: professional, Data: pkg}, nil
}

func runFlashPlanHook(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	value, ok := wctx.Get("firmware_package")
	if !ok {
		return StepResult{Status: StepStatusFailed, Message: "firmware package is not available", Beginner: "firmware package is not available", Critical: true}, fmt.Errorf("firmware package is not available")
	}
	pkg, ok := value.(firmware.FirmwarePackage)
	if !ok {
		return StepResult{Status: StepStatusFailed, Message: "firmware package has an unsupported type", Beginner: "firmware package is invalid", Critical: true}, fmt.Errorf("firmware package has an unsupported type")
	}
	plan := pkg.FlashPlan
	wctx.Set("flash_plan", plan)
	message := fmt.Sprintf("flash plan generated with %d entries", len(plan.Entries))
	return StepResult{Status: StepStatusPassed, Message: message, Beginner: message, Professional: flashPlanProfessional(plan), Data: plan}, nil
}

func runBinaryValidationHook(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	value, ok := wctx.Get("firmware_package")
	if !ok {
		return StepResult{Status: StepStatusFailed, Message: "firmware package is not available", Beginner: "firmware package is not available", Critical: true}, fmt.Errorf("firmware package is not available")
	}
	pkg, ok := value.(firmware.FirmwarePackage)
	if !ok {
		return StepResult{Status: StepStatusFailed, Message: "firmware package has an unsupported type", Beginner: "firmware package is invalid", Critical: true}, fmt.Errorf("firmware package has an unsupported type")
	}
	validator := firmware.NewBinaryValidator()
	report := validator.Validate(pkg)
	wctx.Set("binary_validation", report)
	if report.HasFailures() {
		return StepResult{Status: StepStatusFailed, Message: report.BeginnerSummary(), Beginner: "package not ready to flash", Professional: validationProfessional(report), Data: report}, nil
	}
	return StepResult{Status: StepStatusPassed, Message: report.BeginnerSummary(), Beginner: report.BeginnerSummary(), Professional: validationProfessional(report), Data: report}, nil
}

func runCompileReport(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	exec, _ := wctx.Get("compile_execution")
	pkg, _ := wctx.Get("firmware_package")
	validation, _ := wctx.Get("binary_validation")
	compatibilityReport, _ := wctx.Get("compatibility_report")
	report := CompileWorkflowReport{}
	report.Request = wctx.CompileRequest
	if v, ok := exec.(CompileExecution); ok {
		report.Execution = v
	}
	if v, ok := pkg.(firmware.FirmwarePackage); ok {
		report.Package = v
		report.PackagePath = report.PackageLocation()
	}
	if v, ok := validation.(firmware.ValidationReport); ok {
		report.Validation = v
	}
	if v, ok := compatibilityReport.(compatibility.Report); ok {
		report.Compatibility = v
	}
	report.PackageReady = report.Package.Validate() == nil
	report.ReadyToFlash = report.PackageReady && !report.Validation.HasFailures()
	report.Beginner = compileWorkflowBeginner(
		preflightSummary(report.Request),
		compatibilityReportSummary(report.Compatibility),
		report.Execution,
		report.Package,
		report.Validation,
	)
	report.Professional = compileWorkflowProfessional(report)
	wctx.Set("compile_report", report)
	return StepResult{Status: compileWorkflowStatus(report), Message: report.Beginner, Beginner: report.Beginner, Professional: report.Professional, Data: report}, nil
}

func runFlashPackageValidation(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	hook, ok := wctx.Get("flash_package_validation")
	if !ok {
		return StepResult{Status: StepStatusSkipped, Message: "package validation hook not provided", Beginner: "package validation skipped", Skipped: true}, nil
	}
	fn, ok := hook.(func(context.Context, *WorkflowContext) (StepResult, error))
	if !ok {
		return StepResult{Status: StepStatusFailed, Message: "package validation hook has an unsupported type", Beginner: "package validation hook is invalid", Critical: true}, fmt.Errorf("package validation hook has an unsupported type")
	}
	return fn(context.Background(), wctx)
}

func runTransportSelection(ctx context.Context, wctx *WorkflowContext) (StepResult, error) {
	hook, ok := wctx.Get("transport_selection")
	if !ok {
		return StepResult{Status: StepStatusSkipped, Message: "transport selector not provided", Beginner: "transport selection skipped", Skipped: true}, nil
	}
	fn, ok := hook.(func(context.Context, *WorkflowContext) (StepResult, error))
	if !ok {
		return StepResult{Status: StepStatusFailed, Message: "transport selector has an unsupported type", Beginner: "transport selector is invalid", Critical: true}, fmt.Errorf("transport selector has an unsupported type")
	}
	return fn(ctx, wctx)
}

func runUSBBridgePlaceholder(_ context.Context, _ *WorkflowContext) (StepResult, error) {
	return StepResult{Status: StepStatusSkipped, Message: "USB bridge not implemented yet", Beginner: "USB bridge placeholder", Skipped: true}, nil
}

func runFlashPlaceholder(_ context.Context, _ *WorkflowContext) (StepResult, error) {
	return StepResult{Status: StepStatusSkipped, Message: "flashing not implemented yet", Beginner: "flash placeholder", Skipped: true}, nil
}

func runFlashVerifyPlaceholder(_ context.Context, _ *WorkflowContext) (StepResult, error) {
	return StepResult{Status: StepStatusSkipped, Message: "verification not implemented yet", Beginner: "verify placeholder", Skipped: true}, nil
}

func runDiagnosticsScanner(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	if strings.TrimSpace(wctx.Root) == "" {
		return StepResult{Status: StepStatusSkipped, Message: "scan root not provided", Beginner: "scanner skipped", Skipped: true}, nil
	}
	report, err := aclscanner.New().Scan(wctx.Root)
	if err != nil {
		return StepResult{Status: StepStatusFailed, Message: err.Error(), Beginner: err.Error()}, err
	}
	validation := aclscanner.Validate(report)
	wctx.Set("diagnostics_scan", report)
	wctx.Set("diagnostics_validation", validation)
	return StepResult{Status: StepStatusPassed, Message: validation.BeginnerSummary(), Beginner: validation.BeginnerSummary(), Professional: append(report.ProfessionalDetails(), validation.FindingsDetails()...), Data: report}, nil
}

func runDiagnosticsVerifier(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	report, err := aclverifier.New(wctx.RuntimeRoot).Verify(aclverifier.Request{
		Root:        wctx.Root,
		RuntimeRoot: wctx.RuntimeRoot,
		TargetPath:  wctx.TargetPath,
	})
	if err != nil {
		return StepResult{Status: StepStatusFailed, Message: err.Error(), Beginner: err.Error()}, err
	}
	wctx.Set("diagnostics_verifier", report)
	return StepResult{Status: report.Status, Message: report.Beginner, Beginner: report.Beginner, Professional: append([]string(nil), report.Professional...), Data: report}, nil
}

func runDiagnosticsEnvironment(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	env := map[string]string{}
	for _, key := range []string{"ANDROID_ROOT", "ANDROID_DATA", "TERMUX_VERSION", "PREFIX", "ACL_RUNTIME_ROOT"} {
		if value := strings.TrimSpace(envLookup(key)); value != "" {
			env[key] = value
		}
	}
	wctx.Set("diagnostics_environment", env)
	return StepResult{Status: StepStatusPassed, Message: "android environment collected", Beginner: "android environment collected", Data: env}, nil
}

func runDiagnosticsToolchain(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	toolchain := map[string]string{}
	for _, key := range []string{"compiler.path", "runtime.tools.gcc.path", "runtime.tools.xtensa-esp32-elf-gcc.path"} {
		if value, ok := wctx.Get(key); ok {
			toolchain[key] = fmt.Sprint(value)
		}
	}
	wctx.Set("diagnostics_toolchain", toolchain)
	return StepResult{Status: StepStatusPassed, Message: "toolchain metadata collected", Beginner: "toolchain metadata collected", Data: toolchain}, nil
}

func runDiagnosticsReport(_ context.Context, wctx *WorkflowContext) (StepResult, error) {
	report := DiagnosticsWorkflowReport{}
	if value, ok := wctx.Get("diagnostics_scan"); ok {
		if v, ok := value.(aclscanner.Report); ok {
			report.Scanner = v
		}
	}
	if value, ok := wctx.Get("diagnostics_validation"); ok {
		if v, ok := value.(aclscanner.ValidationReport); ok {
			report.Validation = v
		}
	}
	if value, ok := wctx.Get("diagnostics_verifier"); ok {
		if v, ok := value.(aclverifier.Report); ok {
			report.Verifier = v
		}
	}
	if value, ok := wctx.Get("diagnostics_environment"); ok {
		if v, ok := value.(map[string]string); ok {
			report.Environment = v
		}
	}
	if value, ok := wctx.Get("diagnostics_toolchain"); ok {
		if v, ok := value.(map[string]string); ok {
			report.Toolchain = v
		}
	}
	report.Beginner = diagnosticsBeginner(report)
	report.Professional = diagnosticsProfessional(report)
	wctx.Set("diagnostics_report", report)
	return StepResult{Status: reportStatusFromDiagnostics(report), Message: report.Beginner, Beginner: report.Beginner, Professional: report.Professional, Data: report}, nil
}

func bootstrapBeginner(report BootstrapWorkflowReport) string {
	parts := []string{}
	if strings.TrimSpace(report.ScanValidation.BeginnerSummary()) != "" {
		parts = append(parts, report.ScanValidation.BeginnerSummary())
	}
	if strings.TrimSpace(report.Verifier.Beginner) != "" {
		parts = append(parts, report.Verifier.Beginner)
	}
	if strings.TrimSpace(report.PatchPreview.BeginnerSummary()) != "" {
		parts = append(parts, report.PatchPreview.BeginnerSummary())
	}
	if len(parts) == 0 {
		return "bootstrap completed"
	}
	return strings.Join(parts, "; ")
}

func bootstrapProfessional(report BootstrapWorkflowReport) []string {
	details := append([]string(nil), report.Scan.ProfessionalDetails()...)
	details = append(details, report.ScanValidation.FindingsDetails()...)
	details = append(details, report.Verifier.Professional...)
	details = append(details, report.PatchPreview.ProfessionalDetails()...)
	return details
}

func diagnosticsBeginner(report DiagnosticsWorkflowReport) string {
	parts := []string{}
	if report.Scanner.Summary.TotalEntries > 0 {
		parts = append(parts, report.Scanner.BeginnerSummary())
	}
	if report.Validation.Summary.TotalFilesScanned > 0 {
		parts = append(parts, report.Validation.BeginnerSummary())
	}
	if report.Verifier.Beginner != "" {
		parts = append(parts, report.Verifier.Beginner)
	}
	if len(parts) == 0 {
		return "diagnostics completed"
	}
	return strings.Join(parts, "; ")
}

func diagnosticsProfessional(report DiagnosticsWorkflowReport) []string {
	details := append([]string(nil), report.Scanner.ProfessionalDetails()...)
	details = append(details, report.Validation.FindingsDetails()...)
	details = append(details, report.Verifier.Professional...)
	for k, v := range report.Environment {
		details = append(details, fmt.Sprintf("environment %s=%s", k, v))
	}
	for k, v := range report.Toolchain {
		details = append(details, fmt.Sprintf("toolchain %s=%s", k, v))
	}
	return details
}

func preflightSummary(req CompileRequest) string {
	if strings.TrimSpace(req.SketchPath) == "" || strings.TrimSpace(req.FQBN) == "" {
		return "compile preflight failed"
	}
	return "compile preflight passed"
}

func compatibilityReportSummary(report compatibility.Report) string {
	if len(report.Errors) > 0 {
		return report.Errors[0]
	}
	if len(report.Warnings) > 0 {
		return report.Warnings[0]
	}
	if len(report.Notes) > 0 {
		return report.Notes[0]
	}
	return "compatibility check completed"
}

func compatibilityBeginner(report compatibility.Report) string {
	return compatibilityReportSummary(report)
}

func compatibilityInstallationBeginner(report compatibility.InstallationReport) string {
	if strings.TrimSpace(report.Report.Subject.Name) != "" {
		return compatibilityReportSummary(report.Report)
	}
	return "compatibility check completed"
}

func compatibilityProfessional(report compatibility.Report) []string {
	details := append([]string(nil), report.Notes...)
	details = append(details, report.Warnings...)
	details = append(details, report.Errors...)
	return details
}

func compatibilityInstallationProfessional(report compatibility.InstallationReport) []string {
	details := append([]string(nil), compatibilityProfessional(report.Report)...)
	for _, item := range report.Compatibility {
		if len(item.Professional) > 0 {
			details = append(details, item.Professional...)
		}
	}
	return details
}

func flashPlanProfessional(plan firmware.FlashPlan) []string {
	details := make([]string, 0, len(plan.Entries)+len(plan.Notes))
	for _, entry := range plan.SortedEntries() {
		details = append(details, fmt.Sprintf("0x%x %s -> %s", entry.Offset, entry.Artifact, entry.Path))
	}
	details = append(details, plan.Notes...)
	return details
}

func validationProfessional(report firmware.ValidationReport) []string {
	details := append([]string(nil), report.Warnings...)
	details = append(details, report.Errors...)
	for _, check := range report.Checks {
		if check.Message != "" {
			details = append(details, fmt.Sprintf("%s: %s", check.Name, check.Message))
		}
	}
	return details
}

func compileWorkflowProfessional(report CompileWorkflowReport) []string {
	details := []string{}
	if report.PackagePath != "" {
		details = append(details, "package path: "+report.PackagePath)
	}
	if report.Execution.BuildPath != "" {
		details = append(details, "build path: "+report.Execution.BuildPath)
	}
	if report.Execution.OutputDir != "" {
		details = append(details, "output dir: "+report.Execution.OutputDir)
	}
	details = append(details, report.Package.ProfessionalDetails()...)
	details = append(details, validationProfessional(report.Validation)...)
	details = append(details, compatibilityProfessional(report.Compatibility)...)
	details = append(details, compileWorkflowExecutionDetails(report.Execution)...)
	return details
}

func compileWorkflowExecutionDetails(exec CompileExecution) []string {
	details := []string{}
	if exec.Board != "" {
		details = append(details, "board: "+exec.Board)
	}
	if exec.PlatformPackage != "" {
		details = append(details, "platform package: "+exec.PlatformPackage)
	}
	if exec.PlatformVersion != "" {
		details = append(details, "platform version: "+exec.PlatformVersion)
	}
	if exec.CoreVersion != "" {
		details = append(details, "core version: "+exec.CoreVersion)
	}
	if exec.ToolchainVersion != "" {
		details = append(details, "toolchain version: "+exec.ToolchainVersion)
	}
	if len(exec.Libraries) > 0 {
		for _, lib := range exec.Libraries {
			details = append(details, fmt.Sprintf("library: %s %s", lib.Name, lib.Version))
		}
	}
	return details
}

func compileWorkflowStatus(report CompileWorkflowReport) StepStatus {
	if report.Validation.HasFailures() || !report.ReadyToFlash {
		return StepStatusWarning
	}
	if !report.PackageReady {
		return StepStatusWarning
	}
	return StepStatusPassed
}

func reportStatus(report BootstrapWorkflowReport) StepStatus {
	if report.Verifier.Status == acldiagnostics.StatusFailed || report.ScanValidation.Summary.Errors > 0 {
		return StepStatusFailed
	}
	if report.Verifier.Status == acldiagnostics.StatusWarning || report.ScanValidation.Summary.Warnings > 0 || report.PatchPreview.Summary.WillModify > 0 {
		return StepStatusWarning
	}
	return StepStatusPassed
}

func reportStatusFromDiagnostics(report DiagnosticsWorkflowReport) StepStatus {
	if report.Verifier.Status == acldiagnostics.StatusFailed || report.Validation.Summary.Errors > 0 {
		return StepStatusFailed
	}
	if report.Verifier.Status == acldiagnostics.StatusWarning || report.Validation.Summary.Warnings > 0 {
		return StepStatusWarning
	}
	return StepStatusPassed
}

func statusFromValidation(report aclscanner.ValidationReport) StepStatus {
	if report.Summary.Errors > 0 {
		return StepStatusFailed
	}
	if report.Summary.Warnings > 0 {
		return StepStatusWarning
	}
	return StepStatusPassed
}

func statusFromVerifier(report aclverifier.Report) StepStatus {
	if report.Status == acldiagnostics.StatusFailed {
		return StepStatusFailed
	}
	if report.Status == acldiagnostics.StatusWarning {
		return StepStatusWarning
	}
	return StepStatusPassed
}

func envLookup(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}
