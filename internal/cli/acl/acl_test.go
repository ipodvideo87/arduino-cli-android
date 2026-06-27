package acl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	acldiagnostics "github.com/arduino/arduino-cli/internal/acl/diagnostics"
	aclengine "github.com/arduino/arduino-cli/internal/acl/engine"
	aclinstall "github.com/arduino/arduino-cli/internal/acl/install"
	aclscanner "github.com/arduino/arduino-cli/internal/acl/scanner"
	"github.com/arduino/arduino-cli/internal/acl/transport"
	"github.com/arduino/arduino-cli/internal/acl/upload"
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

func TestACLWorkflowUploadCommandJSON(t *testing.T) {
	packageDir := t.TempDir()

	oldRun := workflowUploadRun
	workflowUploadRun = func(_ context.Context, req workflowUploadRequest) (aclengine.WorkflowReport, error) {
		require.Equal(t, packageDir, req.PackageDir)
		return aclengine.WorkflowReport{
			Name:     "upload",
			Status:   aclengine.StepStatusPassed,
			Beginner: "upload dry-run completed",
			Result: upload.UploadReport{
				SchemaVersion: "1",
				Status:        acldiagnostics.StatusPassed,
				DryRun:        true,
				Beginner:      "upload dry-run ready",
				Professional:  []string{"dry-run: true"},
				Plan: upload.UploadPlan{
					PackageDir: packageDir,
				},
			},
		}, nil
	}
	t.Cleanup(func() { workflowUploadRun = oldRun })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "workflow", "upload", "--json", packageDir})

	require.NoError(t, root.Execute())

	var report aclengine.WorkflowReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Equal(t, "upload", report.Name)
	require.Equal(t, aclengine.StepStatusPassed, report.Status)
	require.NotNil(t, report.Result)
}

func TestACLWorkflowUploadCommandTextAndDetails(t *testing.T) {
	packageDir := t.TempDir()

	oldRun := workflowUploadRun
	workflowUploadRun = func(_ context.Context, req workflowUploadRequest) (aclengine.WorkflowReport, error) {
		require.Equal(t, packageDir, req.PackageDir)
		return aclengine.WorkflowReport{
			Name:     "upload",
			Status:   aclengine.StepStatusWarning,
			Beginner: "upload dry-run completed with warnings",
			Result: upload.UploadReport{
				SchemaVersion: "1",
				Status:        acldiagnostics.StatusWarning,
				DryRun:        true,
				Beginner:      "upload dry-run completed with warnings",
				Professional:  []string{"dry-run: true", "ready: true"},
				Plan: upload.UploadPlan{
					PackageDir: packageDir,
				},
			},
		}, nil
	}
	t.Cleanup(func() { workflowUploadRun = oldRun })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "workflow", "upload", "--details", packageDir})

	require.NoError(t, root.Execute())

	output := buf.String()
	require.Contains(t, output, "ACL Workflow Upload")
	require.Contains(t, output, "upload dry-run completed with warnings")
	require.Contains(t, output, "dry-run: true")
}

func TestACLTransportListCommandJSON(t *testing.T) {
	oldFactory := newTransportProvider
	newTransportProvider = func() transportProvider {
		return fakeCLITransportProvider{
			desc: transport.TransportDescriptor{
				Kind:      transport.KindAndroidUSBFD,
				Name:      "termux-usb",
				Provider:  "termuxusb",
				Available: true,
			},
			report: transport.TransportDiagnosticsReport{
				SchemaVersion: "1",
				Status:        acldiagnostics.StatusPassed,
				Beginner:      "1 USB device discovered",
				Professional:  []string{"device list ready"},
				Devices: []transport.DiscoveredDevice{{
					Provider:        "termuxusb",
					StableID:        "/dev/bus/usb/001/002",
					DisplayName:     "/dev/bus/usb/001/002",
					TransportFamily: transport.TransportFamilyUSBSerial,
				}},
			},
		}
	}
	t.Cleanup(func() { newTransportProvider = oldFactory })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "transport", "list", "--json"})

	require.NoError(t, root.Execute())

	var report transportListReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Len(t, report.Devices, 1)
	require.Equal(t, acldiagnostics.StatusPassed, report.Status)
}

