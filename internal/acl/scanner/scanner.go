package scanner

import "github.com/arduino/arduino-cli/internal/acl/toolcompat"

type (
	Report            = toolcompat.Report
	Entry             = toolcompat.Entry
	Summary           = toolcompat.Summary
	ValidationReport  = toolcompat.ValidationReport
	ValidationSummary = toolcompat.ValidationSummary
	ValidationFinding = toolcompat.ValidationFinding
	Scanner           = toolcompat.Scanner
)

func New() *Scanner {
	return toolcompat.NewScanner()
}

func Validate(report Report) ValidationReport {
	return toolcompat.ValidateReport(report)
}
