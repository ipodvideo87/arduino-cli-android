package acl

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/arduino/arduino-cli/commands"
	"github.com/arduino/arduino-cli/internal/acl/engine"
	"github.com/arduino/arduino-cli/internal/cli/arguments"
	"github.com/arduino/arduino-cli/internal/cli/instance"
	"github.com/arduino/arduino-cli/pkg/fqbn"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
	"github.com/spf13/cobra"
)

var workflowCompileRun = runWorkflowCompile

type workflowCompileOptions struct {
	details bool
	fqbn    string

	buildPath        string
	outputDir        string
	buildProperties  []string
	libraries        []string
	library          []string
	warnings         string
	verbose          bool
	quiet            bool
	clean            bool
	optimizeForDebug bool
	jobs             int32
}

func newWorkflowCompileCommand(srv rpc.ArduinoCoreServiceServer) *cobra.Command {
	opts := workflowCompileOptions{}
	cmd := &cobra.Command{
		Use:   "compile [sketch]",
		Short: "Run the ACL compile workflow through the engine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(opts.fqbn) == "" {
				return fmt.Errorf("--fqbn is required")
			}
			report, err := workflowCompileRun(cmd.Context(), srv, workflowCompileRequest{
				SketchPath:       args[0],
				FQBN:             opts.fqbn,
				BuildPath:        opts.buildPath,
				OutputDir:        opts.outputDir,
				BuildProperties:  append([]string(nil), opts.buildProperties...),
				Libraries:        append([]string(nil), opts.libraries...),
				Library:          append([]string(nil), opts.library...),
				Warnings:         opts.warnings,
				Verbose:          opts.verbose,
				Quiet:            opts.quiet,
				Clean:            opts.clean,
				OptimizeForDebug: opts.optimizeForDebug,
				Jobs:             opts.jobs,
			})
			if isJSON(cmd) {
				if err := writeJSON(cmd, report); err != nil {
					return err
				}
			} else {
				if err := writeCompileWorkflowReport(cmd, report, opts.details); err != nil {
					return err
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&opts.details, "details", false, "Show professional-level details")
	cmd.Flags().StringVar(&opts.fqbn, "fqbn", "", "Fully qualified board name")
	cmd.Flags().StringVar(&opts.buildPath, "build-path", "", "Path where to save compiled files")
	cmd.Flags().StringVar(&opts.outputDir, "output-dir", "", "Save workflow artifacts in this directory")
	cmd.Flags().StringSliceVar(&opts.buildProperties, "build-property", nil, "Override a build property with a custom value")
	cmd.Flags().StringSliceVar(&opts.buildProperties, "build-properties", nil, "List of custom build properties")
	cmd.Flags().StringSliceVar(&opts.libraries, "libraries", nil, "Path to a collection of libraries")
	cmd.Flags().StringSliceVar(&opts.library, "library", nil, "Path to a single library root folder")
	cmd.Flags().StringVar(&opts.warnings, "warnings", "none", "Warning level for gcc")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Turns on verbose mode")
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "Suppresses almost every output")
	cmd.Flags().BoolVar(&opts.clean, "clean", false, "Cleanup the build folder and do not use cached build")
	cmd.Flags().BoolVar(&opts.optimizeForDebug, "optimize-for-debug", false, "Optimize compile output for debug")
	cmd.Flags().Int32VarP(&opts.jobs, "jobs", "j", 0, "Max number of parallel jobs")
	return cmd
}

type workflowCompileRequest struct {
	SketchPath       string
	FQBN             string
	BuildPath        string
	OutputDir        string
	BuildProperties  []string
	Libraries        []string
	Library          []string
	Warnings         string
	Verbose          bool
	Quiet            bool
	Clean            bool
	OptimizeForDebug bool
	Jobs             int32
}

type workflowCompileRunner struct {
	srv      rpc.ArduinoCoreServiceServer
	instance *rpc.Instance
}

