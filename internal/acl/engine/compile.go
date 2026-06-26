package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/arduino/arduino-cli/internal/acl/compatibility"
	"github.com/arduino/arduino-cli/internal/acl/firmware"
)

type CompileRequest struct {
	SketchPath             string   `json:"sketch_path,omitempty"`
	FQBN                   string   `json:"fqbn,omitempty"`
	BuildPath              string   `json:"build_path,omitempty"`
	OutputDir              string   `json:"output_dir,omitempty"`
	BuildProperties        []string `json:"build_properties,omitempty"`
	Libraries              []string `json:"libraries,omitempty"`
	Library                []string `json:"library,omitempty"`
	BuildCachePath         string   `json:"build_cache_path,omitempty"`
	BuildCacheExtraPaths   []string `json:"build_cache_extra_paths,omitempty"`
	Clean                  bool     `json:"clean,omitempty"`
	OptimizeForDebug       bool     `json:"optimize_for_debug,omitempty"`
	Warnings               string   `json:"warnings,omitempty"`
	Verbose                bool     `json:"verbose,omitempty"`
	Quiet                  bool     `json:"quiet,omitempty"`
	ExportBinaries         bool     `json:"export_binaries,omitempty"`
	SkipLibrariesDiscovery bool     `json:"skip_libraries_discovery,omitempty"`
	Jobs                   int32    `json:"jobs,omitempty"`
}

type CompileExecution struct {
	SketchName       string                   `json:"sketch_name,omitempty"`
	FQBN             string                   `json:"fqbn,omitempty"`
	Board            string                   `json:"board,omitempty"`
	PlatformPackage  string                   `json:"platform_package,omitempty"`
	PlatformVersion  string                   `json:"platform_version,omitempty"`
	CoreVersion      string                   `json:"core_version,omitempty"`
	ToolchainVersion string                   `json:"toolchain_version,omitempty"`
	TargetChip       string                   `json:"target_chip,omitempty"`
	TargetFamily     string                   `json:"target_family,omitempty"`
	BuildPath        string                   `json:"build_path,omitempty"`
	OutputDir        string                   `json:"output_dir,omitempty"`
	PackageDir       string                   `json:"package_dir,omitempty"`
	BuildProperties  map[string]string        `json:"build_properties,omitempty"`
	MemoryUsage      firmware.MemoryUsage     `json:"memory_usage,omitempty"`
	Libraries        []firmware.LibraryRef    `json:"libraries,omitempty"`
	Compatibility    []compatibility.Decision `json:"compatibility,omitempty"`
	BuilderResult    map[string]any           `json:"builder_result,omitempty"`
}

type CompileRunner interface {
	Run(context.Context, CompileRequest, func(Event)) (CompileExecution, error)
}

type CompileWorkflowReport struct {
	Request       CompileRequest            `json:"request"`
	Execution     CompileExecution          `json:"execution,omitempty"`
	Compatibility compatibility.Report      `json:"compatibility,omitempty"`
	Package       firmware.FirmwarePackage  `json:"package,omitempty"`
	PackagePath   string                    `json:"package_path,omitempty"`
	Validation    firmware.ValidationReport `json:"validation,omitempty"`
	Beginner      string                    `json:"beginner_summary,omitempty"`
	Professional  []string                  `json:"professional_details,omitempty"`
	PackageReady  bool                      `json:"package_ready,omitempty"`
	ReadyToFlash  bool                      `json:"ready_to_flash,omitempty"`
}

func (r CompileWorkflowReport) JSON() ([]byte, error) {
	return jsonMarshalIndent(r)
}

func (r CompileWorkflowReport) BeginnerSummary() string {
	if strings.TrimSpace(r.Beginner) != "" {
		return r.Beginner
	}
	if r.ReadyToFlash {
		return "compile workflow completed and package is ready to flash"
	}
	return "compile workflow completed"
}

func (r CompileWorkflowReport) ProfessionalDetails() []string {
	details := append([]string(nil), r.Professional...)
	if r.PackagePath != "" {
		details = append(details, "package path: "+r.PackagePath)
	}
	if r.Execution.BuildPath != "" {
		details = append(details, "build path: "+r.Execution.BuildPath)
	}
	if r.Execution.OutputDir != "" {
		details = append(details, "output dir: "+r.Execution.OutputDir)
	}
	if r.Execution.MemoryUsage.ProgramTotalBytes > 0 || r.Execution.MemoryUsage.RAMTotalBytes > 0 {
		details = append(details, fmt.Sprintf("memory usage: flash %d/%d (%d%%), ram %d/%d (%d%%)",
			r.Execution.MemoryUsage.ProgramUsedBytes, r.Execution.MemoryUsage.ProgramTotalBytes, r.Execution.MemoryUsage.ProgramPercent,
			r.Execution.MemoryUsage.RAMUsedBytes, r.Execution.MemoryUsage.RAMTotalBytes, r.Execution.MemoryUsage.RAMPercent,
		))
	}
	return details
}

func (r CompileWorkflowReport) PackageLocation() string {
	if strings.TrimSpace(r.PackagePath) != "" {
		return r.PackagePath
	}
	return r.Execution.PackageDir
}

func compileWorkflowPackageDir(req CompileRequest) string {
	if strings.TrimSpace(req.OutputDir) != "" {
		return req.OutputDir
	}
	if strings.TrimSpace(req.SketchPath) == "" || strings.TrimSpace(req.FQBN) == "" {
		return ""
	}
	sketchDir := filepath.Dir(req.SketchPath)
	fqbnSuffix := strings.ReplaceAll(req.FQBN, ":", ".")
	return filepath.Join(sketchDir, "build", fqbnSuffix, "firmware-package")
}

func compileWorkflowReady(pkg firmware.FirmwarePackage, validation firmware.ValidationReport) bool {
	if validation.HasFailures() {
		return false
	}
	if err := pkg.Validate(); err != nil {
		return false
	}
	return true
}

func compileWorkflowBeginner(preflight string, compatibilitySummary string, exec CompileExecution, pkg firmware.FirmwarePackage, validation firmware.ValidationReport) string {
	parts := []string{}
	if strings.TrimSpace(preflight) != "" {
		parts = append(parts, preflight)
	}
	if strings.TrimSpace(compatibilitySummary) != "" {
		parts = append(parts, compatibilitySummary)
	}
	if strings.TrimSpace(exec.PackageDir) != "" {
		parts = append(parts, "firmware package generated")
	}
	if validation.Status != "" {
		parts = append(parts, validation.BeginnerSummary())
	}
	if len(parts) == 0 {
		return "compile workflow completed"
	}
	return strings.Join(parts, "; ")
}

func jsonMarshalIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func validateCompileRequest(req CompileRequest) error {
	if strings.TrimSpace(req.SketchPath) == "" {
		return errors.New("sketch path is required")
	}
	if strings.TrimSpace(req.FQBN) == "" {
		return errors.New("fqbn is required")
	}
	return nil
}
