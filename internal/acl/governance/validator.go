package governance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
)

var CanonicalRepoRoot = "/data/data/com.termux/files/home/Development/GitHub/arduino-cli-android"

type Options struct {
	RepoRoot         string
	ExpectedRepoRoot string
	WorkingDir       string
}

type CheckResult struct {
	Name     string   `json:"name"`
	Passed   bool     `json:"passed"`
	Messages []string `json:"messages,omitempty"`
}

type Report struct {
	RepoRoot         string                `json:"repo_root"`
	ExpectedRepoRoot string                `json:"expected_repo_root,omitempty"`
	WorkingDir       string                `json:"working_dir"`
	Status           acldiagnostics.Status `json:"status"`
	Checks           []CheckResult         `json:"checks"`
	FailureCount     int                   `json:"failure_count"`
	Summary          string                `json:"summary,omitempty"`
}

func (r Report) HasFailures() bool {
	return r.FailureCount > 0
}

func (r Report) FailedChecks() []CheckResult {
	var failed []CheckResult
	for _, check := range r.Checks {
		if !check.Passed {
			failed = append(failed, check)
		}
	}
	return failed
}

func Validate(opts Options) Report {
	repoRoot := normalizePath(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = CanonicalRepoRoot
	}
	expectedRepoRoot := normalizePath(opts.ExpectedRepoRoot)
	if expectedRepoRoot == "" {
		expectedRepoRoot = CanonicalRepoRoot
	}
	workingDir := normalizePath(opts.WorkingDir)
	if workingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			workingDir = normalizePath(wd)
		}
	}

	report := Report{
		RepoRoot:         repoRoot,
		ExpectedRepoRoot: expectedRepoRoot,
		WorkingDir:       workingDir,
		Status:           acldiagnostics.StatusPassed,
	}

	checks := []CheckResult{
		checkCanonicalRepoRoot(expectedRepoRoot, workingDir),
		checkReadmeOverviewOnly(repoRoot),
		checkStatusStructure(repoRoot),
		checkRoadmapStructure(repoRoot),
		checkRoadmapFutureItems(repoRoot),
		checkTaskRecoveryState(repoRoot),
		checkStalePhrases(repoRoot),
	}

	for _, check := range checks {
		if !check.Passed {
			report.FailureCount++
			report.Status = acldiagnostics.StatusFailed
		}
		report.Checks = append(report.Checks, check)
	}
	if report.FailureCount == 0 {
		report.Summary = "governance validation passed"
	} else {
		report.Summary = fmt.Sprintf("governance validation found %d issue(s)", report.FailureCount)
	}
	return report
}

func checkCanonicalRepoRoot(expectedRepoRoot, workingDir string) CheckResult {
	check := CheckResult{Name: "canonical repo root"}
	if workingDir == "" {
		check.Messages = append(check.Messages, "working directory could not be determined")
		return check
	}
	if expectedRepoRoot != CanonicalRepoRoot {
		check.Messages = append(check.Messages, fmt.Sprintf("expected canonical repo root is %q, required %q", expectedRepoRoot, CanonicalRepoRoot))
		return check
	}
	if workingDir != expectedRepoRoot {
		check.Messages = append(check.Messages, fmt.Sprintf("working directory is %q, expected canonical repo root %q", workingDir, expectedRepoRoot))
		return check
	}
	check.Passed = true
	return check
}

