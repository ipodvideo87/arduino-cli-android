package acl

import (
	"context"
	"fmt"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/evidence"
	"github.com/spf13/cobra"
)

type evidenceCollector interface {
	Collect(context.Context, evidence.CollectOptions) (evidence.EvidenceBundle, error)
}

var newEvidenceCollector = func() evidenceCollector {
	return evidence.NewCollector()
}

type evidenceCollectOptions struct {
	outputDir           string
	includeStreamStatus bool
}

func newEvidenceCommand() *cobra.Command {
	opts := evidenceCollectOptions{}
	var devicePath string
	cmd := &cobra.Command{
		Use:   "evidence",
		Short: "Collect structured native diagnostics evidence",
	}
	collectCmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect a structured native diagnostics evidence bundle",
		Long: strings.TrimSpace(`
Collect a structured native diagnostics evidence bundle from allowlisted,
read-only ACL commands. The collector writes one JSON artifact per run and
prints a concise summary with the artifact path.

The JSON fields are nested under repository.*, environment.*, binary.*, and
device_path. For a quick summary, use:

  jq -r '{repo: .repository.root, branch: .repository.branch, commit: .repository.commit, native_termux: .environment.native_termux, binary: .binary.path, device: .device_path}'

`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			collector := newEvidenceCollector()
			bundle, err := collector.Collect(cmd.Context(), evidence.CollectOptions{
				DevicePath:          devicePath,
				OutputDir:           opts.outputDir,
				IncludeStreamStatus: opts.includeStreamStatus,
			})
			if err != nil {
				return err
			}
			if err := writeEvidenceSummary(cmd, bundle); err != nil {
				return err
			}
			return nil
		},
	}
	collectCmd.Flags().StringVar(&devicePath, "device", "", "USB device path to include in the evidence bundle")
	collectCmd.Flags().StringVar(&opts.outputDir, "output-dir", "", "Directory for the JSON evidence artifact")
	collectCmd.Flags().BoolVar(&opts.includeStreamStatus, "include-stream-status", false, "Include the read-only stream-status command")
	_ = collectCmd.MarkFlagRequired("device")
	cmd.AddCommand(collectCmd)
	return cmd
}

func writeEvidenceSummary(cmd *cobra.Command, bundle evidence.EvidenceBundle) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Evidence Collector")
	fmt.Fprintf(cmd.OutOrStdout(), "Artifact: %s\n", bundle.OutputPath)
	fmt.Fprintf(cmd.OutOrStdout(), "Repository: %s\n", bundle.Repository.Root)
	if bundle.Repository.Branch != "" || bundle.Repository.Commit != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Branch: %s\n", bundle.Repository.Branch)
		fmt.Fprintf(cmd.OutOrStdout(), "Commit: %s\n", bundle.Repository.Commit)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Binary: %s\n", bundle.Binary.Path)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", bundle.Status)
	if strings.TrimSpace(bundle.Summary) != "" {
		fmt.Fprintln(cmd.OutOrStdout(), bundle.Summary)
	}
	return nil
}
