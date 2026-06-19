package runtime

import (
	"fmt"
	"strings"
)

func FormatStatus(report StatusReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ACL Runtime Manager\n")
	fmt.Fprintf(&b, "Root: %s\n", report.Root)
	if report.ActiveRuntimeID == "" {
		fmt.Fprintf(&b, "Active: (none)\n")
	} else {
		fmt.Fprintf(&b, "Active: %s\n", report.ActiveRuntimeID)
	}

	if len(report.Runtimes) == 0 {
		fmt.Fprintf(&b, "Runtimes: none discovered\n")
		return b.String()
	}

	fmt.Fprintf(&b, "Runtimes:\n")
	for _, rt := range report.Runtimes {
		active := ""
		if rt.Active {
			active = " active"
		}
		fmt.Fprintf(&b, "- %s%s\n", rt.ID, active)
		fmt.Fprintf(&b, "  version: %s\n", rt.RuntimeVersion)
		fmt.Fprintf(&b, "  arch: %s\n", rt.Architecture)
		fmt.Fprintf(&b, "  abis: %s\n", strings.Join(rt.SupportedABIs, ", "))
		fmt.Fprintf(&b, "  compatibility: %s\n", rt.CompatibilityLevel)
		fmt.Fprintf(&b, "  status: %s\n", rt.Status)
		fmt.Fprintf(&b, "  path: %s\n", rt.Path)
	}
	return b.String()
}

func FormatValidation(report ValidationReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Runtime %s\n", report.RuntimeID)
	fmt.Fprintf(&b, "Path: %s\n", report.Path)
	fmt.Fprintf(&b, "Status: %s\n", report.Status)
	for _, check := range report.Checks {
		fmt.Fprintf(&b, "- %s: %s", check.Name, check.Status)
		if check.Message != "" {
			fmt.Fprintf(&b, " (%s)", check.Message)
		}
		if check.Path != "" {
			fmt.Fprintf(&b, " [%s]", check.Path)
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}
