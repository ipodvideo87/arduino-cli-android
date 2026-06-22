package runtime

import (
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	StatusPass         = "PASS"
	StatusWarn         = "WARN"
	StatusFail         = "FAIL"
	StatusExperimental = "EXPERIMENTAL"
)

type Manager struct {
	root string
	now  func() time.Time
}

type Runtime struct {
	ID         string
	Path       string
	Manifest   Manifest
	Active     bool
	Validated  bool
	LastReport ValidationReport
}

type StatusReport struct {
	Root             string
	ActiveRuntimeID  string
	ActiveRuntimeDir string
	Runtimes         []RuntimeSummary
}

type RuntimeSummary struct {
	ID                 string
	RuntimeVersion     string
	Architecture       string
	SupportedABIs      []string
	CompatibilityLevel string
	Path               string
	Active             bool
	Valid              bool
	Status             string
	CreatedAt          string
}

type SelectionRequest struct {
	Architecture string
	ABI          string
}

type ValidationReport struct {
	RuntimeID string
	Path      string
	Status    string
	Checks    []CheckResult
}

type CheckResult struct {
	Name    string
	Status  string
	Message string
	Path    string
}

type elfInspection struct {
	Exists      bool
	IsELF       bool
	Machine     string
	SONAME      string
	Interpreter string
	RPath       string
	RunPath     string
	Needed      []string
}

func NewManager(root string) *Manager {
	return &Manager{
		root: root,
		now:  time.Now,
	}
}

func DefaultRoot() (string, error) {
	candidates, err := defaultRootCandidates()
	if err != nil {
		return "", err
	}

	first := ""
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if first == "" {
			first = candidate
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	if first != "" {
		return first, nil
	}

	return "", errors.New("unable to determine ACL runtime root")
}

func (m *Manager) Root() (string, error) {
	if strings.TrimSpace(m.root) != "" {
		return m.root, nil
	}
	return DefaultRoot()
}

func defaultRootCandidates() ([]string, error) {
	candidates := make([]string, 0, 4)
	if root := strings.TrimSpace(os.Getenv("ACL_RUNTIME_ROOT")); root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve ACL_RUNTIME_ROOT %q: %w", root, err)
		}
		candidates = append(candidates, abs)
	}

	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		candidates = append(candidates, filepath.Join(home, ".arduino-cli-android", "acl-runtime"))
	}

	if prefix := strings.TrimSpace(os.Getenv("PREFIX")); prefix != "" {
		candidates = append(candidates, filepath.Join(prefix, "opt", "arduino-cli-android", "acl-runtime"))
	}

	if wd, err := os.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
		candidates = append(candidates, filepath.Join(wd, "acl-runtime"))
	}

	return candidates, nil
}

func (m *Manager) runtimesDir() (string, error) {
	root, err := m.Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "runtimes"), nil
}

func (m *Manager) activeFile() (string, error) {
	root, err := m.Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "active.json"), nil
}

func (m *Manager) ensureRoot() error {
	root, err := m.Root()
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(root, "runtimes"), 0o755)
}

func (m *Manager) Discover() ([]Runtime, error) {
	runtimesDir, err := m.runtimesDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(runtimesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	activeID, _ := m.ActiveRuntimeID()
	runtimes := make([]Runtime, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		rt, err := m.loadRuntime(filepath.Join(runtimesDir, entry.Name()))
		if err != nil {
			runtimes = append(runtimes, Runtime{
				ID:   entry.Name(),
				Path: filepath.Join(runtimesDir, entry.Name()),
				LastReport: ValidationReport{
					RuntimeID: entry.Name(),
					Path:      filepath.Join(runtimesDir, entry.Name()),
					Status:    StatusFail,
					Checks: []CheckResult{
						{Name: "manifest", Status: StatusFail, Message: err.Error()},
					},
				},
			})
			continue
		}
		rt.Active = activeID != "" && activeID == rt.ID
		if report, err := m.ValidateRuntime(rt); err == nil {
			rt.Validated = report.Status != StatusFail
			rt.LastReport = report
		}
		runtimes = append(runtimes, rt)
	}

	sort.Slice(runtimes, func(i, j int) bool { return runtimes[i].ID < runtimes[j].ID })
	return runtimes, nil
}

func (m *Manager) loadRuntime(path string) (Runtime, error) {
	manifestPath := filepath.Join(path, ManifestFileName)
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return Runtime{}, err
	}
	if err := manifest.ValidateBasic(); err != nil {
		return Runtime{}, err
	}

	if filepath.Base(path) != manifest.RuntimeID {
		return Runtime{}, fmt.Errorf("runtime directory %q does not match manifest runtime_id %q", filepath.Base(path), manifest.RuntimeID)
	}

	return Runtime{
		ID:       manifest.RuntimeID,
		Path:     path,
		Manifest: manifest,
	}, nil
}

