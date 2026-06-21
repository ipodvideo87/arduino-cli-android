package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	aclexec "github.com/arduino/arduino-cli/internal/acl/exec"
)

type plannerRunner interface {
	Run(aclexec.Request) (aclexec.ExecutionPlan, aclexec.Result, error)
}

var newPlanner = func(runtimeRoot string) plannerRunner {
	return aclexec.NewPlanner(runtimeRoot)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("acl-exec", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		runtimeRoot = fs.String("runtime-root", "", "runtime root directory")
		targetPath  = fs.String("target", "", "target executable")
		cwd         = fs.String("cwd", "", "working directory for execution")
		apply       = fs.Bool("apply", false, "execute the plan instead of dry-running it")
	)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	remaining := fs.Args()
	planner := newPlanner(strings.TrimSpace(*runtimeRoot))
	plan, result, err := planner.Run(aclexec.Request{
		RuntimeRoot: strings.TrimSpace(*runtimeRoot),
		TargetPath:  strings.TrimSpace(*targetPath),
		Cwd:         strings.TrimSpace(*cwd),
		Args:        remaining,
		Apply:       *apply,
	})
	fmt.Fprint(stdout, formatPlan(plan))
	if err != nil {
		fmt.Fprintln(stderr, err)
		if *apply && result.ExitCode != 0 {
			return result.ExitCode
		}
		return 1
	}
	if *apply {
		fmt.Fprintf(stdout, "stdout: %s\n", result.Stdout)
		fmt.Fprintf(stdout, "stderr: %s\n", result.Stderr)
		fmt.Fprintf(stdout, "exit code: %d\n", result.ExitCode)
	}
	return 0
}

func formatPlan(plan aclexec.ExecutionPlan) string {
	var b strings.Builder
	fmt.Fprintln(&b, "ACL Execution Planner")
	fmt.Fprintln(&b, "---------------------")
	fmt.Fprintf(&b, "Target: %s\n", plan.TargetPath)
	fmt.Fprintf(&b, "Allowed: %t\n", plan.Allowed)
	if plan.RuntimeID != "" {
		fmt.Fprintf(&b, "Runtime ID: %s\n", plan.RuntimeID)
		fmt.Fprintf(&b, "Runtime path: %s\n", plan.RuntimePath)
		fmt.Fprintf(&b, "Loader: %s\n", plan.LoaderPath)
		if len(plan.LibraryPaths) > 0 {
			fmt.Fprintln(&b, "Library files:")
			for _, path := range plan.LibraryPaths {
				fmt.Fprintf(&b, "  - %s\n", path)
			}
		}
		if plan.LibrarySearchPath != "" {
			fmt.Fprintf(&b, "Library search path: %s\n", plan.LibrarySearchPath)
		}
	}
	if plan.Cwd != "" {
		fmt.Fprintf(&b, "CWD: %s\n", plan.Cwd)
	}
	if plan.LaunchMode != "" {
		fmt.Fprintf(&b, "Launch mode: %s\n", plan.LaunchMode)
	}
	if len(plan.Argv) > 0 {
		fmt.Fprintf(&b, "Argv: %s\n", strings.Join(plan.Argv, " "))
	}
	if len(plan.Environment) > 0 {
		fmt.Fprintln(&b, "Environment:")
		for _, kv := range plan.Environment {
			fmt.Fprintf(&b, "  - %s\n", kv)
		}
	}
	if len(plan.Warnings) > 0 {
		fmt.Fprintln(&b, "Warnings:")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(&b, "  - %s\n", warning)
		}
	}
	if len(plan.Errors) > 0 {
		fmt.Fprintln(&b, "Errors:")
		for _, err := range plan.Errors {
			fmt.Fprintf(&b, "  - %s\n", err)
		}
	}
	fmt.Fprintf(&b, "Apply mode: %t\n", plan.Apply)
	if len(plan.Command) > 0 {
		fmt.Fprintf(&b, "Command: %s\n", strings.Join(plan.Command, " "))
	}
	return b.String()
}
