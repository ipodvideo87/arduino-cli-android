package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/upload"
)

type StepStatus = acldiagnostics.Status

const (
	StepStatusPending = acldiagnostics.StatusPending
	StepStatusRunning = acldiagnostics.StatusRunning
	StepStatusPassed  = acldiagnostics.StatusPassed
	StepStatusWarning = acldiagnostics.StatusWarning
	StepStatusFailed  = acldiagnostics.StatusFailed
	StepStatusSkipped = acldiagnostics.StatusSkipped
)

type EventType string

const (
	EventWorkflowStarted  EventType = "workflow.started"
	EventWorkflowFinished EventType = "workflow.finished"
	EventJobStarted       EventType = "job.started"
	EventJobFinished      EventType = "job.finished"
	EventStepStarted      EventType = "step.started"
	EventStepProgress     EventType = "step.progress"
	EventStepFinished     EventType = "step.finished"
	EventStepSkipped      EventType = "step.skipped"
	EventStepFailed       EventType = "step.failed"
)

type Event struct {
	Time     time.Time         `json:"time"`
	Type     EventType         `json:"type"`
	Workflow string            `json:"workflow,omitempty"`
	Job      string            `json:"job,omitempty"`
	Step     string            `json:"step,omitempty"`
	Status   StepStatus        `json:"status,omitempty"`
	Progress int               `json:"progress,omitempty"`
	Message  string            `json:"message,omitempty"`
	Evidence []string          `json:"evidence,omitempty"`
	Error    string            `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type EventSink interface {
	Handle(Event) error
}

type EventBus struct {
	mu    sync.RWMutex
	sinks []EventSink
}

func NewEventBus() *EventBus {
	return &EventBus{}
}

func (b *EventBus) Subscribe(sink EventSink) {
	if b == nil || sink == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sinks = append(b.sinks, sink)
}

func (b *EventBus) Publish(event Event) error {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	sinks := append([]EventSink(nil), b.sinks...)
	b.mu.RUnlock()
	var errs []string
	for _, sink := range sinks {
		if err := sink.Handle(event); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

type WorkflowContext struct {
	Root           string
	RuntimeRoot    string
	TargetPath     string
	BuildPath      string
	OutputDir      string
	SketchName     string
	FQBN           string
	CompileRequest CompileRequest
	CompileRunner  CompileRunner
	UploadRequest  upload.UploadRequest
	Events         *EventBus
	Metadata       map[string]string
	Data           map[string]any
}

func NewContext() *WorkflowContext {
	return &WorkflowContext{
		Metadata: map[string]string{},
		Data:     map[string]any{},
	}
}

func (c *WorkflowContext) Set(key string, value any) {
	if c == nil {
		return
	}
	if c.Data == nil {
		c.Data = map[string]any{}
	}
	c.Data[key] = value
}

func (c *WorkflowContext) Get(key string) (any, bool) {
	if c == nil || c.Data == nil {
		return nil, false
	}
	v, ok := c.Data[key]
	return v, ok
}

func (c *WorkflowContext) Publish(event Event) error {
	if c == nil || c.Events == nil {
		return nil
	}
	return c.Events.Publish(event)
}

type StepExecutor func(context.Context, *WorkflowContext) (StepResult, error)

type Step struct {
	Name     string
	Optional bool
	Critical bool
	Execute  StepExecutor
}

type Job struct {
	Name     string
	Optional bool
	Critical bool
	Steps    []Step
}

type Workflow struct {
	Name string
	Jobs []Job
}

type StepResult struct {
	Name         string     `json:"name"`
	Status       StepStatus `json:"status"`
	Message      string     `json:"message,omitempty"`
	Beginner     string     `json:"beginner_summary,omitempty"`
	Professional []string   `json:"professional_details,omitempty"`
	Evidence     []string   `json:"evidence,omitempty"`
	StartedAt    time.Time  `json:"started_at,omitempty"`
	FinishedAt   time.Time  `json:"finished_at,omitempty"`
	Optional     bool       `json:"optional,omitempty"`
	Critical     bool       `json:"critical,omitempty"`
	Skipped      bool       `json:"skipped,omitempty"`
	Data         any        `json:"data,omitempty"`
	Error        string     `json:"error,omitempty"`
}

type JobResult struct {
	Name         string       `json:"name"`
	Status       StepStatus   `json:"status"`
	Steps        []StepResult `json:"steps,omitempty"`
	Beginner     string       `json:"beginner_summary,omitempty"`
	Professional []string     `json:"professional_details,omitempty"`
	StartedAt    time.Time    `json:"started_at,omitempty"`
	FinishedAt   time.Time    `json:"finished_at,omitempty"`
}

type WorkflowReport struct {
	Name         string            `json:"name"`
	Status       StepStatus        `json:"status"`
	Jobs         []JobResult       `json:"jobs,omitempty"`
	Beginner     string            `json:"beginner_summary,omitempty"`
	Professional []string          `json:"professional_details,omitempty"`
	Result       any               `json:"result,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	StartedAt    time.Time         `json:"started_at,omitempty"`
	FinishedAt   time.Time         `json:"finished_at,omitempty"`
}

