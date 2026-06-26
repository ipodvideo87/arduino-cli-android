package acl

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
	aclengine "github.com/arduino/arduino-cli/internal/acl/engine"
	aclinstall "github.com/arduino/arduino-cli/internal/acl/install"
	aclscanner "github.com/arduino/arduino-cli/internal/acl/scanner"
	aclverifier "github.com/arduino/arduino-cli/internal/acl/verifier"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func newTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "arduino-cli"}
	root.PersistentFlags().Bool("json", false, "json output")
	root.AddCommand(NewCommand(nil))
	return root
}

func createExecutableScript(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\necho hello\n"), 0o755))
	return path
}

func TestACLScanCommandTextAndDetails(t *testing.T) {
	rootDir := t.TempDir()
	script := createExecutableScript(t, rootDir, "tool.sh")

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "scan", "--details", rootDir})

	require.NoError(t, root.Execute())

	output := buf.String()
	require.Contains(t, output, "ACL Scanner")
	require.Contains(t, output, "entries scanned")
	require.Contains(t, output, "tool.sh")
	require.Contains(t, output, filepath.Base(script))
}

func TestACLScanCommandJSON(t *testing.T) {
	rootDir := t.TempDir()
	createExecutableScript(t, rootDir, "tool.sh")

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "scan", "--json", rootDir})

	require.NoError(t, root.Execute())

	var report aclscanner.Report
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Equal(t, filepath.Clean(rootDir), report.Root)
	require.NotEmpty(t, report.Entries)
	require.NotEmpty(t, report.Summary.TotalEntries)
}

func TestACLScanCommandBeginnerAndProfessionalSeparation(t *testing.T) {
	rootDir := t.TempDir()
	createExecutableScript(t, rootDir, "tool.sh")

	root := newTestRoot()
	beginnerBuf := &bytes.Buffer{}
	root.SetOut(beginnerBuf)
	root.SetErr(beginnerBuf)
	root.SetArgs([]string{"acl", "scan", rootDir})

	require.NoError(t, root.Execute())
	require.Contains(t, beginnerBuf.String(), "ACL Scanner")
	require.NotContains(t, beginnerBuf.String(), "tool.sh:")

	root = newTestRoot()
	detailsBuf := &bytes.Buffer{}
	root.SetOut(detailsBuf)
	root.SetErr(detailsBuf)
	root.SetArgs([]string{"acl", "scan", "--details", rootDir})

	require.NoError(t, root.Execute())
	require.Contains(t, detailsBuf.String(), "tool.sh:")
}

func TestACLVerifyCommandFailureReporting(t *testing.T) {
	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "verify", "/definitely/not/present"})

	err := root.Execute()
	require.Error(t, err)
	require.Contains(t, buf.String(), "ACL Verifier")
	require.Contains(t, buf.String(), "scan root")
}

func TestACLVerifyCommandJSONSuccess(t *testing.T) {
	rootDir := t.TempDir()
	createExecutableScript(t, rootDir, "tool.sh")

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "verify", "--json", rootDir})

	require.NoError(t, root.Execute())

	var parsed aclverifier.Report
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	require.Equal(t, filepath.Clean(rootDir), parsed.Request.Root)
}

func TestACLPatchPreviewCommandJSON(t *testing.T) {
	rootDir := t.TempDir()
	createExecutableScript(t, rootDir, "tool.sh")

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "patch-preview", "--json", rootDir})

	require.NoError(t, root.Execute())

	var report map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Equal(t, filepath.Clean(rootDir), report["root"])
	require.Contains(t, report, "summary")
}

func TestACLBootstrapCommandDetailsAndPipeline(t *testing.T) {
	rootDir := t.TempDir()
	createExecutableScript(t, rootDir, "tool.sh")
	runtimeRoot := t.TempDir()

	oldExec := newBootstrapExec
	newBootstrapExec = func(report *BootstrapReport) aclinstall.StageExecutor {
		return &testBootstrapExecutor{report: report}
	}
	t.Cleanup(func() { newBootstrapExec = oldExec })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "bootstrap", "--details", "--runtime-root", runtimeRoot, rootDir})

	require.NoError(t, root.Execute())

	output := buf.String()
	require.Contains(t, output, "ACL Bootstrap")
	require.Contains(t, output, "Status:")
	require.Contains(t, output, "permission-runtime-fixes")
	require.Contains(t, output, ".acl/runtime/ld-linux-aarch64.so.1")
}

func TestACLBootstrapCommandJSON(t *testing.T) {
	rootDir := t.TempDir()
	createExecutableScript(t, rootDir, "tool.sh")
	runtimeRoot := t.TempDir()

	oldExec := newBootstrapExec
	newBootstrapExec = func(report *BootstrapReport) aclinstall.StageExecutor {
		return &testBootstrapExecutor{report: report}
	}
	t.Cleanup(func() { newBootstrapExec = oldExec })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "bootstrap", "--json", "--runtime-root", runtimeRoot, rootDir})

	require.NoError(t, root.Execute())

	var report BootstrapReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Equal(t, filepath.Clean(rootDir), report.Root)
	require.NotEmpty(t, report.Pipeline.Stages)
	require.Equal(t, acldiagnostics.StatusWarning, report.Status)
}

