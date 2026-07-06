package evidence

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/version"
)

const (
	SchemaVersion        = "1"
	defaultOutputDirName = ".acl/evidence"
)

var allowedCommands = map[string]struct{}{
	"git":         {},
	"arduino-cli": {},
}

type Runner interface {
	Run(context.Context, CommandSpec) CommandResult
}

type CommandSpec struct {
	Name        string
	Args        []string
	Allowlisted bool
	Mutating    bool
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

type RepositoryIdentity struct {
	Root        string `json:"root,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Commit      string `json:"commit,omitempty"`
	Status      string `json:"status,omitempty"`
	BranchState string `json:"branch_state,omitempty"`
}

type EnvironmentIdentity struct {
	WorkingDir   string            `json:"working_dir,omitempty"`
	Shell        string            `json:"shell,omitempty"`
	NativeTermux bool              `json:"native_termux"`
	TermuxPrefix string            `json:"termux_prefix,omitempty"`
	SelectedEnv  map[string]string `json:"selected_env,omitempty"`
	GOOS         string            `json:"goos,omitempty"`
	GOARCH       string            `json:"goarch,omitempty"`
}

type BinaryIdentity struct {
	Path       string    `json:"path,omitempty"`
	Version    string    `json:"version,omitempty"`
	Commit     string    `json:"commit,omitempty"`
	Date       string    `json:"date,omitempty"`
	Status     string    `json:"status,omitempty"`
	SizeBytes  int64     `json:"size_bytes,omitempty"`
	ModTimeUTC time.Time `json:"mod_time_utc,omitempty"`
	SHA256     string    `json:"sha256,omitempty"`
}

type CommandEvidence struct {
	Name                 string          `json:"name"`
	Args                 []string        `json:"args,omitempty"`
	Allowlisted          bool            `json:"allowlisted"`
	Mutating             bool            `json:"mutating"`
	StartedAtUTC         time.Time       `json:"started_at_utc,omitempty"`
	FinishedAtUTC        time.Time       `json:"finished_at_utc,omitempty"`
	DurationMS           int64           `json:"duration_ms,omitempty"`
	ExitCode             int             `json:"exit_code,omitempty"`
	Stdout               string          `json:"stdout,omitempty"`
	Stderr               string          `json:"stderr,omitempty"`
	NormalizedReport     json.RawMessage `json:"normalized_report,omitempty"`
	NormalizedReportType string          `json:"normalized_report_type,omitempty"`
	Warning              string          `json:"warning,omitempty"`
}

type EvidenceBundle struct {
	SchemaVersion  string                `json:"schema_version"`
	RunID          string                `json:"run_id"`
	CollectedAtUTC time.Time             `json:"collected_at_utc"`
	Repository     RepositoryIdentity    `json:"repository"`
	Environment    EnvironmentIdentity   `json:"environment"`
	Binary         BinaryIdentity        `json:"binary"`
	DevicePath     string                `json:"device_path"`
	Commands       []CommandEvidence     `json:"commands"`
	Status         acldiagnostics.Status `json:"status"`
	Warnings       []string              `json:"warnings,omitempty"`
	Limitations    []string              `json:"limitations,omitempty"`
	Summary        string                `json:"summary,omitempty"`
	NextStep       string                `json:"next_step,omitempty"`
	OutputPath     string                `json:"output_path,omitempty"`
}

func (b EvidenceBundle) JSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

type Collector struct {
	runner     Runner
	now        func() time.Time
	binaryPath string
}

type CollectOptions struct {
	DevicePath          string
	OutputDir           string
	IncludeStreamStatus bool
}

func NewCollector() *Collector {
	binPath := ""
	if exe, err := os.Executable(); err == nil {
		binPath = exe
	}
	return &Collector{
		runner:     execRunner{},
		now:        func() time.Time { return time.Now().UTC() },
		binaryPath: binPath,
	}
}

func NewCollectorWithRunner(r Runner) *Collector {
	c := NewCollector()
	if r != nil {
		c.runner = r
	}
	return c
}

func (c *Collector) Collect(ctx context.Context, opts CollectOptions) (EvidenceBundle, error) {
	if strings.TrimSpace(opts.DevicePath) == "" {
		return EvidenceBundle{}, errors.New("device path is required")
	}
	if c.runner == nil {
		c.runner = execRunner{}
	}
	if c.now == nil {
		c.now = func() time.Time { return time.Now().UTC() }
	}
	if strings.TrimSpace(c.binaryPath) == "" {
		if exe, err := os.Executable(); err == nil {
			c.binaryPath = exe
		}
	}

	runID := randomID()
	bundle := EvidenceBundle{
		SchemaVersion:  SchemaVersion,
		RunID:          runID,
		CollectedAtUTC: c.now(),
		Environment:    captureEnvironment(),
		Binary:         captureBinaryIdentity(c.binaryPath),
		DevicePath:     opts.DevicePath,
		Status:         acldiagnostics.StatusPassed,
		Warnings:       []string{},
		Limitations:    []string{},
		Summary:        "native-termux evidence collection completed",
		NextStep:       "review normalized ACL reports and raw traces before promoting findings",
	}

	repoCommands := []CommandSpec{
		{Allowlisted: true, Name: "git", Args: []string{"rev-parse", "--show-toplevel"}},
		{Allowlisted: true, Name: "git", Args: []string{"branch", "--show-current"}},
		{Allowlisted: true, Name: "git", Args: []string{"rev-parse", "HEAD"}},
		{Allowlisted: true, Name: "git", Args: []string{"status", "--short", "--branch"}},
	}
	aclCommands := []CommandSpec{
		{Allowlisted: true, Name: c.binaryPath, Args: []string{"acl", "transport", "list", "--json"}},
		{Allowlisted: true, Name: c.binaryPath, Args: []string{"acl", "transport", "diagnose", "--json", "--details", "--device", opts.DevicePath}},
		{Allowlisted: true, Name: c.binaryPath, Args: []string{"acl", "transport", "probe-fd", "--json", "--device", opts.DevicePath}},
	}
	if opts.IncludeStreamStatus {
		aclCommands = append(aclCommands, CommandSpec{Allowlisted: true, Name: c.binaryPath, Args: []string{"acl", "transport", "stream-status", "--json", "--device", opts.DevicePath}})
	}
	plan := append(repoCommands, aclCommands...)
	if err := validatePlan(plan); err != nil {
		return EvidenceBundle{}, err
	}

	for _, spec := range plan {
		startedAt := c.now()
		result := c.runner.Run(ctx, spec)
		finishedAt := c.now()
		entry := evidenceFromResult(spec, result, startedAt, finishedAt)
		bundle.Commands = append(bundle.Commands, entry)
		applyEvidenceToBundle(&bundle, spec, entry)
		if inferred := inferBundleStatus(entry); inferred.Severity() > bundle.Status.Severity() {
			bundle.Status = inferred
		}
		if result.Err != nil || result.ExitCode != 0 {
			if acldiagnostics.StatusWarning.Severity() > bundle.Status.Severity() {
				bundle.Status = acldiagnostics.StatusWarning
			}
			if result.Err != nil {
				bundle.Warnings = append(bundle.Warnings, fmt.Sprintf("%s: %v", spec.Name, result.Err))
			} else {
				bundle.Warnings = append(bundle.Warnings, fmt.Sprintf("%s exited with code %d", spec.Name, result.ExitCode))
			}
		}
	}

	repoRoot := deriveRepoRoot(bundle.Commands)
	if repoRoot != "" {
		bundle.Repository.Root = repoRoot
	}
	if bundle.Repository.Root == "" {
		if wd, err := os.Getwd(); err == nil {
			bundle.Repository.Root = wd
		}
	}

	outputDir := strings.TrimSpace(opts.OutputDir)
	if outputDir == "" {
		outputDir = filepath.Join(bundle.Repository.Root, defaultOutputDirName)
	} else if !filepath.IsAbs(outputDir) && bundle.Repository.Root != "" {
		outputDir = filepath.Join(bundle.Repository.Root, outputDir)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return EvidenceBundle{}, err
	}
	outputPath := filepath.Join(outputDir, fmt.Sprintf("evidence-%s-%s.json", time.Now().UTC().Format("20060102T150405Z"), runID))
	bundle.OutputPath = outputPath
	if err := writeBundle(outputPath, bundle); err != nil {
		return EvidenceBundle{}, err
	}
	return bundle, nil
}

func inferBundleStatus(entry CommandEvidence) acldiagnostics.Status {
	if len(entry.NormalizedReport) == 0 {
		return acldiagnostics.StatusPending
	}
	var payload map[string]any
	if err := json.Unmarshal(entry.NormalizedReport, &payload); err != nil {
		return acldiagnostics.StatusPending
	}
	statusValue, ok := payload["status"]
	if !ok {
		return acldiagnostics.StatusPending
	}
	status, ok := statusValue.(string)
	if !ok {
		return acldiagnostics.StatusPending
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(acldiagnostics.StatusFailed):
		return acldiagnostics.StatusFailed
	case string(acldiagnostics.StatusWarning):
		return acldiagnostics.StatusWarning
	case string(acldiagnostics.StatusSkipped):
		return acldiagnostics.StatusSkipped
	case string(acldiagnostics.StatusPassed):
		return acldiagnostics.StatusPassed
	default:
		return acldiagnostics.StatusPending
	}
}

func validatePlan(plan []CommandSpec) error {
	for _, spec := range plan {
		if strings.TrimSpace(spec.Name) == "" {
			return errors.New("command name is required")
		}
		if _, ok := allowedCommands[filepath.Base(spec.Name)]; !ok {
			return fmt.Errorf("command %q is not allowlisted", spec.Name)
		}
		if spec.Mutating {
			return fmt.Errorf("command %q is mutating and not allowed", spec.Name)
		}
	}
	return nil
}

func evidenceFromResult(spec CommandSpec, result CommandResult, startedAt, finishedAt time.Time) CommandEvidence {
	duration := time.Duration(0)
	if !startedAt.IsZero() && !finishedAt.IsZero() {
		duration = finishedAt.Sub(startedAt)
	}
	entry := CommandEvidence{
		Name:          spec.Name,
		Args:          append([]string(nil), spec.Args...),
		Allowlisted:   spec.Allowlisted,
		Mutating:      spec.Mutating,
		StartedAtUTC:  startedAt,
		FinishedAtUTC: finishedAt,
		DurationMS:    duration.Milliseconds(),
		ExitCode:      result.ExitCode,
		Stdout:        result.Stdout,
		Stderr:        result.Stderr,
	}
	if result.Err != nil {
		entry.Warning = result.Err.Error()
	}
	if trimmed := strings.TrimSpace(result.Stdout); trimmed != "" && json.Valid([]byte(trimmed)) {
		entry.NormalizedReport = json.RawMessage(trimmed)
		entry.NormalizedReportType = "json"
	}
	if entry.NormalizedReportType == "" && result.ExitCode == 0 && result.Err == nil {
		entry.NormalizedReportType = chooseNormalizedReportType(spec)
	}
	return entry
}

func chooseNormalizedReportType(spec CommandSpec) string {
	if len(spec.Args) == 0 {
		return ""
	}
	for _, arg := range spec.Args {
		if strings.TrimSpace(arg) == "--json" {
			return "acl-json"
		}
	}
	return ""
}

func applyEvidenceToBundle(bundle *EvidenceBundle, spec CommandSpec, entry CommandEvidence) {
	switch {
	case spec.Name == "git" && len(spec.Args) > 0 && spec.Args[0] == "rev-parse" && len(spec.Args) > 1 && spec.Args[1] == "--show-toplevel":
		if root := strings.TrimSpace(entry.Stdout); root != "" {
			bundle.Repository.Root = strings.TrimSpace(root)
		}
	case spec.Name == "git" && len(spec.Args) > 0 && spec.Args[0] == "branch" && len(spec.Args) > 1 && spec.Args[1] == "--show-current":
		bundle.Repository.Branch = strings.TrimSpace(entry.Stdout)
	case spec.Name == "git" && len(spec.Args) > 0 && spec.Args[0] == "rev-parse" && len(spec.Args) > 1 && spec.Args[1] == "HEAD":
		bundle.Repository.Commit = strings.TrimSpace(entry.Stdout)
	case spec.Name == "git" && len(spec.Args) > 0 && spec.Args[0] == "status" && len(spec.Args) > 1 && spec.Args[1] == "--short":
		bundle.Repository.Status = strings.TrimSpace(entry.Stdout)
	}
}

func deriveRepoRoot(commands []CommandEvidence) string {
	for _, cmd := range commands {
		if cmd.Name == "git" && len(cmd.Args) > 1 && cmd.Args[0] == "rev-parse" && cmd.Args[1] == "--show-toplevel" {
			return strings.TrimSpace(cmd.Stdout)
		}
	}
	return ""
}

func captureEnvironment() EnvironmentIdentity {
	selected := map[string]string{}
	for _, key := range []string{
		"PREFIX",
		"PATH",
		"HOME",
		"PWD",
		"SHELL",
		"TERMUX_VERSION",
		"TERMUX__VERSION",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			selected[key] = value
		}
	}
	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell == "" {
		shell = runtime.GOOS + "/" + runtime.GOARCH
	}
	prefix := strings.TrimSpace(os.Getenv("PREFIX"))
	nativeTermux := prefix != "" || strings.TrimSpace(os.Getenv("TERMUX_VERSION")) != ""
	return EnvironmentIdentity{
		WorkingDir:   mustGetwd(),
		Shell:        shell,
		NativeTermux: nativeTermux,
		TermuxPrefix: prefix,
		SelectedEnv:  selected,
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
	}
}

func captureBinaryIdentity(path string) BinaryIdentity {
	exe := strings.TrimSpace(path)
	if exe == "" {
		exe, _ = os.Executable()
		exe = strings.TrimSpace(exe)
	}
	info := version.VersionInfo
	identity := BinaryIdentity{
		Path:       exe,
		SizeBytes:  0,
		ModTimeUTC: time.Time{},
	}
	if info != nil {
		identity.Version = info.VersionString
		identity.Commit = info.Commit
		identity.Date = info.Date
		identity.Status = info.Status
	}
	if stat, err := os.Stat(exe); err == nil {
		identity.SizeBytes = stat.Size()
		identity.ModTimeUTC = stat.ModTime().UTC()
		if sum, err := fileSHA256(exe); err == nil {
			identity.SHA256 = sum
		}
	}
	return identity
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func randomID() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}

func writeBundle(path string, bundle EvidenceBundle) error {
	data, err := bundle.JSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, spec CommandSpec) CommandResult {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	var stdoutBuf strings.Builder
	var stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	result := CommandResult{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}
	if err != nil {
		result.Err = err
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result
	}
	result.ExitCode = 0
	return result
}
