package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
	"github.com/arduino/arduino-cli/internal/acl/firmware"
	aclscanner "github.com/arduino/arduino-cli/internal/acl/scanner"
	"github.com/arduino/arduino-cli/internal/acl/upload"
	aclverifier "github.com/arduino/arduino-cli/internal/acl/verifier"
	paths "github.com/arduino/go-paths-helper"
	properties "github.com/arduino/go-properties-orderedmap"
	"github.com/stretchr/testify/require"
)

func TestWorkflowStepOrdering(t *testing.T) {
	var order []string
	wf := Workflow{
		Name: "ordering",
		Jobs: []Job{
			{
				Name: "job",
				Steps: []Step{
					{Name: "one", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
						order = append(order, "one")
						return StepResult{Status: StepStatusPassed, Beginner: "one"}, nil
					}},
					{Name: "two", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
						order = append(order, "two")
						return StepResult{Status: StepStatusPassed, Beginner: "two"}, nil
					}},
					{Name: "three", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
						order = append(order, "three")
						return StepResult{Status: StepStatusPassed, Beginner: "three"}, nil
					}},
				},
			},
		},
	}

	report, err := New().Run(context.Background(), wf, NewContext())
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two", "three"}, order)
	require.Equal(t, StepStatusPassed, report.Status)
	require.Len(t, report.Jobs, 1)
	require.Len(t, report.Jobs[0].Steps, 3)
}

func TestEventEmission(t *testing.T) {
	sink := &captureSink{}
	engine := New()
	engine.Events.Subscribe(sink)

	wf := Workflow{
		Name: "events",
		Jobs: []Job{{Name: "job", Steps: []Step{{Name: "step", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
			return StepResult{Status: StepStatusPassed, Beginner: "ok"}, nil
		}}}}},
	}

	_, err := engine.Run(context.Background(), wf, NewContext())
	require.NoError(t, err)

	require.NotEmpty(t, sink.Events())
	require.Equal(t, EventWorkflowStarted, sink.Events()[0].Type)
	require.Equal(t, EventStepStarted, sink.Events()[2].Type)
	require.Equal(t, EventStepFinished, sink.Events()[3].Type)
	require.Equal(t, EventWorkflowFinished, sink.Events()[len(sink.Events())-1].Type)
}

func TestFailurePropagationStopsCriticalSteps(t *testing.T) {
	var ran []string
	wf := Workflow{
		Name: "failure",
		Jobs: []Job{{
			Name: "job",
			Steps: []Step{
				{Name: "first", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
					ran = append(ran, "first")
					return StepResult{Status: StepStatusPassed, Beginner: "first"}, nil
				}},
				{Name: "second", Critical: true, Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
					ran = append(ran, "second")
					return StepResult{Status: StepStatusFailed, Beginner: "second failed", Message: "boom"}, context.DeadlineExceeded
				}},
				{Name: "third", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
					ran = append(ran, "third")
					return StepResult{Status: StepStatusPassed, Beginner: "third"}, nil
				}},
			},
		}},
	}

	report, err := New().Run(context.Background(), wf, NewContext())
	require.Error(t, err)
	require.Equal(t, []string{"first", "second"}, ran)
	require.Equal(t, StepStatusFailed, report.Status)
	require.Len(t, report.Jobs[0].Steps, 2)
}

func TestSkippedOptionalSteps(t *testing.T) {
	wf := Workflow{
		Name: "skipped",
		Jobs: []Job{{
			Name: "job",
			Steps: []Step{
				{Name: "optional", Optional: true},
				{Name: "required", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
					return StepResult{Status: StepStatusPassed, Beginner: "required"}, nil
				}},
			},
		}},
	}

	report, err := New().Run(context.Background(), wf, NewContext())
	require.NoError(t, err)
	require.Equal(t, StepStatusPassed, report.Status)
	require.True(t, report.Jobs[0].Steps[0].Skipped)
	require.Equal(t, StepStatusSkipped, report.Jobs[0].Steps[0].Status)
}

