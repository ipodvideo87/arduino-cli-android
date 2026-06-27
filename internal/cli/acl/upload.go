package acl

import (
	"context"
	"fmt"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/engine"
	"github.com/arduino/arduino-cli/internal/acl/upload"
	"github.com/spf13/cobra"
)

var workflowUploadRun = runWorkflowUpload

type workflowUploadRequest struct {
	PackageDir string
}

func newWorkflowUploadCommand() *cobra.Command {
	opts := transportCommandOptions{}
	var packageDir string
	cmd := &cobra.Command{
		Use:   "upload <firmware-package>",
		Short: "Run the ACL upload dry-run workflow",
		Long: strings.TrimSpace(`
Run the ACL upload dry-run workflow against a firmware package.

This command is dry-run only by design. It validates the package, derives the
ordered upload plan, and reports diagnostics without opening a transport
stream or writing to hardware.
`),
		Example: "arduino-cli acl workflow upload ~/Development/Sketches/build/esp32.esp32.esp32s3/firmware-package",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageDir = args[0]
			report, err := workflowUploadRun(cmd.Context(), workflowUploadRequest{PackageDir: packageDir})
			if isJSON(cmd) {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if err := writeUploadWorkflowReport(cmd, report, opts.details); err != nil {
					return err
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	return cmd
}

func runWorkflowUpload(ctx context.Context, req workflowUploadRequest) (engine.WorkflowReport, error) {
	if strings.TrimSpace(req.PackageDir) == "" {
		return engine.WorkflowReport{}, fmt.Errorf("firmware package directory is required")
	}
	wctx := engine.NewContext()
	wctx.UploadRequest = upload.UploadRequest{
		PackageDir: req.PackageDir,
	}
	return engine.New().Run(ctx, engine.UploadWorkflow(), wctx)
}

func writeUploadWorkflowReport(cmd *cobra.Command, report engine.WorkflowReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Workflow Upload")
	fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", report.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	if details {
		if result, ok := report.Result.(upload.UploadReport); ok {
			for _, detail := range result.ProfessionalDetails() {
				fmt.Fprintln(cmd.OutOrStdout(), detail)
			}
		} else {
			for _, detail := range report.ProfessionalDetails() {
				fmt.Fprintln(cmd.OutOrStdout(), detail)
			}
		}
	}
	return nil
}