func TestACLTransportDiagnoseCommandTextAndDetails(t *testing.T) {
	oldFactory := newTransportProvider
	newTransportProvider = func() transportProvider {
		return fakeCLITransportProvider{
			desc: transport.TransportDescriptor{
				Kind:      transport.KindAndroidUSBFD,
				Name:      "termux-usb",
				Provider:  "termuxusb",
				Available: true,
			},
			report: transport.TransportDiagnosticsReport{
				SchemaVersion: "1",
				Status:        acldiagnostics.StatusWarning,
				Beginner:      "Termux USB diagnostics completed with limitations",
				Professional:  []string{"device list ready", "endpoint export is diagnostic-only"},
				Traces: []transport.CommandTrace{{
					Command:        "termux-usb",
					Args:           []string{"-l"},
					Stdout:         "[\"/dev/bus/usb/001/002\"]",
					ExitCode:       0,
					Interpretation: "termux-usb discovery",
				}},
			},
		}
	}
	t.Cleanup(func() { newTransportProvider = oldFactory })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "transport", "diagnose", "--details", "--device", "/dev/bus/usb/001/002"})

	require.NoError(t, root.Execute())
	output := buf.String()
	require.Contains(t, output, "ACL Transport Diagnose")
	require.Contains(t, output, "Termux USB diagnostics completed")
	require.Contains(t, output, "termux-usb -l")
}

func TestACLTransportAcquireCommandJSON(t *testing.T) {
	oldFactory := newTransportProvider
	newTransportProvider = func() transportProvider {
		return fakeCLITransportProvider{
			desc: transport.TransportDescriptor{
				Kind:      transport.KindAndroidUSBFD,
				Name:      "termux-usb",
				Provider:  "termuxusb",
				Available: true,
			},
			permission: transport.PermissionResult{
				State:       transport.PermissionStateGranted,
				Method:      "termux-usb -r",
				UserMessage: "USB permission granted",
				Professional: []string{
					"command: termux-usb -r /dev/bus/usb/001/002",
				},
			},
			report: transport.TransportDiagnosticsReport{
				SchemaVersion: "1",
				Status:        acldiagnostics.StatusPassed,
				Beginner:      "Termux USB diagnostics completed",
			},
		}
	}
	t.Cleanup(func() { newTransportProvider = oldFactory })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "transport", "acquire", "--json", "--device", "/dev/bus/usb/001/002"})

	require.NoError(t, root.Execute())

	var report transportAcquireReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Equal(t, transport.PermissionStateGranted, report.Permission.State)
	require.Equal(t, "/dev/bus/usb/001/002", report.DevicePath)
}

func TestACLTransportProbeFDCommandJSON(t *testing.T) {
	oldFactory := newTransportProvider
	newTransportProvider = func() transportProvider {
		return fakeCLITransportProvider{
			desc: transport.TransportDescriptor{
				Kind:      transport.KindAndroidUSBFD,
				Name:      "termux-usb",
				Provider:  "termuxusb",
				Available: true,
			},
			probe: transport.TransportStreamDiagnosticsReport{
				SchemaVersion:    "1",
				Status:           acldiagnostics.StatusWarning,
				Provider:         "termuxusb",
				ProviderKind:     transport.KindAndroidUSBFD,
				State:            transport.TransportStreamStateExperimental,
				FDEnvPresent:     true,
				FDObserved:       true,
				FDValid:          true,
				FDInspectable:    true,
				StreamSupported:  false,
				StreamProven:     false,
				ReadState:        transport.StreamObservationUnsupported,
				WriteState:       transport.StreamObservationUnsupported,
				CloseState:       transport.StreamObservationUnsupported,
				EOFState:         transport.StreamObservationUnsupported,
				DisconnectState:  transport.StreamObservationUnsupported,
				TermuxUSBCommand: "termux-usb -r -E -e helper /dev/bus/usb/001/002",
				Beginner:         "TERMUX_USB_FD observed; stream support remains experimental",
				Professional:     []string{"fd handoff observed", "byte-stream bridge not yet implemented"},
				NextStep:         "add a bounded byte-stream bridge or transport stream adapter",
				HandoffMode:      "env",
				FDSource:         "environment",
			},
		}
	}
	t.Cleanup(func() { newTransportProvider = oldFactory })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "transport", "probe-fd", "--json", "--device", "/dev/bus/usb/001/002"})

	require.NoError(t, root.Execute())

	var report transport.TransportStreamDiagnosticsReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.True(t, report.FDObserved)
	require.Equal(t, acldiagnostics.StatusWarning, report.Status)
	require.Equal(t, "env", report.HandoffMode)
	require.Equal(t, "environment", report.FDSource)
	require.Equal(t, "termux-usb -r -E -e helper /dev/bus/usb/001/002", report.TermuxUSBCommand)
	require.Equal(t, transport.TransportStreamStateExperimental, report.State)
	require.Contains(t, report.BeginnerSummary(), "TERMUX_USB_FD")
}

