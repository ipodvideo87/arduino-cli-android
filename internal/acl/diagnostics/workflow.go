package diagnostics

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusPassed  Status = "passed"
	StatusWarning Status = "warning"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

func (s Status) String() string {
	if s == "" {
		return string(StatusPending)
	}
	return string(s)
}

func (s Status) IsTerminal() bool {
	switch s {
	case StatusPassed, StatusWarning, StatusFailed, StatusSkipped:
		return true
	default:
		return false
	}
}

func (s Status) Severity() int {
	switch s {
	case StatusFailed:
		return 4
	case StatusWarning:
		return 3
	case StatusPassed:
		return 2
	case StatusSkipped:
		return 1
	default:
		return 0
	}
}

type Step struct {
	Name       string            `json:"name"`
	Status     Status            `json:"status"`
	Message    string            `json:"message,omitempty"`
	StartedAt  time.Time         `json:"started_at,omitempty"`
	FinishedAt time.Time         `json:"finished_at,omitempty"`
	Evidence   []string          `json:"evidence,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type Workflow struct {
	Name      string    `json:"name"`
	Domain    string    `json:"domain,omitempty"`
	Steps     []Step    `json:"steps"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

func NewWorkflow(name string, stepNames ...string) *Workflow {
	w := &Workflow{Name: name}
	for _, stepName := range stepNames {
		w.AddStep(stepName)
	}
	return w
}

func (w *Workflow) AddStep(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if _, ok := w.Step(name); ok {
		return
	}
	w.Steps = append(w.Steps, Step{Name: name, Status: StatusPending})
	w.UpdatedAt = time.Now().UTC()
}

func (w *Workflow) SetStatus(name string, status Status, message string) {
	idx := w.indexOf(name)
	now := time.Now().UTC()
	if idx < 0 {
		w.Steps = append(w.Steps, Step{Name: name})
		idx = len(w.Steps) - 1
	}
	step := &w.Steps[idx]
	step.Name = strings.TrimSpace(name)
	if step.Name == "" {
		step.Name = name
	}
	step.Status = status
	step.Message = message
	if status == StatusRunning && step.StartedAt.IsZero() {
		step.StartedAt = now
	}
	if status.IsTerminal() {
		if step.StartedAt.IsZero() {
			step.StartedAt = now
		}
		step.FinishedAt = now
	}
	if status == StatusPending {
		step.StartedAt = time.Time{}
		step.FinishedAt = time.Time{}
	}
	w.UpdatedAt = now
}

func (w Workflow) Step(name string) (Step, bool) {
	idx := w.indexOf(name)
	if idx < 0 {
		return Step{}, false
	}
	return w.Steps[idx], true
}

func (w Workflow) Counts() map[Status]int {
	counts := map[Status]int{}
	for _, step := range w.Steps {
		if step.Status == "" {
			counts[StatusPending]++
			continue
		}
		counts[step.Status]++
	}
	return counts
}

func (w Workflow) Progress() float64 {
	if len(w.Steps) == 0 {
		return 0
	}
	done := 0
	for _, step := range w.Steps {
		if step.Status.IsTerminal() {
			done++
		}
	}
	return float64(done) / float64(len(w.Steps))
}

func (w Workflow) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", w.Name)
	if w.Domain != "" {
		fmt.Fprintf(&b, "domain: %s\n", w.Domain)
	}
	for _, step := range w.Steps {
		fmt.Fprintf(&b, "- %s: %s", step.Name, step.Status.String())
		if step.Message != "" {
			fmt.Fprintf(&b, " (%s)", step.Message)
		}
		if len(step.Evidence) > 0 {
			fmt.Fprintf(&b, " [%s]", strings.Join(step.Evidence, ", "))
		}
		fmt.Fprintln(&b)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (w Workflow) StepNames() []string {
	names := make([]string, 0, len(w.Steps))
	for _, step := range w.Steps {
		names = append(names, step.Name)
	}
	sort.Strings(names)
	return names
}

func (w Workflow) indexOf(name string) int {
	name = strings.TrimSpace(name)
	for i, step := range w.Steps {
		if step.Name == name {
			return i
		}
	}
	return -1
}
