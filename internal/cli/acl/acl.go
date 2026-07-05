package acl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
	aclengine "github.com/arduino/arduino-cli/internal/acl/engine"
	aclinstall "github.com/arduino/arduino-cli/internal/acl/install"
	aclpatcher "github.com/arduino/arduino-cli/internal/acl/patcher"
	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
	aclscanner "github.com/arduino/arduino-cli/internal/acl/scanner"
	acltoolcompat "github.com/arduino/arduino-cli/internal/acl/toolcompat"
	aclverifier "github.com/arduino/arduino-cli/internal/acl/verifier"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
	"github.com/spf13/cobra"
)

var (
	scanRootFunc      = defaultRoot
	previewRootFunc   = defaultRoot
	verifyRootFunc    = defaultRoot
	bootstrapRootFunc = defaultRoot

	bootstrapRuntimeFunc = defaultRuntimeRoot

	newScannerFunc   = aclscanner.New
	newVerifierFunc  = aclverifier.New
	previewPatchFunc = aclpatcher.DryRun
	newBootstrapExec = func(report *BootstrapReport) aclinstall.StageExecutor { return &bootstrapStageExecutor{report: report} }
)

func NewCommand(srv rpc.ArduinoCoreServiceServer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acl",
		Short: "Android Compatibility Layer diagnostics and bootstrap commands",
	}
	cmd.AddCommand(newScanCommand())
	cmd.AddCommand(newVerifyCommand())
	cmd.AddCommand(newPatchPreviewCommand())
	cmd.AddCommand(newBootstrapCommand())
	cmd.AddCommand(newEvidenceCommand())
	cmd.AddCommand(newTransportCommand())
	cmd.AddCommand(newWorkflowCommand(srv))
	return cmd
}

type scanCommandOptions struct {
	details bool
}

func newScanCommand() *cobra.Command {
	opts := scanCommandOptions{}
	cmd := &cobra.Command{
		Use:   "scan [root]",
		Short: "Scan installed tools and executables",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := scanRootFunc()
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				root = args[0]
			}
			report, err := newScannerFunc().Scan(root)
			if err != nil {
				return err
			}
			if isJSON(cmd) {
				return writeJSON(cmd, report)
			}
			return writeScanReport(cmd, report, opts.details)
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	return cmd
}

type verifyCommandOptions struct {
	details bool
}

