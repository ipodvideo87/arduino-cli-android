// Command acl-verify performs a suite of Android/Termux pre-flight environment
// checks and reports the results with human-readable remediation hints.
//
// Usage:
//
//	acl-verify [flags]
//
// Flags:
//
//	--check name   Run only the named check (can be repeated)
//	--json         Emit results as JSON to stdout
//	--list         List all available check names and exit
//	--quiet        Suppress per-check output; only exit code communicates status
//
// Exit codes:
//
//	0   All checks passed
//	2   A required dependency (patchelf, linker) is missing
//	3   SELinux enforcing mode detected with a dangerous process context
//	4   Filesystem / PREFIX accessibility problem
//	5   W^X restriction detected
//	6   /proc filesystem restricted
//	99  Internal error during a check
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/arduino/arduino-cli/acl/verifier"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		// cobra already prints the error; use exit code 1 for CLI errors.
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var (
		checks []string
		asJSON bool
		list   bool
		quiet  bool
	)

	cmd := &cobra.Command{
		Use:   "acl-verify",
		Short: "Android/Termux pre-flight environment checker for the ACL",
		Long: `acl-verify validates the Android execution environment before any ACL tool
launch. It checks Termux PREFIX accessibility, SELinux enforcement, /proc
readability, W^X restrictions, and the presence of required packages
(patchelf, dynamic linker).

Each failed check prints a remediation hint. The exit code reflects the most
severe failure category encountered:

  0  All checks passed
  2  Missing dependency  (patchelf, linker)
  3  SELinux enforcing with dangerous context
  4  Filesystem / PREFIX problem
  5  W^X restriction on Termux directories
  6  /proc filesystem restricted
  99 Internal error`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				printCheckList()
				return nil
			}

			var results []verifier.Result
			if len(checks) > 0 {
				// Validate requested check names before running.
				known := make(map[string]bool, len(verifier.All))
				for _, c := range verifier.All {
					known[c.Name] = true
				}
				for _, name := range checks {
					if !known[name] {
						return fmt.Errorf("unknown check name %q (use --list to see available checks)", name)
					}
				}
				results = verifier.RunSelected(checks)
			} else {
				results = verifier.RunAll()
			}

			if asJSON {
				return emitJSON(results)
			}

			if !quiet {
				emitText(results)
			}

			code := verifier.OverallExitCode(results)
			if code != verifier.ExitOK {
				os.Exit(int(code))
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&checks, "check", nil,
		"Run only the named check (repeat for multiple checks)")
	cmd.Flags().BoolVar(&asJSON, "json", false,
		"Emit results as a JSON array to stdout")
	cmd.Flags().BoolVar(&list, "list", false,
		"List all available check names and descriptions, then exit")
	cmd.Flags().BoolVar(&quiet, "quiet", false,
		"Suppress per-check output; communicate status only via exit code")

	return cmd
}

// printCheckList prints all registered checks to stdout.
func printCheckList() {
	fmt.Println("Available checks:")
	for _, c := range verifier.All {
		fmt.Printf("  %-28s %s\n", c.Name, c.Description)
	}
}

// emitText writes human-readable results to stdout.
func emitText(results []verifier.Result) {
	passed := 0
	failed := 0
	for _, r := range results {
		fmt.Println(r.String())
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("\n%d/%d checks passed", passed, len(results))
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()
}

// jsonResult is the JSON serialisation shape for a single Result.
type jsonResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Code    int    `json:"exit_code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// emitJSON serialises results to stdout as a JSON array.
func emitJSON(results []verifier.Result) error {
	out := make([]jsonResult, len(results))
	for i, r := range results {
		out[i] = jsonResult{
			Name:    r.Name,
			Passed:  r.Passed,
			Code:    int(r.Code),
			Message: r.Message,
			Hint:    r.Hint,
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("JSON encode: %w", err)
	}
	// Still exit with the appropriate code even in JSON mode.
	code := verifier.OverallExitCode(results)
	if code != verifier.ExitOK {
		os.Exit(int(code))
	}
	return nil
}