func (m *Manager) Validate(id string) (ValidationReport, error) {
	rt, err := m.Load(id)
	if err != nil {
		return ValidationReport{}, err
	}
	return m.ValidateRuntime(rt)
}

func (m *Manager) Load(id string) (Runtime, error) {
	runtimesDir, err := m.runtimesDir()
	if err != nil {
		return Runtime{}, err
	}
	return m.loadRuntime(filepath.Join(runtimesDir, id))
}

func (m *Manager) ValidateRuntime(rt Runtime) (ValidationReport, error) {
	report := ValidationReport{
		RuntimeID: rt.ID,
		Path:      rt.Path,
		Status:    compatibilityStatus(rt.Manifest.CompatibilityLevel),
	}

	add := func(name, status, message, path string) {
		report.Checks = append(report.Checks, CheckResult{
			Name:    name,
			Status:  status,
			Message: message,
			Path:    path,
		})
		switch status {
		case StatusFail:
			report.Status = StatusFail
		case StatusWarn:
			if report.Status == StatusPass {
				report.Status = StatusWarn
			}
		case StatusExperimental:
			if report.Status == StatusPass {
				report.Status = StatusExperimental
			}
		}
	}

	if err := rt.Manifest.ValidateBasic(); err != nil {
		add("manifest", StatusFail, err.Error(), filepath.Join(rt.Path, ManifestFileName))
		return report, nil
	}
	add("manifest", StatusPass, "manifest parsed", filepath.Join(rt.Path, ManifestFileName))

	files := rt.Manifest.AllFiles()
	for _, file := range files {
		path := filepath.Join(rt.Path, file.Path)
		info, err := os.Lstat(path)
		if err != nil {
			add(file.Name, StatusFail, fmt.Sprintf("missing: %v", err), path)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			add(file.Name, StatusWarn, "symbolic link present", path)
		}

		inspected, err := inspectELF(path)
		if err != nil {
			add(file.Name, StatusFail, err.Error(), path)
			continue
		}
		if !inspected.IsELF {
			add(file.Name, StatusFail, "file is not ELF", path)
			continue
		}
		add(file.Name, StatusPass, fmt.Sprintf("ELF %s", inspected.Machine), path)

		expectedArch := strings.TrimSpace(rt.Manifest.Architecture)
		if expectedArch != "" && !architectureMatches(expectedArch, inspected.Machine) {
			add(file.Name+":arch", StatusFail, fmt.Sprintf("expected %s, got %s", expectedArch, inspected.Machine), path)
		}

		if file.SONAME != "" && inspected.SONAME != "" && file.SONAME != inspected.SONAME {
			add(file.Name+":soname", StatusFail, fmt.Sprintf("expected %s, got %s", file.SONAME, inspected.SONAME), path)
		}

		if file.SHA256 != "" {
			sum, err := sha256Hex(path)
			if err != nil {
				add(file.Name+":sha256", StatusFail, err.Error(), path)
			} else if !strings.EqualFold(sum, file.SHA256) {
				add(file.Name+":sha256", StatusFail, fmt.Sprintf("expected %s, got %s", file.SHA256, sum), path)
			} else {
				add(file.Name+":sha256", StatusPass, "hash verified", path)
			}
		}
	}

	if len(rt.Manifest.SupportedABIs) == 0 {
		add("supported_abis", StatusFail, "at least one ABI is required", rt.Path)
	}

	return report, nil
}