func (r WorkflowReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func (r WorkflowReport) StepCount() int {
	count := 0
	for _, job := range r.Jobs {
		count += len(job.Steps)
	}
	return count
}

func (r WorkflowReport) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	if r.Status == "" {
		return "workflow pending"
	}
	return fmt.Sprintf("workflow %s", r.Status)
}

func (r WorkflowReport) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	for _, job := range r.Jobs {
		details = append(details, fmt.Sprintf("job %s: %s", job.Name, job.Status))
	}
	return details
}

type Engine struct {
	Events *EventBus
}

func New() *Engine {
	return &Engine{Events: NewEventBus()}
}

func (e *Engine) Run(ctx context.Context, workflow Workflow, wctx *WorkflowContext) (WorkflowReport, error) {
	if wctx == nil {
		wctx = NewContext()
	}
	if wctx.Metadata == nil {
		wctx.Metadata = map[string]string{}
	}
	if wctx.Data == nil {
		wctx.Data = map[string]any{}
	}
	wctx.Events = e.Events

	report := WorkflowReport{
		Name:      workflow.Name,
		Status:    StepStatusPending,
		Metadata:  cloneStringMap(wctx.Metadata),
		StartedAt: time.Now().UTC(),
	}
	_ = e.publish(Event{Time: report.StartedAt, Type: EventWorkflowStarted, Workflow: workflow.Name, Metadata: cloneStringMap(wctx.Metadata)})

	criticalErr := error(nil)
	for _, job := range workflow.Jobs {
		jobResult, jobErr := e.runJob(ctx, workflow.Name, job, wctx)
		report.Jobs = append(report.Jobs, jobResult)
		report.Status = mergeStatus(report.Status, jobResult.Status)
		if len(jobResult.Steps) > 0 {
			if data := jobResult.Steps[len(jobResult.Steps)-1].Data; data != nil {
				report.Result = data
			}
		}
		if jobErr != nil {
			criticalErr = jobErr
			break
		}
	}

	report.FinishedAt = time.Now().UTC()
	report.Status = finalizeStatus(report.Status, report.Jobs)
	report.Beginner = summarizeWorkflow(report)
	report.Professional = collectProfessional(report)
	_ = e.publish(Event{Time: report.FinishedAt, Type: EventWorkflowFinished, Workflow: workflow.Name, Status: report.Status, Message: report.Beginner, Metadata: cloneStringMap(wctx.Metadata)})

	if criticalErr != nil {
		return report, criticalErr
	}
	if report.Status == StepStatusFailed {
		return report, errors.New(report.BeginnerSummary())
	}
	return report, nil
}