func TestACLWorkflowBootstrapCommandJSON(t *testing.T) {
	rootDir := t.TempDir()
	createExecutableScript(t, rootDir, "tool.sh")
	runtimeRoot := t.TempDir()

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "workflow", "bootstrap", "--json", "--runtime-root", runtimeRoot, rootDir})

	require.NoError(t, root.Execute())

	var report aclengine.WorkflowReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Equal(t, "bootstrap", report.Name)
	require.NotEmpty(t, report.Jobs)
	require.NotNil(t, report.Result)
}

func TestACLWorkflowCompileCommandJSON(t *testing.T) {
	sketchDir := t.TempDir()

	oldRun := workflowCompileRun
	workflowCompileRun = func(_ context.Context, _ rpc.ArduinoCoreServiceServer, _ workflowCompileRequest) (aclengine.WorkflowReport, error) {
		return aclengine.WorkflowReport{
			Name:     "compile",
			Status:   aclengine.StepStatusPassed,
			Beginner: "compile workflow completed",
			Result: aclengine.CompileWorkflowReport{
				PackagePath:  filepath.Join(sketchDir, "build", "esp32.esp32.esp32s3", "firmware-package"),
				Beginner:     "compile workflow completed and package is ready to flash",
				Professional: []string{"package path: " + filepath.Join(sketchDir, "build", "esp32.esp32.esp32s3", "firmware-package")},
				PackageReady: true,
				ReadyToFlash: true,
			},
		}, nil
	}
	t.Cleanup(func() { workflowCompileRun = oldRun })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "workflow", "compile", "--json", "--fqbn", "esp32:esp32:esp32s3", sketchDir})

	require.NoError(t, root.Execute())

	var report aclengine.WorkflowReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Equal(t, "compile", report.Name)
	require.Equal(t, aclengine.StepStatusPassed, report.Status)
	require.NotNil(t, report.Result)
}

func TestACLWorkflowCompileCommandTextAndDetails(t *testing.T) {
	sketchDir := t.TempDir()
	pkgDir := filepath.Join(sketchDir, "build", "esp32.esp32.esp32s3", "firmware-package")

	oldRun := workflowCompileRun
	workflowCompileRun = func(_ context.Context, _ rpc.ArduinoCoreServiceServer, _ workflowCompileRequest) (aclengine.WorkflowReport, error) {
		return aclengine.WorkflowReport{
			Name:     "compile",
			Status:   aclengine.StepStatusPassed,
			Beginner: "compile workflow completed",
			Result: aclengine.CompileWorkflowReport{
				PackagePath:  pkgDir,
				Beginner:     "compile workflow completed and package is ready to flash",
				Professional: []string{"package path: " + pkgDir, "build path: /tmp/build"},
				PackageReady: true,
				ReadyToFlash: true,
			},
		}, nil
	}
	t.Cleanup(func() { workflowCompileRun = oldRun })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "workflow", "compile", "--details", "--fqbn", "esp32:esp32:esp32s3", sketchDir})

	require.NoError(t, root.Execute())

	output := buf.String()
	require.Contains(t, output, "ACL Workflow Compile")
	require.Contains(t, output, "compile workflow completed")
	require.Contains(t, output, "package path: "+pkgDir)
}

func TestACLWorkflowCompileCommandSeparatesBuildAndOutputDirs(t *testing.T) {
	sketchDir := t.TempDir()
	buildDir := filepath.Join(sketchDir, "build-dir")
	outputDir := filepath.Join(sketchDir, "export-dir")

	oldRun := workflowCompileRun
	workflowCompileRun = func(_ context.Context, _ rpc.ArduinoCoreServiceServer, req workflowCompileRequest) (aclengine.WorkflowReport, error) {
		require.Equal(t, buildDir, req.BuildPath)
		require.Equal(t, outputDir, req.OutputDir)
		return aclengine.WorkflowReport{
			Name:     "compile",
			Status:   aclengine.StepStatusPassed,
			Beginner: "compile workflow completed",
		}, nil
	}
	t.Cleanup(func() { workflowCompileRun = oldRun })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "workflow", "compile", "--fqbn", "esp32:esp32:esp32s3", "--build-path", buildDir, "--output-dir", outputDir, sketchDir})

	require.NoError(t, root.Execute())
}

type testBootstrapExecutor struct {
	report *BootstrapReport
}

func (e *testBootstrapExecutor) Execute(_ context.Context, req aclinstall.StageRequest) (aclinstall.StageResult, error) {
	if e == nil || e.report == nil {
		return aclinstall.StageResult{Status: acldiagnostics.StatusFailed, Message: "missing report"}, nil
	}
	switch req.Stage {
	case aclinstall.StagePermissionRuntimeFixes:
		return aclinstall.StageResult{
			Status:   acldiagnostics.StatusWarning,
			Message:  "runtime permission repairs would be required",
			Evidence: []string{".acl/runtime/ld-linux-aarch64.so.1"},
		}, nil
	case aclinstall.StageExecutableValidation:
		if e.report.ScanValidation.Summary.Errors > 0 {
			return aclinstall.StageResult{Status: acldiagnostics.StatusFailed, Message: "scanner validation failed"}, nil
		}
		return aclinstall.StageResult{Status: acldiagnostics.StatusPassed, Message: "scanner validation passed"}, nil
	default:
		return aclinstall.StageResult{Status: acldiagnostics.StatusPassed, Message: string(req.Stage)}, nil
	}
}