func (m *Manager) Status() (StatusReport, error) {
	runtimes, err := m.Discover()
	if err != nil {
		return StatusReport{}, err
	}

	root, err := m.Root()
	if err != nil {
		return StatusReport{}, err
	}

	activeID, _ := m.ActiveRuntimeID()
	report := StatusReport{Root: root, ActiveRuntimeID: activeID}
	if activeID != "" {
		report.ActiveRuntimeDir = filepath.Join(root, "runtimes", activeID)
	}

	for _, rt := range runtimes {
		summary := RuntimeSummary{
			ID:                 rt.ID,
			RuntimeVersion:     rt.Manifest.RuntimeVersion,
			Architecture:       rt.Manifest.Architecture,
			SupportedABIs:      append([]string(nil), rt.Manifest.SupportedABIs...),
			CompatibilityLevel: rt.Manifest.CompatibilityLevel,
			Path:               rt.Path,
			Active:             rt.Active,
			Valid:              rt.Validated && rt.LastReport.Status != StatusFail,
			Status:             rt.LastReport.Status,
			CreatedAt:          rt.Manifest.CreatedAt,
		}
		report.Runtimes = append(report.Runtimes, summary)
	}
	return report, nil
}

func (m *Manager) SelectCompatible(req SelectionRequest) (Runtime, error) {
	runtimes, err := m.Discover()
	if err != nil {
		return Runtime{}, err
	}

	var best *Runtime
	bestScore := int64(-1)
	for i := range runtimes {
		rt := runtimes[i]
		if rt.LastReport.Status == StatusFail {
			continue
		}
		if req.Architecture != "" && !architectureMatches(req.Architecture, rt.Manifest.Architecture) {
			continue
		}
		if req.ABI != "" && !containsString(rt.Manifest.SupportedABIs, req.ABI) {
			continue
		}

		score := compatibilityScore(rt.Manifest.CompatibilityLevel)*1_000_000_000_000 + creationScore(rt.Manifest.CreatedAt)
		if score > bestScore {
			best = &runtimes[i]
			bestScore = score
		}
	}

	if best == nil {
		return Runtime{}, fmt.Errorf("no compatible runtime found for architecture=%q abi=%q", req.Architecture, req.ABI)
	}
	return *best, nil
}

func (m *Manager) InstallFromDir(sourceDir string) (Runtime, error) {
	if err := m.ensureRoot(); err != nil {
		return Runtime{}, err
	}

	manifest, err := LoadManifest(filepath.Join(sourceDir, ManifestFileName))
	if err != nil {
		return Runtime{}, err
	}
	if err := manifest.ValidateBasic(); err != nil {
		return Runtime{}, err
	}

	targetDir := filepath.Join(m.mustRuntimesDir(), manifest.RuntimeID)
	if _, err := os.Stat(targetDir); err == nil {
		return Runtime{}, fmt.Errorf("runtime %q already exists", manifest.RuntimeID)
	}

	if err := copyTree(sourceDir, targetDir); err != nil {
		_ = os.RemoveAll(targetDir)
		return Runtime{}, err
	}

	rt, err := m.loadRuntime(targetDir)
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return Runtime{}, err
	}
	report, err := m.ValidateRuntime(rt)
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return Runtime{}, err
	}
	rt.Validated = report.Status != StatusFail
	rt.LastReport = report
	if report.Status == StatusFail {
		_ = os.RemoveAll(targetDir)
		return Runtime{}, fmt.Errorf("installed runtime %q failed validation: %s", manifest.RuntimeID, validationFailureDetail(report))
	}
	return rt, nil
}

func (m *Manager) Activate(id string) error {
	rt, err := m.Load(id)
	if err != nil {
		return err
	}
	report, err := m.ValidateRuntime(rt)
	if err != nil {
		return err
	}
	if report.Status == StatusFail {
		return fmt.Errorf("runtime %q failed validation and cannot be activated", id)
	}

	active := activeSelection{
		RuntimeID:  id,
		SelectedAt: m.now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(active, "", "  ")
	if err != nil {
		return err
	}
	file, err := m.activeFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644)
}

