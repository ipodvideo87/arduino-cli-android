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
		checkRequiredDocumentsExist(repoRoot),
		checkReadmeOverviewOnly(repoRoot),
		checkRequiredRouting(repoRoot),
		checkStatusStructure(repoRoot),
		checkRoadmapStructure(repoRoot),
		checkRoadmapFutureItems(repoRoot),
		checkTaskRecoveryState(repoRoot),
		checkHistoricalClassification(repoRoot),
		checkStalePathHygiene(repoRoot),
		checkWorkflowIntentReview(repoRoot),
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

func checkRequiredDocumentsExist(repoRoot string) CheckResult {
	check := CheckResult{Name: "required documents exist"}
	required := []string{
		"AGENTS.md",
		"STATUS.md",
		"docs/android/ROADMAP.md",
		"docs/android/DEVELOPMENT_WORKFLOW.md",
		"docs/android/DOCUMENTATION_ARCHITECTURE.md",
		"docs/android/ENGINEERING_MILESTONE_SUMMARY.md",
		"docs/android/QUEUED_BRANCH_REVIEW.md",
		"docs/android/CLOSEOUT_REPORTING_STANDARD.md",
		"docs/android/ENGINEERING_JUDGMENT_STANDARD.md",
		"docs/android/GOVERNANCE_COVERAGE_MATRIX.md",
		"docs/android/ENGINEERING_DEBT_REGISTER.md",
		"docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md",
		"docs/android/TASK_RECOVERY.md",
	}
	for _, relPath := range required {
		if _, err := os.Stat(filepath.Join(repoRoot, relPath)); err != nil {
			check.Messages = append(check.Messages, fmt.Sprintf("required document %s is missing: %v", relPath, err))
		}
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
	return check
}

func checkRequiredRouting(repoRoot string) CheckResult {
	check := CheckResult{Name: "required routing"}
	agents, err := readFile(repoRoot, "AGENTS.md")
	if err != nil {
		check.Messages = append(check.Messages, err.Error())
		return check
	}
	workflow, err := readFile(repoRoot, "docs/android/DEVELOPMENT_WORKFLOW.md")
	if err != nil {
		check.Messages = append(check.Messages, err.Error())
		return check
	}
	requiredPhrases := []struct {
		content string
		path    string
		want    []string
	}{
		{
			content: agents,
			path:    "AGENTS.md",
			want: []string{
				"docs/android/DEVELOPMENT_WORKFLOW.md",
				"operational front door",
			},
		},
		{
			content: workflow,
			path:    "docs/android/DEVELOPMENT_WORKFLOW.md",
			want: []string{
				"docs/android/CLOSEOUT_REPORTING_STANDARD.md",
				"docs/android/ENGINEERING_JUDGMENT_STANDARD.md",
				"docs/android/ENGINEERING_KNOWLEDGE_FRAMEWORK.md",
			},
		},
	}
	for _, req := range requiredPhrases {
		for _, want := range req.want {
			if !strings.Contains(req.content, want) {
				check.Messages = append(check.Messages, fmt.Sprintf("%s is missing required routing phrase %q", req.path, want))
			}
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
	if !strings.Contains(content, "This document is the current snapshot of project progress.") {
		check.Messages = append(check.Messages, "STATUS.md is missing current-state authority language")
	}
	if !strings.Contains(content, "STATUS.md is authoritative for the current snapshot") {
		check.Messages = append(check.Messages, "STATUS.md is missing current-snapshot ownership language")
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
	if !strings.Contains(content, "docs/android/ROADMAP.md is authoritative for future ordering") {
		check.Messages = append(check.Messages, "docs/android/ROADMAP.md is missing future-ordering ownership language")
	}
	if strings.Contains(content, "current snapshot") {
		check.Messages = append(check.Messages, "docs/android/ROADMAP.md presents current-state language that belongs in STATUS.md")
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
		!strings.Contains(content, "Objective:") &&
		strings.Contains(content, "This file is the live recovery snapshot")
	for _, heading := range idleHeadings {
		if !containsExactLine(content, heading) {
			idle = false
			check.Messages = append(check.Messages, fmt.Sprintf("TASK_RECOVERY.md is missing required idle-template heading %q", heading))
		}
	}
	active := strings.Contains(content, "Objective:") &&
		strings.Contains(content, "## Intended Plan") &&
		!strings.Contains(content, "Status: idle") &&
		strings.Contains(content, "This file is the live recovery snapshot")
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

func checkHistoricalClassification(repoRoot string) CheckResult {
	check := CheckResult{Name: "historical classification"}
	historicalDocs := []string{
		"docs/android/ENGINEERING_MILESTONE_SUMMARY.md",
		"docs/android/QUEUED_BRANCH_REVIEW.md",
	}
	for _, relPath := range historicalDocs {
		content, err := readFile(repoRoot, relPath)
		if err != nil {
			check.Messages = append(check.Messages, err.Error())
			continue
		}
		requiredPhrases := []string{
			"## Historical Classification",
			"This document is historical evidence.",
			"It is not authoritative for current status.",
		}
		for _, want := range requiredPhrases {
			if !strings.Contains(content, want) {
				check.Messages = append(check.Messages, fmt.Sprintf("%s is missing historical classification phrase %q", relPath, want))
			}
		}
		if !strings.Contains(content, "For the current snapshot, see [STATUS.md](") {
			check.Messages = append(check.Messages, fmt.Sprintf("%s is missing the explicit STATUS.md route for the current snapshot", relPath))
		}
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
	return check
}

func checkStalePathHygiene(repoRoot string) CheckResult {
	check := CheckResult{Name: "stale path hygiene"}
	currentStateDocs := []string{
		"STATUS.md",
		"docs/android/ROADMAP.md",
		"docs/android/DEVELOPMENT_WORKFLOW.md",
	}
	stalePath := "/root/arduino-cli-android"
	for _, relPath := range currentStateDocs {
		content, err := readFile(repoRoot, relPath)
		if err != nil {
			check.Messages = append(check.Messages, err.Error())
			continue
		}
		if strings.Contains(content, stalePath) {
			check.Messages = append(check.Messages, fmt.Sprintf("%s still presents %s as the active canonical repository", relPath, stalePath))
		}
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
	return check
}

func checkWorkflowIntentReview(repoRoot string) CheckResult {
	check := CheckResult{Name: "workflow intent review"}
	content, err := readFile(repoRoot, "docs/android/DEVELOPMENT_WORKFLOW.md")
	if err != nil {
		check.Messages = append(check.Messages, err.Error())
		return check
	}
	required := []string{
		"## Canonical Document Change Review",
		"Before editing an important canonical document:",
		"After editing an important canonical document:",
		"current-state synchronization",
		"historical/current classification result",
		"final compliance judgment",
	}
	for _, want := range required {
		if !strings.Contains(content, want) {
			check.Messages = append(check.Messages, fmt.Sprintf("docs/android/DEVELOPMENT_WORKFLOW.md is missing required intent-review content %q", want))
		}
	}
	if len(check.Messages) == 0 {
		check.Passed = true
	}
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