func TestBeginnerProfessionalSeparationAndJSON(t *testing.T) {
	rootDir := t.TempDir()
	buildDir := filepath.Join(rootDir, "build")
	pkgDir := filepath.Join(rootDir, "package")
	require.NoError(t, os.MkdirAll(buildDir, 0o755))
	projectName := "demo.ino"
	files := map[string][]byte{
		filepath.Join(buildDir, projectName+".bin"):            []byte("app"),
		filepath.Join(buildDir, projectName+".elf"):            []byte("elf"),
		filepath.Join(buildDir, projectName+".map"):            []byte("map"),
		filepath.Join(buildDir, projectName+".partitions.bin"): []byte("partitions"),
		filepath.Join(rootDir, "bootloader.bin"):               []byte("bootloader"),
		filepath.Join(rootDir, "boot_app0.bin"):                []byte("boot_app0"),
	}
	for path, data := range files {
		require.NoError(t, os.WriteFile(path, data, 0o644))
	}
	props := properties.NewMap()
	props.Set("build.project_name", projectName)
	props.Set("runtime.platform.path", rootDir)
	props.Set("tools.esptool.upload.pattern", `esptool write_flash 0xe000 "`+filepath.Join(rootDir, "boot_app0.bin")+`" 0x1000 "`+filepath.Join(rootDir, "bootloader.bin")+`" 0x10000 "`+filepath.Join(buildDir, projectName+".bin")+`" 0x8000 "`+filepath.Join(buildDir, projectName+".partitions.bin")+`"`)
	_, err := firmware.BuildFirmwarePackage(firmware.BuildInput{
		BuildPath:        paths.New(buildDir),
		OutputDir:        paths.New(pkgDir),
		Properties:       props,
		SketchName:       "demo",
		ProjectName:      projectName,
		FQBN:             "esp32:esp32:esp32s3",
		Board:            "esp32s3",
		PlatformPackage:  "esp32",
		PlatformVersion:  "3.3.10",
		CoreVersion:      "3.3.10",
		ToolchainVersion: "gcc-14.2.0",
		TargetChip:       "ESP32-S3",
	})
	require.NoError(t, err)

	ctx := NewContext()
	ctx.Root = rootDir
	ctx.RuntimeRoot = rootDir
	ctx.CompileRequest = CompileRequest{
		SketchPath: filepath.Join(rootDir, "demo.ino"),
		FQBN:       "esp32:esp32:esp32s3",
		OutputDir:  pkgDir,
	}
	ctx.CompileRunner = compileRunnerFunc(func(_ context.Context, req CompileRequest, _ func(Event)) (CompileExecution, error) {
		return CompileExecution{
			SketchName:       "demo",
			FQBN:             req.FQBN,
			Board:            "esp32s3",
			PlatformPackage:  "esp32",
			PlatformVersion:  "3.3.10",
			CoreVersion:      "3.3.10",
			ToolchainVersion: "gcc-14.2.0",
			BuildPath:        buildDir,
			PackageDir:       pkgDir,
			MemoryUsage: firmware.MemoryUsage{
				ProgramUsedBytes:  1033681,
				ProgramTotalBytes: 1310720,
				ProgramPercent:    78,
				RAMUsedBytes:      44948,
				RAMTotalBytes:     327680,
				RAMPercent:        13,
			},
		}, nil
	})
	ctx.Set("compatibility_report", map[string]string{"decision": "selected"})

	report, err := New().Run(context.Background(), CompileWorkflow(), ctx)
	require.NoError(t, err)
	require.NotEmpty(t, report.BeginnerSummary())
	require.NotEmpty(t, report.ProfessionalDetails())
	require.Contains(t, report.BeginnerSummary(), "compile")
	require.Contains(t, report.ProfessionalDetails(), "workflow result available")

	data, err := report.JSON()
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, "compile", parsed["name"])
	require.Contains(t, parsed, "jobs")
	require.Contains(t, parsed, "result")
}

type compileRunnerFunc func(context.Context, CompileRequest, func(Event)) (CompileExecution, error)

func (f compileRunnerFunc) Run(ctx context.Context, req CompileRequest, publish func(Event)) (CompileExecution, error) {
	return f(ctx, req, publish)
}

func TestBootstrapWorkflowUsesScannerAndVerifier(t *testing.T) {
	rootDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "tool.sh"), []byte("#!/bin/sh\necho hi\n"), 0o755))

	ctx := NewContext()
	ctx.Root = rootDir
	ctx.RuntimeRoot = rootDir

	report, err := New().Run(context.Background(), BootstrapWorkflow(), ctx)
	require.NoError(t, err)
	require.Equal(t, "bootstrap", report.Name)
	require.NotEmpty(t, report.BeginnerSummary())
	require.NotEmpty(t, report.ProfessionalDetails())
	require.Contains(t, report.BeginnerSummary(), "validation")
}

