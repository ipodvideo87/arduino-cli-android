package toolcompat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	ValidationSeverityInfo    = "info"
	ValidationSeverityWarning = "warning"
	ValidationSeverityError   = "error"
)

type ValidationReport struct {
	Root     string              `json:"root"`
	Summary  ValidationSummary   `json:"summary"`
	Findings []ValidationFinding `json:"findings,omitempty"`
	Notes    []string            `json:"notes,omitempty"`
}

type ValidationSummary struct {
	TotalFilesScanned        int  `json:"total_files_scanned"`
	ExecutableELFs           int  `json:"executable_elf_count"`
	SharedLibraryRuntimeELFs int  `json:"shared_library_runtime_elf_count"`
	ScriptCount              int  `json:"script_count"`
	UnsupportedIgnoredCount  int  `json:"unsupported_ignored_count"`
	Warnings                 int  `json:"warnings"`
	Errors                   int  `json:"errors"`
	Passed                   bool `json:"passed"`
}

type ValidationFinding struct {
	Severity              string   `json:"severity"`
	RelativePath          string   `json:"relative_path"`
	ExecutableType        string   `json:"executable_type"`
	CompatibilityCategory string   `json:"compatibility_category"`
	PatchClass            string   `json:"patch_class"`
	Messages              []string `json:"messages"`
}

func (r Report) Validate() ValidationReport {
	return ValidateReport(r)
}

func ValidateReport(report Report) ValidationReport {
	result := ValidationReport{
		Root: report.Root,
		Summary: ValidationSummary{
			TotalFilesScanned: report.Summary.TotalEntries,
		},
	}

	for _, entry := range report.Entries {
		switch entry.PatchClass {
		case PatchClassLoaderAndRPath:
			result.Summary.ExecutableELFs++
			validateLoaderAndRPath(&result, entry)
		case PatchClassRuntimeDependency, PatchClassRPathOnly:
			result.Summary.SharedLibraryRuntimeELFs++
			validateRuntimeDependency(&result, entry)
		case PatchClassScript:
			result.Summary.ScriptCount++
			validateScript(&result, entry)
		case PatchClassUnsupported:
			validateUnsupported(&result, entry)
		default:
			if entry.CompatibilityCategory == CategoryAndroidCompatible {
				// Native Android-compatible tools are already valid and do not require ACL patching.
				continue
			}
			addWarning(&result, entry, "unrecognized patch class")
		}
	}

	result.Summary.Warnings = countFindings(result.Findings, ValidationSeverityWarning)
	result.Summary.Errors = countFindings(result.Findings, ValidationSeverityError)
	result.Summary.Passed = result.Summary.Errors == 0
	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].Severity != result.Findings[j].Severity {
			return result.Findings[i].Severity < result.Findings[j].Severity
		}
		return result.Findings[i].RelativePath < result.Findings[j].RelativePath
	})
	return result
}

func (s *Scanner) Validate(root string) (ValidationReport, error) {
	report, err := s.Scan(root)
	if err != nil {
		return ValidationReport{}, err
	}
	return ValidateReport(report), nil
}