func checkReadmeOverviewOnly(repoRoot string) CheckResult {
	check := CheckResult{Name: "README overview-only"}
	content, err := readFile(repoRoot, "README.md")
	if err != nil {
		check.Messages = append(check.Messages, err.Error())
		return check
	}
	for _, heading := range []string{
		"## Current Validated State",
		"## What Is Still Experimental",
		"## Current Focus",
		"## Recent Work",
		"## Future Work",
	} {
		if containsExactLine(content, heading) {
			check.Messages = append(check.Messages, fmt.Sprintf("README.md contains forbidden status-style heading %q", heading))
		}
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
	return check
}

func checkStatusStructure(repoRoot string) CheckResult {
	check := CheckResult{Name: "STATUS structure"}
	content, err := readFile(repoRoot, "STATUS.md")
	if err != nil {
		check.Messages = append(check.Messages, err.Error())
		return check
	}
	requiredHeadings := []string{
		"## Current Mission",
		"## STATUS Versus ROADMAP",
		"## Next Engineering Milestone",
	}
	for _, heading := range requiredHeadings {
		if !containsExactLine(content, heading) {
			check.Messages = append(check.Messages, fmt.Sprintf("STATUS.md is missing required heading %q", heading))
		}
	}
	if strings.Contains(content, "target chip metadata is not set") {
		check.Messages = append(check.Messages, "STATUS.md still contains resolved target-chip unresolved wording")
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
	return check
}

func checkRoadmapStructure(repoRoot string) CheckResult {
	check := CheckResult{Name: "ROADMAP structure"}
	content, err := readFile(repoRoot, "docs/android/ROADMAP.md")
	if err != nil {
		check.Messages = append(check.Messages, err.Error())
		return check
	}
	if !containsExactLine(content, "## Next Milestones") {
		check.Messages = append(check.Messages, "docs/android/ROADMAP.md is missing required heading \"## Next Milestones\"")
	}
	if !containsExactLine(content, "## STATUS Sync") {
		check.Messages = append(check.Messages, "docs/android/ROADMAP.md is missing required heading \"## STATUS Sync\"")
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
	return check
}

func checkRoadmapFutureItems(repoRoot string) CheckResult {
	check := CheckResult{Name: "ROADMAP future items"}
	content, err := readFile(repoRoot, "docs/android/ROADMAP.md")
	if err != nil {
		check.Messages = append(check.Messages, err.Error())
		return check
	}
	section := sectionAfterHeading(content, "## Next Milestones")
	if section == "" {
		if containsExactLine(content, "## Next Milestones") {
			check.Passed = true
			return check
		}
		check.Messages = append(check.Messages, "unable to read the Next Milestones section")
		return check
	}
	staleItems := []string{
		"Validate the Termux USB provider and file-descriptor handoff path on native Termux",
		"Validate the TERMUX_USB_FD probe surface on native Termux",
		"Validate the transport stream foundation on native Termux",
		"Validate upload prepare-only planning on native Termux and keep the CLI/report contract explicit",
		"Native full-flash bootloader package validation on native Termux",
	}
	for _, item := range staleItems {
		if strings.Contains(section, item) {
			check.Messages = append(check.Messages, fmt.Sprintf("Next Milestones still contains completed item %q", item))
		}
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
	return check
}

func checkTaskRecoveryState(repoRoot string) CheckResult {
	check := CheckResult{Name: "TASK_RECOVERY state"}
	content, err := readFile(repoRoot, "docs/android/TASK_RECOVERY.md")
	if err != nil {
		check.Messages = append(check.Messages, err.Error())
		return check
	}
	idleHeadings := []string{
		"## Active Task",
		"## Intended Plan",
		"## Progress",
		"## Files",
		"## Validation",
		"## Safest Next Action",
		"## Canonical Follow-Through",
		"## Reset State",
	}
	idle := strings.Contains(content, "Status: idle") &&
		strings.Contains(content, "- none") &&
		!strings.Contains(content, "Objective:")
	for _, heading := range idleHeadings {
		if !containsExactLine(content, heading) {
			idle = false
			check.Messages = append(check.Messages, fmt.Sprintf("TASK_RECOVERY.md is missing required idle-template heading %q", heading))
		}
	}
	active := strings.Contains(content, "Objective:") &&
		strings.Contains(content, "## Intended Plan") &&
		!strings.Contains(content, "Status: idle")
	if idle || active {
		check.Messages = nil
		check.Passed = true
		return check
	}
	if strings.Contains(content, "Objective:") && strings.Contains(content, "Status: idle") {
		check.Messages = append(check.Messages, "TASK_RECOVERY.md mixes idle state with active task content")
		return check
	}
	check.Messages = append(check.Messages, "TASK_RECOVERY.md is neither idle nor explicitly active")
	return check
}

func checkStalePhrases(repoRoot string) CheckResult {
	check := CheckResult{Name: "stale phrases"}
	paths := []string{"STATUS.md", "docs/android/ROADMAP.md", "docs/android/DEVELOPMENT_WORKFLOW.md", "README.md", "docs/android/TASK_RECOVERY.md"}
	stalePhrases := []string{
		"target chip metadata is not set",
		"remaining target-chip metadata warning",
	}
	for _, path := range paths {
		content, err := readFile(repoRoot, path)
		if err != nil {
			check.Messages = append(check.Messages, err.Error())
			continue
		}
		for _, phrase := range stalePhrases {
			if strings.Contains(content, phrase) {
				check.Messages = append(check.Messages, fmt.Sprintf("%s still contains stale phrase %q", path, phrase))
			}
		}
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
	return check
}

func readFile(repoRoot, relPath string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", errors.New("repository root is required")
	}
	path := filepath.Join(repoRoot, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", relPath, err)
	}
	return string(data), nil
}

func containsExactLine(content, want string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

func sectionAfterHeading(content, heading string) string {
	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return ""
	}
	var section []string
	for _, line := range lines[start:] {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			break
		}
		section = append(section, line)
	}
	return strings.Join(section, "\n")
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.Clean(trimmed)
}