func (e *Engine) runJob(ctx context.Context, workflowName string, job Job, wctx *WorkflowContext) (JobResult, error) {
	result := JobResult{
		Name:      job.Name,
		Status:    StepStatusPending,
		StartedAt: time.Now().UTC(),
	}
	_ = e.publish(Event{Time: result.StartedAt, Type: EventJobStarted, Workflow: workflowName, Job: job.Name})

	for _, step := range job.Steps {
		stepResult, err := e.runStep(ctx, workflowName, job.Name, step, wctx)
		result.Steps = append(result.Steps, stepResult)
		result.Status = mergeStatus(result.Status, stepResult.Status)
		if err != nil {
			if stepResult.Critical || job.Critical {
				result.FinishedAt = time.Now().UTC()
				result.Status = StepStatusFailed
				result.Beginner = summarizeJob(result)
				result.Professional = collectJobProfessional(result)
				_ = e.publish(Event{Time: result.FinishedAt, Type: EventJobFinished, Workflow: workflowName, Job: job.Name, Status: result.Status, Message: result.Beginner})
				return result, err
			}
		}
		if stepResult.Status == StepStatusFailed && (stepResult.Critical || job.Critical) {
			result.FinishedAt = time.Now().UTC()
			result.Status = StepStatusFailed
			result.Beginner = summarizeJob(result)
			result.Professional = collectJobProfessional(result)
			_ = e.publish(Event{Time: result.FinishedAt, Type: EventJobFinished, Workflow: workflowName, Job: job.Name, Status: result.Status, Message: result.Beginner})
			return result, errors.New(stepResult.Message)
		}
	}

	result.FinishedAt = time.Now().UTC()
	result.Status = finalizeJobStatus(result.Status, result.Steps)
	result.Beginner = summarizeJob(result)
	result.Professional = collectJobProfessional(result)
	_ = e.publish(Event{Time: result.FinishedAt, Type: EventJobFinished, Workflow: workflowName, Job: job.Name, Status: result.Status, Message: result.Beginner})
	return result, nil
}

func (e *Engine) runStep(ctx context.Context, workflowName, jobName string, step Step, wctx *WorkflowContext) (StepResult, error) {
	now := time.Now().UTC()
	result := StepResult{
		Name:      step.Name,
		Status:    StepStatusPending,
		Optional:  step.Optional,
		Critical:  step.Critical,
		StartedAt: now,
	}
	_ = e.publish(Event{Time: now, Type: EventStepStarted, Workflow: workflowName, Job: jobName, Step: step.Name, Status: StepStatusRunning})

	if step.Execute == nil {
		if step.Optional {
			result.Status = StepStatusSkipped
			result.Skipped = true
			result.Message = "optional step not configured"
			result.FinishedAt = time.Now().UTC()
			_ = e.publish(Event{Time: result.FinishedAt, Type: EventStepSkipped, Workflow: workflowName, Job: jobName, Step: step.Name, Status: result.Status, Message: result.Message})
			return result, nil
		}
		result.Status = StepStatusFailed
		result.Message = "step executor is not configured"
		result.FinishedAt = time.Now().UTC()
		_ = e.publish(Event{Time: result.FinishedAt, Type: EventStepFailed, Workflow: workflowName, Job: jobName, Step: step.Name, Status: result.Status, Message: result.Message})
		return result, errors.New(result.Message)
	}

	executed, err := step.Execute(ctx, wctx)
	executed.Name = step.Name
	executed.Optional = step.Optional
	executed.Critical = step.Critical
	if executed.StartedAt.IsZero() {
		executed.StartedAt = now
	}
	executed.FinishedAt = time.Now().UTC()
	if executed.Status == "" {
		if err != nil {
			executed.Status = StepStatusFailed
		} else {
			executed.Status = StepStatusPassed
		}
	}
	if executed.Beginner == "" {
		executed.Beginner = executed.Message
	}
	if executed.Beginner == "" && executed.Status == StepStatusPassed {
		executed.Beginner = "passed"
	}
	if executed.Message == "" {
		executed.Message = executed.Beginner
	}
	if err != nil && executed.Status != StepStatusFailed {
		executed.Status = StepStatusFailed
		if executed.Message == "" {
			executed.Message = err.Error()
		}
	}
	switch executed.Status {
	case StepStatusSkipped:
		executed.Skipped = true
		_ = e.publish(Event{Time: executed.FinishedAt, Type: EventStepSkipped, Workflow: workflowName, Job: jobName, Step: step.Name, Status: executed.Status, Message: executed.Message, Evidence: executed.Evidence})
	case StepStatusFailed:
		_ = e.publish(Event{Time: executed.FinishedAt, Type: EventStepFailed, Workflow: workflowName, Job: jobName, Step: step.Name, Status: executed.Status, Message: executed.Message, Evidence: executed.Evidence, Error: errString(err)})
	default:
		_ = e.publish(Event{Time: executed.FinishedAt, Type: EventStepFinished, Workflow: workflowName, Job: jobName, Step: step.Name, Status: executed.Status, Message: executed.Message, Evidence: executed.Evidence})
	}
	if executed.Data != nil {
		wctx.Set(step.Name, executed.Data)
	}
	return executed, err
}