func runWorkflowCompile(ctx context.Context, srv rpc.ArduinoCoreServiceServer, req workflowCompileRequest) (engine.WorkflowReport, error) {
	if srv == nil {
		return engine.WorkflowReport{}, fmt.Errorf("compile service is not available")
	}
	sketchPath := arguments.InitSketchPath(req.SketchPath)
	loadResp, err := srv.LoadSketch(ctx, &rpc.LoadSketchRequest{SketchPath: sketchPath.String()})
	if err != nil {
		return engine.WorkflowReport{}, err
	}
	sketch := loadResp.GetSketch()
	if sketch == nil {
		return engine.WorkflowReport{}, fmt.Errorf("sketch data is unavailable")
	}
	profileName := ""
	if sketch.GetDefaultProfile() != nil {
		profileName = sketch.GetDefaultProfile().GetName()
	}
	inst, _ := instance.CreateAndInitWithProfile(ctx, srv, profileName, sketchPath)

	engineRequest := engine.CompileRequest{
		SketchPath:       sketchPath.String(),
		FQBN:             req.FQBN,
		BuildPath:        req.BuildPath,
		OutputDir:        req.OutputDir,
		BuildProperties:  append([]string(nil), req.BuildProperties...),
		Libraries:        append([]string(nil), req.Libraries...),
		Library:          append([]string(nil), req.Library...),
		Clean:            req.Clean,
		OptimizeForDebug: req.OptimizeForDebug,
		Warnings:         req.Warnings,
		Verbose:          req.Verbose,
		Quiet:            req.Quiet,
		Jobs:             req.Jobs,
	}

	wctx := engine.NewContext()
	wctx.CompileRequest = engineRequest
	wctx.CompileRunner = &workflowCompileRunner{
		srv:      srv,
		instance: inst,
	}

	return engine.New().Run(ctx, engine.CompileWorkflow(), wctx)
}

func (r *workflowCompileRunner) Run(ctx context.Context, req engine.CompileRequest, publish func(engine.Event)) (engine.CompileExecution, error) {
	if r == nil || r.srv == nil {
		return engine.CompileExecution{}, fmt.Errorf("compile server is not available")
	}
	if publish != nil {
		publish(engine.Event{Type: engine.EventStepProgress, Step: "compile", Progress: 0, Message: "compile request started"})
	}

	compileReq := &rpc.CompileRequest{
		Instance:                   r.instance,
		Fqbn:                       req.FQBN,
		SketchPath:                 req.SketchPath,
		BuildPath:                  req.BuildPath,
		ExportDir:                  req.OutputDir,
		BuildProperties:            append([]string(nil), req.BuildProperties...),
		Warnings:                   req.Warnings,
		Verbose:                    req.Verbose,
		Quiet:                      req.Quiet,
		Libraries:                  append([]string(nil), req.Libraries...),
		Library:                    append([]string(nil), req.Library...),
		Clean:                      req.Clean,
		OptimizeForDebug:           req.OptimizeForDebug,
		Jobs:                       req.Jobs,
		DoNotExpandBuildProperties: false,
	}

	server, _ := commands.CompilerServerToStreams(ctx, io.Discard, io.Discard, nil)
	if err := r.srv.Compile(compileReq, server); err != nil {
		return engine.CompileExecution{}, err
	}

	if publish != nil {
		publish(engine.Event{Type: engine.EventStepProgress, Step: "compile", Progress: 100, Message: "compile request completed"})
	}

	exec := engine.CompileExecution{
		SketchName:      filepath.Base(req.SketchPath),
		FQBN:            req.FQBN,
		BuildPath:       req.BuildPath,
		PackageDir:      workflowCompilePackageDir(req.SketchPath, req.FQBN),
		BuildProperties: map[string]string{},
	}
	return exec, nil
}

func workflowCompilePackageDir(sketchPath, fqbnString string) string {
	parsed, err := fqbn.Parse(fqbnString)
	if err != nil {
		fqbnString = strings.ReplaceAll(fqbnString, ":", ".")
	} else {
		fqbnString = strings.ReplaceAll(parsed.StringWithoutConfig(), ":", ".")
	}
	return filepath.Join(filepath.Dir(sketchPath), "build", fqbnString, "firmware-package")
}

func writeCompileWorkflowReport(cmd *cobra.Command, report engine.WorkflowReport, details bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "ACL Workflow Compile")
	fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", report.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", report.Status)
	fmt.Fprintln(cmd.OutOrStdout(), report.BeginnerSummary())
	if details {
		for _, detail := range report.ProfessionalDetails() {
			fmt.Fprintln(cmd.OutOrStdout(), detail)
		}
		if result, ok := report.Result.(engine.CompileWorkflowReport); ok {
			for _, detail := range result.ProfessionalDetails() {
				fmt.Fprintln(cmd.OutOrStdout(), detail)
			}
		}
	}
	return nil
}