func TestACLTransportStreamStatusCommandJSON(t *testing.T) {
	oldFactory := newTransportProvider
	newTransportProvider = func() transportProvider {
		return fakeCLITransportProvider{
			desc: transport.TransportDescriptor{
				Kind:      transport.KindAndroidUSBFD,
				Name:      "termux-usb",
				Provider:  "termuxusb",
				Available: true,
			},
			probe: transport.TransportStreamDiagnosticsReport{
				SchemaVersion:    "1",
				Status:           acldiagnostics.StatusWarning,
				Provider:         "termuxusb",
				ProviderKind:     transport.KindAndroidUSBFD,
				State:            transport.TransportStreamStateExperimental,
				FDEnvPresent:     true,
				FDObserved:       true,
				FDValid:          true,
				FDInspectable:    true,
				StreamSupported:  false,
				StreamProven:     false,
				ReadState:        transport.StreamObservationExperimental,
				WriteState:       transport.StreamObservationExperimental,
				CloseState:       transport.StreamObservationExperimental,
				EOFState:         transport.StreamObservationExperimental,
				DisconnectState:  transport.StreamObservationExperimental,
				TermuxUSBCommand: "termux-usb -r -E -e helper /dev/bus/usb/001/002",
				Beginner:         "TERMUX_USB_FD observed; stream support remains experimental",
				Professional:     []string{"fd handoff observed", "bounded stream foundation in progress"},
				NextStep:         "add a bounded byte-stream bridge or transport stream adapter",
				HandoffMode:      "env",
				FDSource:         "environment",
			},
		}
	}
	t.Cleanup(func() { newTransportProvider = oldFactory })

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "transport", "stream-status", "--json", "--device", "/dev/bus/usb/001/002"})

	require.NoError(t, root.Execute())

	var report transport.TransportStreamDiagnosticsReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.Equal(t, transport.TransportStreamStateExperimental, report.State)
	require.Equal(t, transport.StreamObservationExperimental, report.ReadState)
	require.Equal(t, "termux-usb -r -E -e helper /dev/bus/usb/001/002", report.TermuxUSBCommand)
}

func TestACLTransportProbeFDHelperCommandJSON(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "termux-usb-fd-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	t.Setenv("TERMUX_USB_FD", fmt.Sprintf("%d", file.Fd()))

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "transport", "probe-fd-helper", "--json"})

	require.NoError(t, root.Execute())

	var report transport.TransportStreamDiagnosticsReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.True(t, report.FDObserved)
	require.True(t, report.FDValid)
	require.True(t, report.FDInspectable)
	require.Equal(t, "environment", report.FDSource)
	require.Equal(t, "env", report.HandoffMode)
	require.Equal(t, acldiagnostics.StatusWarning, report.Status)
}

func TestACLTransportProbeFDHelperCommandArgumentJSON(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "termux-usb-fd-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	t.Setenv("TERMUX_USB_FD", "")

	root := newTestRoot()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"acl", "transport", "probe-fd-helper", strconv.FormatUint(uint64(file.Fd()), 10), "--json"})

	require.NoError(t, root.Execute())

	var report transport.TransportStreamDiagnosticsReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &report))
	require.True(t, report.FDObserved)
	require.True(t, report.FDValid)
	require.True(t, report.FDInspectable)
	require.Equal(t, "argument", report.FDSource)
	require.Equal(t, "argv", report.HandoffMode)
}

type fakeCLITransportProvider struct {
	desc       transport.TransportDescriptor
	report     transport.TransportDiagnosticsReport
	permission transport.PermissionResult
	devices    []transport.DiscoveredDevice
	probe      transport.TransportStreamDiagnosticsReport
}

func (f fakeCLITransportProvider) Descriptor() transport.TransportDescriptor { return f.desc }

func (f fakeCLITransportProvider) Discover(context.Context, transport.DiscoveryRequest) ([]transport.DiscoveredDevice, error) {
	return append([]transport.DiscoveredDevice(nil), f.devices...), nil
}

func (f fakeCLITransportProvider) RequestPermission(context.Context, transport.PermissionRequest) (transport.PermissionResult, error) {
	if f.permission.State == "" {
		return transport.PermissionResult{State: transport.PermissionStateUnavailable}, nil
	}
	return f.permission, nil
}

func (f fakeCLITransportProvider) Open(context.Context, transport.OpenRequest) (transport.TransportSession, error) {
	return nil, nil
}

func (f fakeCLITransportProvider) Diagnostics(context.Context, transport.DiagnosticsRequest) (transport.TransportDiagnosticsReport, error) {
	return f.report, nil
}

func (f fakeCLITransportProvider) Probe(context.Context, transport.StreamProbeRequest) (transport.TransportStreamDiagnosticsReport, error) {
	if f.probe.SchemaVersion == "" {
		return transport.TransportStreamDiagnosticsReport{
			SchemaVersion: "1",
			Status:        acldiagnostics.StatusWarning,
			Beginner:      "stream probe is unavailable",
			Limitations:   []string{"probe not implemented in fake"},
		}, nil
	}
	return f.probe, nil
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