func (e *Engine) publish(event Event) error {
	if e == nil || e.Events == nil {
		return nil
	}
	return e.Events.Publish(event)
}

func mergeStatus(current, next StepStatus) StepStatus {
	if next == "" {
		return current
	}
	if current == "" {
		return next
	}
	if next.Severity() > current.Severity() {
		return next
	}
	if current == StepStatusPending && next == StepStatusRunning {
		return current
	}
	return current
}

func finalizeJobStatus(current StepStatus, steps []StepResult) StepStatus {
	if len(steps) == 0 {
		return StepStatusPending
	}
	for _, step := range steps {
		if step.Status == StepStatusFailed {
			return StepStatusFailed
		}
	}
	if current == StepStatusWarning {
		return StepStatusWarning
	}
	if current == StepStatusPending {
		return StepStatusPending
	}
	if current == "" {
		return StepStatusPassed
	}
	return current
}

func finalizeStatus(current StepStatus, jobs []JobResult) StepStatus {
	if len(jobs) == 0 {
		return StepStatusPending
	}
	for _, job := range jobs {
		if job.Status == StepStatusFailed {
			return StepStatusFailed
		}
	}
	if current == StepStatusWarning {
		return StepStatusWarning
	}
	if current == StepStatusPending {
		return StepStatusPending
	}
	if current == "" {
		return StepStatusPassed
	}
	return current
}

func summarizeJob(r JobResult) string {
	parts := []string{}
	for _, step := range r.Steps {
		if strings.TrimSpace(step.Beginner) != "" {
			parts = append(parts, step.Beginner)
		} else if strings.TrimSpace(step.Message) != "" {
			parts = append(parts, step.Message)
		}
	}
	if len(parts) == 0 {
		return "job completed"
	}
	return strings.Join(parts, "; ")
}

func summarizeWorkflow(r WorkflowReport) string {
	parts := []string{}
	for _, job := range r.Jobs {
		if strings.TrimSpace(job.Beginner) != "" {
			parts = append(parts, job.Beginner)
		}
	}
	if len(parts) == 0 {
		return "workflow completed"
	}
	return strings.Join(parts, "; ")
}

func collectJobProfessional(r JobResult) []string {
	details := make([]string, 0, len(r.Steps))
	for _, step := range r.Steps {
		if len(step.Professional) > 0 {
			details = append(details, step.Professional...)
		}
		if step.Message != "" {
			details = append(details, fmt.Sprintf("%s: %s", step.Name, step.Message))
		}
	}
	return details
}

func collectProfessional(r WorkflowReport) []string {
	details := make([]string, 0, len(r.Jobs))
	for _, job := range r.Jobs {
		details = append(details, fmt.Sprintf("job %s: %s", job.Name, job.Status))
		details = append(details, job.Professional...)
	}
	if r.Result != nil {
		details = append(details, "workflow result available")
	}
	return details
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