func (r ValidationReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func FormatValidationReport(report ValidationReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ACL Toolchain Compatibility Validation\n")
	fmt.Fprintf(&b, "-------------------------------------\n")
	fmt.Fprintf(&b, "Root: %s\n", report.Root)
	fmt.Fprintf(&b, "Total files scanned: %d\n", report.Summary.TotalFilesScanned)
	fmt.Fprintf(&b, "Executable ELF count: %d\n", report.Summary.ExecutableELFs)
	fmt.Fprintf(&b, "Shared-library/runtime ELF count: %d\n", report.Summary.SharedLibraryRuntimeELFs)
	fmt.Fprintf(&b, "Script count: %d\n", report.Summary.ScriptCount)
	fmt.Fprintf(&b, "Unsupported ignored count: %d\n", report.Summary.UnsupportedIgnoredCount)
	fmt.Fprintf(&b, "Warnings: %d\n", report.Summary.Warnings)
	fmt.Fprintf(&b, "Errors: %d\n", report.Summary.Errors)
	if report.Summary.Passed {
		fmt.Fprintf(&b, "Result: PASS\n")
	} else {
		fmt.Fprintf(&b, "Result: FAIL\n")
	}

	if len(report.Findings) == 0 {
		fmt.Fprintln(&b, "Findings: none")
		return b.String()
	}

	warnings := filterFindings(report.Findings, ValidationSeverityWarning)
	errors := filterFindings(report.Findings, ValidationSeverityError)

	if len(errors) > 0 {
		fmt.Fprintln(&b, "Errors:")
		for _, finding := range errors {
			writeFinding(&b, finding)
		}
	}
	if len(warnings) > 0 {
		fmt.Fprintln(&b, "Warnings:")
		for _, finding := range warnings {
			writeFinding(&b, finding)
		}
	}
	return b.String()
}

func validateLoaderAndRPath(report *ValidationReport, entry Entry) {
	ok := true
	messages := []string{}
	if !isELFEntry(entry) {
		ok = false
		messages = append(messages, "expected ELF executable")
	}
	if !isAArch64Machine(entry.Architecture) {
		ok = false
		messages = append(messages, fmt.Sprintf("expected AArch64 architecture, got %q", entry.Architecture))
	}
	if strings.TrimSpace(entry.Interpreter) == "" {
		ok = false
		messages = append(messages, "expected PT_INTERP to be present")
	}
	if !hasAnyRuntimePath(entry) {
		ok = false
		messages = append(messages, "expected RPATH or RUNPATH to be present")
	}
	if !entry.RequiresRuntime {
		ok = false
		messages = append(messages, "expected requires_runtime=true")
	}
	if !ok {
		addError(report, entry, messages...)
	}
}

func validateRuntimeDependency(report *ValidationReport, entry Entry) {
	ok := true
	messages := []string{}
	if !isELFEntry(entry) {
		ok = false
		messages = append(messages, "expected ELF binary")
	}
	if !isAArch64Machine(entry.Architecture) {
		ok = false
		messages = append(messages, fmt.Sprintf("expected AArch64 architecture, got %q", entry.Architecture))
	}
	if strings.TrimSpace(entry.Interpreter) != "" {
		ok = false
		messages = append(messages, "did not expect PT_INTERP for runtime dependency")
	}
	if !hasAnyRuntimePath(entry) {
		ok = false
		messages = append(messages, "expected RPATH or RUNPATH to be present")
	}
	if !entry.RequiresRuntime {
		ok = false
		messages = append(messages, "expected requires_runtime=true")
	}
	if !ok {
		addError(report, entry, messages...)
	}
}

func validateScript(report *ValidationReport, entry Entry) {
	if !isScriptEntry(entry) {
		addError(report, entry, "expected script entry for script-no-elf-patch classification")
	}
}

func validateUnsupported(report *ValidationReport, entry Entry) {
	if isHostExecutableCandidate(entry) {
		addError(report, entry, "host executable could not be classified")
		return
	}
	report.Summary.UnsupportedIgnoredCount++
}

func addWarning(report *ValidationReport, entry Entry, messages ...string) {
	report.Findings = append(report.Findings, ValidationFinding{
		Severity:              ValidationSeverityWarning,
		RelativePath:          entry.RelativePath,
		ExecutableType:        entry.ExecutableType,
		CompatibilityCategory: entry.CompatibilityCategory,
		PatchClass:            entry.PatchClass,
		Messages:              messages,
	})
}

func addError(report *ValidationReport, entry Entry, messages ...string) {
	report.Findings = append(report.Findings, ValidationFinding{
		Severity:              ValidationSeverityError,
		RelativePath:          entry.RelativePath,
		ExecutableType:        entry.ExecutableType,
		CompatibilityCategory: entry.CompatibilityCategory,
		PatchClass:            entry.PatchClass,
		Messages:              messages,
	})
}

func filterFindings(findings []ValidationFinding, severity string) []ValidationFinding {
	var out []ValidationFinding
	for _, finding := range findings {
		if finding.Severity == severity {
			out = append(out, finding)
		}
	}
	return out
}

func countFindings(findings []ValidationFinding, severity string) int {
	count := 0
	for _, finding := range findings {
		if finding.Severity == severity {
			count++
		}
	}
	return count
}

func writeFinding(b *strings.Builder, finding ValidationFinding) {
	fmt.Fprintf(b, "  - %s (%s, %s)\n", finding.RelativePath, finding.CompatibilityCategory, finding.PatchClass)
	fmt.Fprintf(b, "    type: %s\n", finding.ExecutableType)
	for _, msg := range finding.Messages {
		fmt.Fprintf(b, "    %s\n", msg)
	}
}

func hasAnyRuntimePath(entry Entry) bool {
	return strings.TrimSpace(entry.RPath) != "" || strings.TrimSpace(entry.RunPath) != ""
}

func isELFEntry(entry Entry) bool {
	return strings.EqualFold(strings.TrimSpace(entry.ExecutableType), "elf")
}

func isScriptEntry(entry Entry) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(entry.ExecutableType)), "script")
}

func isHostExecutableCandidate(entry Entry) bool {
	switch strings.ToLower(strings.TrimSpace(entry.ExecutableType)) {
	case "elf":
		return isAArch64Machine(entry.Architecture)
	case "binary":
		return true
	default:
		return false
	}
}

func isAArch64Machine(machine string) bool {
	machine = strings.ToLower(strings.TrimSpace(machine))
	return machine == "aarch64" || machine == "em_aarch64" || strings.Contains(machine, "aarch64")
}
