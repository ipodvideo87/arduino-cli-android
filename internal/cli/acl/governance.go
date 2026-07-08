package acl

import (
	"fmt"
	"os"
	"strings"

	aclgovernance "github.com/arduino/arduino-cli/internal/acl/governance"
	"github.com/spf13/cobra"
)

var governanceRepoRootFunc = func() string {
	return canonicalRepoRoot()
}

var governanceWorkingDirFunc = func() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func newGovernanceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "governance",
		Short: "Validate repository governance and documentation currency",
	}
	cmd.AddCommand(newGovernanceValidateCommand())
	return cmd
}

func newGovernanceValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate read-only governance rules and documentation boundaries",
		Long: strings.TrimSpace(`
Validate read-only repository governance rules, documentation boundaries, and
task-recovery state. The command reports all findings together and exits
non-zero if any required check fails.
`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report := aclgovernance.Validate(aclgovernance.Options{
				RepoRoot:         governanceRepoRootFunc(),
				ExpectedRepoRoot: canonicalRepoRoot(),
				WorkingDir:       governanceWorkingDirFunc(),
			})
			if isJSON(cmd) {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if err := writeGovernanceValidationReport(cmd, report); err != nil {
					return err
				}
			}
			if report.HasFailures() {
				return fmt.Errorf("governance validation failed with %d issue(s)", report.FailureCount)
			}
			return nil
		},
	}
	return cmd
}

func writeGovernanceValidationReport(cmd *cobra.Command, report aclgovernance.Report) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Governance Validation")
	fmt.Fprintf(cmd.OutOrStdout(), "Repo root: %s\n", report.RepoRoot)
	fmt.Fprintf(cmd.OutOrStdout(), "Working dir: %s\n", report.WorkingDir)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	fmt.Fprintf(cmd.OutOrStdout(), "Summary: %s\n", report.Summary)
	for _, check := range report.Checks {
		outcome := "PASS"
		if !check.Passed {
			outcome = "FAIL"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "- %s: %s\n", check.Name, outcome)
		for _, message := range check.Messages {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", message)
		}
	}
	return nil
}