func newVerifyCommand() *cobra.Command {
	opts := verifyCommandOptions{}
	var runtimeRoot string
	var targetPath string
	cmd := &cobra.Command{
		Use:   "verify [root]",
		Short: "Run preflight verification for installed toolchains",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := verifyRootFunc()
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				root = args[0]
			}
			report, err := newVerifierFunc(runtimeRoot).Verify(aclverifier.Request{
				Root:        root,
				TargetPath:  targetPath,
				RuntimeRoot: runtimeRoot,
			})
			if err != nil && strings.TrimSpace(report.Beginner) == "" {
				report.Beginner = err.Error()
			}
			if isJSON(cmd) {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if err := writeVerifierReport(cmd, report, opts.details); err != nil {
					return err
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&runtimeRoot, "runtime-root", bootstrapRuntimeFunc(), "Android ACL runtime root")
	cmd.Flags().StringVar(&targetPath, "target", "", "Executable to preflight")
	return cmd
}

type patchPreviewCommandOptions struct {
	details bool
}

func newPatchPreviewCommand() *cobra.Command {
	opts := patchPreviewCommandOptions{}
	cmd := &cobra.Command{
		Use:   "patch-preview [root]",
		Short: "Preview Android patch changes without modifying files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := previewRootFunc()
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				root = args[0]
			}
			report, err := previewPatchFunc(root)
			if err != nil {
				return err
			}
			if isJSON(cmd) {
				return writeJSON(cmd, report)
			}
			return writePatchPreviewReport(cmd, report, opts.details)
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	return cmd
}

type bootstrapCommandOptions struct {
	details bool
}

func newBootstrapCommand() *cobra.Command {
	opts := bootstrapCommandOptions{}
	var runtimeRoot string
	var targetPath string
	cmd := &cobra.Command{
		Use:   "bootstrap [root]",
		Short: "Run Android bootstrap checks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := bootstrapRootFunc()
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				root = args[0]
			}
			report, err := buildBootstrapReport(root, runtimeRoot, targetPath)
			if isJSON(cmd) {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if err := writeBootstrapReport(cmd, report, opts.details); err != nil {
					return err
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&runtimeRoot, "runtime-root", bootstrapRuntimeFunc(), "Android ACL runtime root")
	cmd.Flags().StringVar(&targetPath, "target", "", "Executable to include in bootstrap checks")
	return cmd
}

func newWorkflowCommand(srv rpc.ArduinoCoreServiceServer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Experimental ACL workflow engine commands",
	}
	cmd.AddCommand(newWorkflowCompileCommand(srv))
	cmd.AddCommand(newWorkflowUploadCommand())
	cmd.AddCommand(newWorkflowBootstrapCommand())
	cmd.AddCommand(newWorkflowDiagnosticsCommand())
	return cmd
}

func newWorkflowBootstrapCommand() *cobra.Command {
	opts := bootstrapCommandOptions{}
	var runtimeRoot string
	var targetPath string
	cmd := &cobra.Command{
		Use:   "bootstrap [root]",
		Short: "Run the ACL bootstrap workflow through the engine",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := aclengine.NewContext()
			ctx.Root = bootstrapRootFunc()
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				ctx.Root = args[0]
			}
			ctx.RuntimeRoot = runtimeRoot
			ctx.TargetPath = targetPath
			report, err := aclengine.New().Run(cmd.Context(), aclengine.BootstrapWorkflow(), ctx)
			if isJSON(cmd) {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if err := writeWorkflowReport(cmd, report, opts.details); err != nil {
					return err
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&runtimeRoot, "runtime-root", bootstrapRuntimeFunc(), "Android ACL runtime root")
	cmd.Flags().StringVar(&targetPath, "target", "", "Executable to include in bootstrap checks")
	return cmd
}

func newWorkflowDiagnosticsCommand() *cobra.Command {
	opts := bootstrapCommandOptions{}
	var runtimeRoot string
	var targetPath string
	cmd := &cobra.Command{
		Use:   "diagnostics [root]",
		Short: "Run the ACL diagnostics workflow through the engine",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := aclengine.NewContext()
			ctx.Root = bootstrapRootFunc()
			if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
				ctx.Root = args[0]
			}
			ctx.RuntimeRoot = runtimeRoot
			ctx.TargetPath = targetPath
			report, err := aclengine.New().Run(cmd.Context(), aclengine.DiagnosticsWorkflow(), ctx)
			if isJSON(cmd) {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if err := writeWorkflowReport(cmd, report, opts.details); err != nil {
					return err
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&runtimeRoot, "runtime-root", bootstrapRuntimeFunc(), "Android ACL runtime root")
	cmd.Flags().StringVar(&targetPath, "target", "", "Executable to include in diagnostics")
	return cmd
}

type BootstrapReport struct {
	Root           string                      `json:"root"`
	RuntimeRoot    string                      `json:"runtime_root,omitempty"`
	Environment    BootstrapEnvironment        `json:"environment"`
	Scan           aclscanner.Report           `json:"scan,omitempty"`
	ScanValidation aclscanner.ValidationReport `json:"scan_validation,omitempty"`
	Verifier       aclverifier.Report          `json:"verifier,omitempty"`
	PatchPreview   aclpatcher.Report           `json:"patch_preview,omitempty"`
	Pipeline       aclinstall.PatchManifest    `json:"pipeline"`
	Status         acldiagnostics.Status       `json:"status"`
	Beginner       string                      `json:"beginner_summary,omitempty"`
	Details        []string                    `json:"professional_details,omitempty"`
}

type BootstrapEnvironment struct {
	Platform      string `json:"platform"`
	GoOS          string `json:"go_os"`
	GoArch        string `json:"go_arch"`
	RuntimeRoot   string `json:"runtime_root,omitempty"`
	TermuxVersion string `json:"termux_version,omitempty"`
	AndroidRoot   string `json:"android_root,omitempty"`
	AndroidData   string `json:"android_data,omitempty"`
}

func buildBootstrapReport(root, runtimeRoot, targetPath string) (BootstrapReport, error) {
	scanner := newScannerFunc()
	scan, err := scanner.Scan(root)
	if err != nil {
		return BootstrapReport{}, err
	}
	scanValidation := aclscanner.Validate(scan)

	preview, err := previewPatchFunc(root)
	if err != nil {
		return BootstrapReport{}, err
	}

	verifier := newVerifierFunc(runtimeRoot)
	verifyReport, verifyErr := verifier.Verify(aclverifier.Request{
		Root:        root,
		TargetPath:  targetPath,
		RuntimeRoot: runtimeRoot,
	})

	report := BootstrapReport{
		Root:           root,
		RuntimeRoot:    runtimeRoot,
		Environment:    detectBootstrapEnvironment(runtimeRoot),
		Scan:           scan,
		ScanValidation: scanValidation,
		Verifier:       verifyReport,
		PatchPreview:   preview,
		Status:         acldiagnostics.StatusPassed,
	}

	if verifyErr != nil {
		report.Status = acldiagnostics.StatusFailed
		report.Beginner = verifyErr.Error()
		report.Details = report.collectDetails()
		report.Pipeline = aclinstall.PatchManifest{
			PackageName:    "acl-bootstrap",
			PackageVersion: "1",
			Source:         "bootstrap",
			Metadata: map[string]string{
				"root":        root,
				"runtimeRoot": runtimeRoot,
			},
		}
		return report, verifyErr
	}

	manifest := aclinstall.PatchManifest{
		PackageName:    "acl-bootstrap",
		PackageVersion: "1",
		Source:         "bootstrap",
		Metadata: map[string]string{
			"root":        root,
			"runtimeRoot": runtimeRoot,
		},
	}
	pipeline := aclinstall.NewAndroidInstallPatchPipeline(newBootstrapExec(&report))
	if err := pipeline.Run(context.Background(), &manifest); err != nil {
		report.Status = acldiagnostics.StatusFailed
		report.Pipeline = manifest
		report.Beginner = err.Error()
		report.Details = report.collectDetails()
		return report, err
	}

	report.Pipeline = manifest
	report.Status = manifest.Status
	report.Beginner = report.collectBeginner()
	report.Details = report.collectDetails()
	return report, nil
}

func (r BootstrapReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func (r BootstrapReport) collectBeginner() string {
	parts := []string{}
	if r.ScanValidation.Summary.TotalFilesScanned > 0 {
		parts = append(parts, r.ScanValidation.BeginnerSummary())
	}
	if r.Verifier.Beginner != "" {
		parts = append(parts, r.Verifier.Beginner)
	}
	if r.PatchPreview.Summary.TotalCandidates > 0 {
		parts = append(parts, r.PatchPreview.BeginnerSummary())
	}
	if len(parts) == 0 {
		parts = append(parts, "bootstrap completed")
	}
	return strings.Join(parts, "; ")
}

func (r BootstrapReport) collectDetails() []string {
	details := append([]string(nil), r.Scan.ProfessionalDetails()...)
	details = append(details, r.ScanValidation.FindingsDetails()...)
	details = append(details, r.Verifier.Professional...)
	details = append(details, r.PatchPreview.ProfessionalDetails()...)
	details = append(details, r.Pipeline.StageSummaries()...)
	for _, stage := range r.Pipeline.Stages {
		if len(stage.Evidence) > 0 {
			details = append(details, fmt.Sprintf("%s evidence: %s", stage.Name, strings.Join(stage.Evidence, ", ")))
		}
	}
	details = append(details, fmt.Sprintf("environment: %s", r.Environment.Platform))
	if r.Environment.RuntimeRoot != "" {
		details = append(details, "runtime root: "+r.Environment.RuntimeRoot)
	}
	return details
}

type bootstrapStageExecutor struct {
	report *BootstrapReport
}

func (e *bootstrapStageExecutor) Execute(_ context.Context, req aclinstall.StageRequest) (aclinstall.StageResult, error) {
	if e == nil || e.report == nil {
		return aclinstall.StageResult{Status: acldiagnostics.StatusFailed, Message: "bootstrap report is unavailable"}, nil
	}

	switch req.Stage {
	case aclinstall.StageDownload, aclinstall.StageExtract, aclinstall.StageRegister:
		return aclinstall.StageResult{Status: acldiagnostics.StatusSkipped, Message: "not part of read-only bootstrap checks"}, nil
	case aclinstall.StageAndroidPatch:
		if e.report.PatchPreview.Summary.TotalCandidates > 0 {
			return aclinstall.StageResult{Status: acldiagnostics.StatusPassed, Message: "patch preview available"}, nil
		}
		return aclinstall.StageResult{Status: acldiagnostics.StatusPassed, Message: "no Android patch changes required"}, nil
	case aclinstall.StagePermissionRuntimeFixes:
		if e.report.PatchPreview.Summary.PermissionRepairs > 0 {
			return aclinstall.StageResult{
				Status:   acldiagnostics.StatusWarning,
				Message:  "runtime permission repairs would be required",
				Evidence: []string{".acl/runtime/ld-linux-aarch64.so.1"},
			}, nil
		}
		return aclinstall.StageResult{Status: acldiagnostics.StatusPassed, Message: "no runtime permission repairs required"}, nil
	case aclinstall.StageExecutableValidation:
		if e.report.ScanValidation.Summary.Errors > 0 {
			return aclinstall.StageResult{Status: acldiagnostics.StatusFailed, Message: "scanner validation failed"}, nil
		}
		if e.report.ScanValidation.Summary.Warnings > 0 {
			return aclinstall.StageResult{Status: acldiagnostics.StatusWarning, Message: "scanner validation produced warnings"}, nil
		}
		return aclinstall.StageResult{Status: acldiagnostics.StatusPassed, Message: "scanner validation passed"}, nil
	case aclinstall.StageSelfTest:
		if e.report.Verifier.Status == acldiagnostics.StatusFailed {
			return aclinstall.StageResult{Status: acldiagnostics.StatusFailed, Message: "verifier failed"}, nil
		}
		if strings.TrimSpace(e.report.Verifier.Request.TargetPath) == "" {
			return aclinstall.StageResult{Status: acldiagnostics.StatusSkipped, Message: "no target executable provided"}, nil
		}
		return aclinstall.StageResult{Status: acldiagnostics.StatusPassed, Message: "preflight target verified"}, nil
	case aclinstall.StageReady:
		if e.report.Status == acldiagnostics.StatusFailed {
			return aclinstall.StageResult{Status: acldiagnostics.StatusFailed, Message: "bootstrap checks failed"}, nil
		}
		return aclinstall.StageResult{Status: acldiagnostics.StatusPassed, Message: "bootstrap checks complete"}, nil
	default:
		return aclinstall.StageResult{Status: acldiagnostics.StatusSkipped, Message: "unhandled stage"}, nil
	}
}

func writeJSON(cmd *cobra.Command, value any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func writeScanReport(cmd *cobra.Command, report aclscanner.Report, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Scanner")
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	if details {
		for _, detail := range report.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
	}
	return nil
}

func writeVerifierReport(cmd *cobra.Command, report aclverifier.Report, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Verifier")
	fmt.Fprintln(cmd.OutOrStdout(), report.Beginner)
	if details {
		for _, detail := range report.Professional {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
	}
	return nil
}

func writePatchPreviewReport(cmd *cobra.Command, report aclpatcher.Report, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Patch Preview")
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	if details {
		for _, detail := range report.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
	}
	return nil
}

func writeBootstrapReport(cmd *cobra.Command, report BootstrapReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Bootstrap")
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	fmt.Fprintln(cmd.OutOrStdout(), report.Beginner)
	if details {
		for _, detail := range report.Details {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
	}
	return nil
}

func writeWorkflowReport(cmd *cobra.Command, report aclengine.WorkflowReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Workflow")
	fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", report.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	if details {
		for _, detail := range report.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
	}
	return nil
}

func isJSON(cmd *cobra.Command) bool {
	value, _ := cmd.Flags().GetBool("json")
	return value
}

func defaultRoot() string {
	root, err := acltoolcompat.DefaultPackagesRoot()
	if err != nil || strings.TrimSpace(root) == "" {
		return "."
	}
	return root
}

func defaultRuntimeRoot() string {
	root, err := aclruntime.DefaultRoot()
	if err != nil || strings.TrimSpace(root) == "" {
		return "."
	}
	return root
}

func detectBootstrapEnvironment(runtimeRoot string) BootstrapEnvironment {
	return BootstrapEnvironment{
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		GoOS:          runtime.GOOS,
		GoArch:        runtime.GOARCH,
		RuntimeRoot:   strings.TrimSpace(runtimeRoot),
		TermuxVersion: strings.TrimSpace(os.Getenv("TERMUX_VERSION")),
		AndroidRoot:   strings.TrimSpace(os.Getenv("ANDROID_ROOT")),
		AndroidData:   strings.TrimSpace(os.Getenv("ANDROID_DATA")),
	}
}