func (m *Manager) Deactivate() error {
	file, err := m.activeFile()
	if err != nil {
		return err
	}
	if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *Manager) ActiveRuntimeID() (string, error) {
	file, err := m.activeFile()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var active activeSelection
	if err := json.Unmarshal(data, &active); err != nil {
		return "", fmt.Errorf("decode active runtime: %w", err)
	}
	return active.RuntimeID, nil
}

func (m *Manager) mustRuntimesDir() string {
	dir, _ := m.runtimesDir()
	return dir
}

type activeSelection struct {
	RuntimeID  string `json:"runtime_id"`
	SelectedAt string `json:"selected_at"`
}

func inspectELF(path string) (elfInspection, error) {
	var result elfInspection
	info, err := os.Stat(path)
	if err != nil {
		return result, err
	}
	result.Exists = info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0

	f, err := elf.Open(path)
	if err != nil {
		if errors.Is(err, elf.ErrNoSymbols) {
			return result, nil
		}
		return result, fmt.Errorf("ELF open: %w", err)
	}
	defer f.Close()

	result.IsELF = true
	result.Machine = archFromMachine(f.FileHeader.Machine)
	result.SONAME = readDynamicString(f, elf.DT_SONAME)
	result.Interpreter = readInterpreter(f)
	result.RPath = readDynamicString(f, elf.DT_RPATH)
	result.RunPath = readDynamicString(f, elf.DT_RUNPATH)
	result.Needed, _ = f.ImportedLibraries()
	return result, nil
}

func readInterpreter(f *elf.File) string {
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}
		data := make([]byte, prog.Filesz)
		if _, err := prog.ReadAt(data, 0); err != nil {
			return ""
		}
		return strings.TrimRight(string(data), "\x00")
	}
	return ""
}

func readDynamicString(f *elf.File, tag elf.DynTag) string {
	values, err := f.DynString(tag)
	if err != nil || len(values) == 0 {
		return ""
	}
	return strings.Join(values, ":")
}

func sha256Hex(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}

		target := filepath.Join(dst, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink package entry %q is not allowed", rel)
		}

		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		default:
			if !info.Mode().IsRegular() {
				return fmt.Errorf("package entry %q is not a regular file", rel)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, in); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		}
	})
}

func archFromMachine(machine elf.Machine) string {
	switch machine {
	case elf.EM_AARCH64:
		return "aarch64"
	case elf.EM_ARM:
		return "arm"
	case elf.EM_386:
		return "i386"
	case elf.EM_X86_64:
		return "x86_64"
	case elf.EM_RISCV:
		return "riscv64"
	default:
		return strings.ToLower(machine.String())
	}
}

func architectureMatches(expected, actual string) bool {
	return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(actual))
}

func compatibilityScore(level string) int64 {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "stable":
		return 3
	case "preview", "beta":
		return 2
	case "experimental":
		return 1
	default:
		return 0
	}
}

func creationScore(createdAt string) int64 {
	if strings.TrimSpace(createdAt) == "" {
		return 0
	}
	ts, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return 0
	}
	return ts.Unix()
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if strings.EqualFold(strings.TrimSpace(v), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func compatibilityStatus(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "stable":
		return StatusPass
	case "preview", "beta":
		return StatusWarn
	case "experimental":
		return StatusExperimental
	default:
		return StatusWarn
	}
}

func validationFailureDetail(report ValidationReport) string {
	for _, check := range report.Checks {
		if check.Status != StatusFail {
			continue
		}
		switch {
		case check.Message != "" && check.Path != "":
			return fmt.Sprintf("%s: %s [%s]", check.Name, check.Message, check.Path)
		case check.Message != "":
			return fmt.Sprintf("%s: %s", check.Name, check.Message)
		case check.Path != "":
			return fmt.Sprintf("%s [%s]", check.Name, check.Path)
		default:
			return check.Name
		}
	}
	return report.Status
}
