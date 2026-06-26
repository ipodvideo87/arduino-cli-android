package verifier

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/diagnostics"
	aclexec "github.com/arduino/arduino-cli/internal/acl/exec"
	aclscan "github.com/arduino/arduino-cli/internal/acl/scanner"
)

type Request struct {
	Root        string   `json:"root,omitempty"`
	TargetPath  string   `json:"target_path,omitempty"`
	RuntimeRoot string   `json:"runtime_root,omitempty"`
	Cwd         string   `json:"cwd,omitempty"`
	Args        []string `json:"args,omitempty"`
}

type Report struct {
	Request      Request                  `json:"request"`
	Scan         aclscan.Report           `json:"scan,omitempty"`
	Validation   aclscan.ValidationReport `json:"validation,omitempty"`
	Execution    aclexec.DiagnosticReport `json:"execution,omitempty"`
	Status       diagnostics.Status       `json:"status"`
	Beginner     string                   `json:"beginner_summary,omitempty"`
	Professional []string                 `json:"professional_details,omitempty"`
}

type Verifier struct {
	scanner *aclscan.Scanner
	planner *aclexec.Planner
}

func New(runtimeRoot string) *Verifier {
	return &Verifier{
		scanner: aclscan.New(),
		planner: aclexec.NewPlanner(runtimeRoot),
	}
}

func (v *Verifier) Verify(req Request) (Report, error) {
	if v == nil {
		return Report{}, fmt.Errorf("verifier is nil")
	}
	if v.scanner == nil {
		v.scanner = aclscan.New()
	}
	if v.planner == nil {
		v.planner = aclexec.NewPlanner(req.RuntimeRoot)
	}

	report := Report{Request: req}
	if strings.TrimSpace(req.Root) != "" {
		scan, err := v.scanner.Scan(req.Root)
		if err != nil {
			return Report{}, err
		}
		report.Scan = scan
		report.Validation = aclscan.Validate(scan)
	}

	if strings.TrimSpace(req.TargetPath) != "" {
		plan, result, err := v.planner.Run(aclexec.Request{
			RuntimeRoot: req.RuntimeRoot,
			TargetPath:  req.TargetPath,
			Cwd:         req.Cwd,
			Args:        req.Args,
		})
		report.Execution = aclexec.BuildDiagnosticReport(plan, aclexec.Request{
			RuntimeRoot: req.RuntimeRoot,
			TargetPath:  req.TargetPath,
			Cwd:         req.Cwd,
			Args:        req.Args,
		}, result)
		if err != nil {
			report.Status = diagnostics.StatusFailed
			report.Beginner = err.Error()
			report.Professional = append(report.Professional, report.Execution.Hints...)
			return report, err
		}
	}

	report.Status = diagnostics.StatusPassed
	if report.Validation.Summary.Errors > 0 || report.Execution.Result.StartError != "" {
		report.Status = diagnostics.StatusFailed
	}
	if report.Validation.Summary.Warnings > 0 || len(report.Execution.Hints) > 0 {
		if report.Status != diagnostics.StatusFailed {
			report.Status = diagnostics.StatusWarning
		}
	}
	report.Beginner = report.beginnerSummary()
	report.Professional = report.professionalDetails()
	return report, nil
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func (r Report) beginnerSummary() string {
	parts := []string{}
	if r.Scan.Summary.TotalEntries > 0 {
		parts = append(parts, r.Scan.BeginnerSummary())
	}
	if r.Validation.Summary.TotalFilesScanned > 0 {
		parts = append(parts, r.Validation.BeginnerSummary())
	}
	if r.Execution.Target.Path != "" {
		if r.Execution.Result.StartError != "" {
			parts = append(parts, r.Execution.Result.StartError)
		} else if r.Execution.Result.Mode != "" {
			parts = append(parts, r.Execution.Result.Mode)
		}
	}
	if len(parts) == 0 {
		return "preflight completed"
	}
	return strings.Join(parts, "; ")
}

func (r Report) professionalDetails() []string {
	details := append([]string(nil), r.Scan.ProfessionalDetails()...)
	for _, finding := range r.Validation.Findings {
		details = append(details, fmt.Sprintf("%s: %s", finding.RelativePath, strings.Join(finding.Messages, "; ")))
	}
	if r.Execution.Target.Path != "" {
		if r.Execution.Runtime.PTInterp != "" {
			details = append(details, "target interpreter: "+r.Execution.Runtime.PTInterp)
		}
		if r.Execution.TargetData.LikelySource != "" {
			details = append(details, "target source: "+r.Execution.TargetData.LikelySource)
		}
		if r.Execution.TargetData.HasPTInterp {
			details = append(details, "target has PT_INTERP")
		}
		details = append(details, r.Execution.Hints...)
	}
	return details
}