func TestDiagnosticsWorkflowWithFakes(t *testing.T) {
	ctx := NewContext()
	ctx.Root = t.TempDir()
	ctx.RuntimeRoot = t.TempDir()
	ctx.Set("diagnostics_environment", map[string]string{"TERMUX_VERSION": "1"})
	ctx.Set("diagnostics_toolchain", map[string]string{"compiler.path": "/tmp/compiler"})
	scan := aclscanner.Report{Root: ctx.Root, Summary: aclscanner.Summary{TotalEntries: 1}}
	validation := aclscanner.ValidationReport{Root: ctx.Root, Summary: aclscanner.ValidationSummary{TotalFilesScanned: 1, Passed: true}}
	verifier := aclverifier.Report{Beginner: "verifier ready"}
	wf := Workflow{
		Name: "diagnostics",
		Jobs: []Job{{
			Name: "diagnostics",
			Steps: []Step{
				{Name: "scanner", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
					ctx.Set("diagnostics_scan", scan)
					ctx.Set("diagnostics_validation", validation)
					return StepResult{Status: StepStatusPassed, Beginner: "scanner ready", Professional: []string{"scanner detail"}, Data: scan}, nil
				}},
				{Name: "verifier", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
					ctx.Set("diagnostics_verifier", verifier)
					return StepResult{Status: StepStatusPassed, Beginner: "verifier ready", Professional: []string{"verifier detail"}, Data: verifier}, nil
				}},
				{Name: "android environment checks", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
					return StepResult{Status: StepStatusPassed, Beginner: "environment ok", Data: map[string]string{"TERMUX_VERSION": "1"}}, nil
				}},
				{Name: "toolchain checks", Execute: func(context.Context, *WorkflowContext) (StepResult, error) {
					return StepResult{Status: StepStatusPassed, Beginner: "toolchain ok", Data: map[string]string{"compiler.path": "/tmp/compiler"}}, nil
				}},
				{Name: "diagnostics report", Critical: true, Execute: runDiagnosticsReport},
			},
		}},
	}

	report, err := New().Run(context.Background(), wf, ctx)
	require.NoError(t, err)
	require.Equal(t, "diagnostics", report.Name)
	require.NotEmpty(t, report.BeginnerSummary())
	require.NotEmpty(t, report.ProfessionalDetails())
}

func TestUploadWorkflowUsesPrepareOnlyExecutor(t *testing.T) {
	packageDir := t.TempDir()

	oldExecutor := newUploadExecutor
	newUploadExecutor = func() upload.UploadExecutor {
		return uploadExecutorFunc(func(ctx context.Context, req upload.UploadExecutionRequest) (upload.UploadExecutionReport, error) {
			require.Equal(t, packageDir, req.PackageDir)
			return upload.UploadExecutionReport{
				SchemaVersion: "1",
				Status:        acldiagnostics.StatusPassed,
				DryRun:        true,
				PrepareOnly:   true,
				Beginner:      "upload execution prepared",
				Professional:  []string{"prepare-only: true"},
				Plan: upload.UploadExecutionPlan{
					PackageDir: packageDir,
					Operations: []upload.UploadOperation{{
						Name:     "application",
						Artifact: firmware.ArtifactApplicationBinary,
						Offset:   0x10000,
					}},
				},
			}, nil
		})
	}
	t.Cleanup(func() { newUploadExecutor = oldExecutor })

	ctx := NewContext()
	ctx.UploadRequest = upload.UploadRequest{PackageDir: packageDir}

	report, err := New().Run(context.Background(), UploadWorkflow(), ctx)
	require.NoError(t, err)
	require.Equal(t, "upload", report.Name)
	require.NotEmpty(t, report.Jobs)
	require.NotNil(t, report.Result)
	result, ok := report.Result.(upload.UploadExecutionReport)
	require.True(t, ok)
	require.True(t, result.DryRun)
	require.True(t, result.PrepareOnly)
	require.Equal(t, packageDir, result.Plan.PackageDir)
	require.Len(t, result.Plan.Operations, 1)
}

type uploadExecutorFunc func(context.Context, upload.UploadExecutionRequest) (upload.UploadExecutionReport, error)

func (f uploadExecutorFunc) Prepare(context.Context, upload.UploadExecutionRequest) (upload.UploadExecutionPlan, error) {
	report, err := f(context.Background(), upload.UploadExecutionRequest{})
	return report.Plan, err
}

func (f uploadExecutorFunc) PrepareOnly(ctx context.Context, req upload.UploadExecutionRequest) (upload.UploadExecutionReport, error) {
	return f(ctx, req)
}

type captureSink struct {
	mu     sync.Mutex
	events []Event
}

func (s *captureSink) Handle(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *captureSink) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Event(nil), s.events...)
}
