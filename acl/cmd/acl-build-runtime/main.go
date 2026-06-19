package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	aclbuilder "github.com/arduino/arduino-cli/internal/acl/builder"
	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
	"github.com/spf13/cobra"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if err := validateRawArgs(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	cmd := newRootCommand(stdout, stderr)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		if isUsageError(err) {
			fmt.Fprintln(stderr, err)
			return 2
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func validateRawArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--lib":
			if i+1 >= len(args) {
				return usageError("missing --lib")
			}
			if strings.TrimSpace(args[i+1]) == "" {
				return usageError("empty --lib value")
			}
			i++
		case "--lib=":
			return usageError("empty --lib value")
		default:
			if strings.HasPrefix(args[i], "--lib=") && strings.TrimSpace(strings.TrimPrefix(args[i], "--lib=")) == "" {
				return usageError("empty --lib value")
			}
		}
	}
	return nil
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	var runtimeName string
	var runtimeID string
	var runtimeVersion string
	var arch string
	var abi []string
	var compatibility string
	var loader string
	var libs []string
	var output string
	var createdAt string
	var source string
	var sourceVersion string
	var gitCommit string
	var builderName string
	var notes string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:           "acl-build-runtime",
		Short:         "Build an ACL runtime package from runtime assets",
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: strings.Join([]string{
			"  acl-build-runtime \\",
			"    --name acl-runtime-aarch64 \\",
			"    --version 0.1.0 \\",
			"    --arch aarch64 \\",
			"    --abi android-aarch64 \\",
			"    --compatibility experimental \\",
			"    --loader /path/to/ld-linux-aarch64.so.1 \\",
			"    --lib /path/to/libc.so.6 \\",
			"    --lib /path/to/libdl.so.2 \\",
			"    --output /tmp/acl-runtime-package",
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			createdAt, err = resolveCreatedAt(createdAt)
			if err != nil {
				return err
			}
			if err := validateBuildFlags(runtimeID, runtimeVersion, arch, abi, compatibility, loader, libs, output); err != nil {
				return err
			}
			spec := aclbuilder.PackageSpec{
				RuntimeName:        runtimeName,
				RuntimeID:          runtimeID,
				RuntimeVersion:     runtimeVersion,
				Architecture:       arch,
				SupportedABIs:      abi,
				CompatibilityLevel: compatibility,
				CreatedAt:          createdAt,
				Loader: aclbuilder.SourceAsset{
					Name:         filepath.Base(loader),
					SourcePath:   loader,
					RelativePath: filepath.Join(aclbuilder.DefaultLoaderDir, filepath.Base(loader)),
					Kind:         "loader",
					Required:     true,
				},
				Libraries: buildLibraryAssets(libs),
				Build: aclruntime.BuildInfo{
					Tool:          "acl-build-runtime",
					Builder:       firstNonEmpty(builderName, filepath.Base(os.Args[0])),
					Source:        source,
					SourceVersion: sourceVersion,
					GitCommit:     gitCommit,
					GoVersion:     runtime.Version(),
					BuiltAt:       createdAt,
					HostOS:        runtime.GOOS,
					HostArch:      runtime.GOARCH,
					Notes:         notes,
				},
			}

			builder := aclbuilder.NewBuilder()
			result, err := builder.Package(output, spec)
			if err != nil {
				return err
			}
			if err := builder.Verify(output); err != nil {
				return err
			}

			if jsonOutput {
				return printJSON(result)
			}
			fmt.Fprintf(stdout, "runtime_id: %s\n", result.RuntimeID)
			fmt.Fprintf(stdout, "output: %s\n", result.PackageDir)
			fmt.Fprintf(stdout, "loader: %s\n", result.Manifest.Loader.Path)
			fmt.Fprintf(stdout, "libraries: %d\n", len(result.Manifest.Libraries))
			fmt.Fprintln(stdout, "verified: yes")
			return nil
		},
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	cmd.Flags().StringVar(&runtimeName, "name", "", "Human-readable runtime name")
	cmd.Flags().StringVar(&runtimeID, "id", "", "Explicit runtime ID override")
	cmd.Flags().StringVar(&runtimeID, "runtime-id", "", "Explicit runtime ID override")
	cmd.Flags().StringVar(&runtimeVersion, "version", "", "Runtime version")
	cmd.Flags().StringVar(&arch, "arch", "", "Runtime architecture")
	cmd.Flags().StringSliceVar(&abi, "abi", nil, "Supported ABI (repeatable)")
	cmd.Flags().StringVar(&compatibility, "compatibility", "", "Compatibility level")
	cmd.Flags().StringVar(&loader, "loader", "", "Path to the runtime loader")
	cmd.Flags().StringSliceVar(&libs, "lib", nil, "Path to a runtime library (repeatable)")
	cmd.Flags().StringVar(&output, "output", "", "Output directory for the runtime package")
	cmd.Flags().StringVar(&createdAt, "created-at", "", "Package creation time in RFC3339 format")
	cmd.Flags().StringVar(&source, "source", "", "Source identifier for build metadata")
	cmd.Flags().StringVar(&sourceVersion, "source-version", "", "Source version for build metadata")
	cmd.Flags().StringVar(&gitCommit, "git-commit", "", "Git commit for build metadata")
	cmd.Flags().StringVar(&builderName, "builder", "", "Builder name for metadata")
	cmd.Flags().StringVar(&notes, "notes", "", "Free-form build notes")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print JSON output")

	cmd.AddCommand(newVerifyCommand(&jsonOutput))
	return cmd
}

