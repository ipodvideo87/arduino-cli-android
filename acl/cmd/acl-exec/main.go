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
	request := aclexec.Request{
		RuntimeRoot: strings.TrimSpace(*runtimeRoot),
		TargetPath:  strings.TrimSpace(*targetPath),
		Cwd:         strings.TrimSpace(*cwd),
		Args:        remaining,
		Apply:       *apply,
	}
	plan, result, err := planner.Run(request)
	report := aclexec.BuildDiagnosticReport(plan, request, result)
	fmt.Fprint(stdout, aclexec.FormatDiagnosticReport(report))
	if err != nil {
		fmt.Fprintln(stderr, err)
		if *apply && result.ExitCode != 0 {
			return result.ExitCode
		}
		return 1
	}
	return 0
}