func newVerifyCommand(jsonOutput *bool) *cobra.Command {
	return &cobra.Command{
		Use:           "verify <package-dir>",
		Short:         "Verify an existing ACL runtime package",
		Args:          cobra.ExactArgs(1),
		Example:       "  acl-build-runtime verify /tmp/acl-runtime-package",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aclbuilder.NewBuilder().Verify(args[0]); err != nil {
				return err
			}
			if *jsonOutput {
				return printJSON(map[string]string{
					"package_dir": args[0],
					"status":      "ok",
				})
			}
			fmt.Printf("Verified runtime package at %s\n", args[0])
			return nil
		},
	}
}

func validateBuildFlags(runtimeID, runtimeVersion, arch string, abi []string, compatibility, loader string, libs []string, output string) error {
	switch {
	case strings.TrimSpace(runtimeVersion) == "":
		return usageError("missing --version")
	case strings.TrimSpace(arch) == "":
		return usageError("missing --arch")
	case len(abi) == 0:
		return usageError("missing --abi")
	case strings.TrimSpace(compatibility) == "":
		return usageError("missing --compatibility")
	case strings.TrimSpace(loader) == "":
		return usageError("missing --loader")
	case len(libs) == 0:
		return usageError("missing --lib")
	case strings.TrimSpace(output) == "":
		return usageError("missing --output")
	}

	if strings.TrimSpace(runtimeID) != "" && filepath.Base(runtimeID) != runtimeID {
		return usageError("invalid --id")
	}

	seen := map[string]struct{}{}
	for _, lib := range libs {
		if strings.TrimSpace(lib) == "" {
			return usageError("empty --lib value")
		}
		if _, ok := seen[lib]; ok {
			return usageError("duplicate --lib path")
		}
		seen[lib] = struct{}{}
	}
	return nil
}

func resolveCreatedAt(createdAt string) (string, error) {
	if strings.TrimSpace(createdAt) != "" {
		return createdAt, nil
	}
	if sourceDateEpoch := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); sourceDateEpoch != "" {
		seconds, err := strconv.ParseInt(sourceDateEpoch, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid SOURCE_DATE_EPOCH %q", sourceDateEpoch)
		}
		return time.Unix(seconds, 0).UTC().Format(time.RFC3339), nil
	}
	return time.Now().UTC().Format(time.RFC3339), nil
}

type usageErr struct {
	message string
}

func (e usageErr) Error() string {
	return e.message
}

func usageError(message string) error {
	return usageErr{message: message}
}

func isUsageError(err error) bool {
	_, ok := err.(usageErr)
	return ok
}

func buildLibraryAssets(paths []string) []aclbuilder.SourceAsset {
	assets := make([]aclbuilder.SourceAsset, 0, len(paths))
	for _, path := range paths {
		base := filepath.Base(path)
		assets = append(assets, aclbuilder.SourceAsset{
			Name:         base,
			SourcePath:   path,
			RelativePath: filepath.Join(aclbuilder.DefaultLibraryDir, base),
			Kind:         "library",
			Required:     true,
		})
	}
	return assets
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
