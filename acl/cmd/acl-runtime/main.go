package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	aclruntime "github.com/arduino/arduino-cli/internal/acl/runtime"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var root string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "acl-runtime",
		Short: "Manage installed ACL runtime packages",
	}

	manager := func() *aclruntime.Manager {
		return aclruntime.NewManager(root)
	}

	cmd.PersistentFlags().StringVar(&root, "root", "", "ACL runtime store root (defaults to ACL_RUNTIME_ROOT or the user config dir)")
	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Print JSON output")

	cmd.AddCommand(&cobra.Command{
		Use:     "install <package-dir>",
		Short:   "Install a runtime package into the ACL runtime store",
		Args:    cobra.ExactArgs(1),
		Example: "  acl-runtime install /tmp/acl-runtime-package",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := manager().InstallFromDir(args[0])
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(rt)
			}
			fmt.Printf("Installed runtime %s at %s\n", rt.ID, rt.Path)
			if rt.LastReport.Status != "" {
				fmt.Print(aclruntime.FormatValidation(rt.LastReport))
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Short:   "List installed runtimes",
		Args:    cobra.NoArgs,
		Example: "  acl-runtime list",
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimes, err := manager().Discover()
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(runtimes)
			}
			if len(runtimes) == 0 {
				fmt.Println("No runtimes installed.")
				return nil
			}
			for _, rt := range runtimes {
				active := ""
				if rt.Active {
					active = " (active)"
				}
				fmt.Printf("%s%s\n", rt.ID, active)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "status",
		Short:   "Show runtime store status",
		Args:    cobra.NoArgs,
		Example: "  acl-runtime status",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := manager().Status()
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(report)
			}
			fmt.Print(aclruntime.FormatStatus(report))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "validate <runtime-id>",
		Short:   "Validate an installed runtime",
		Args:    cobra.ExactArgs(1),
		Example: "  acl-runtime validate acl-runtime-aarch64-glibc-0.1",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := manager().Validate(args[0])
			if err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(report)
			}
			fmt.Print(aclruntime.FormatValidation(report))
			return nil
		},
	})

	cmd.AddCommand(selectCommand(manager, &jsonOutput))

	cmd.AddCommand(&cobra.Command{
		Use:     "activate <runtime-id>",
		Short:   "Mark a validated runtime as active",
		Args:    cobra.ExactArgs(1),
		Example: "  acl-runtime activate acl-runtime-aarch64-glibc-0.1",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager().Activate(args[0]); err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(map[string]string{"active_runtime_id": args[0]})
			}
			fmt.Printf("Activated runtime %s\n", args[0])
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "deactivate",
		Short:   "Clear the active runtime selection",
		Args:    cobra.NoArgs,
		Example: "  acl-runtime deactivate",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager().Deactivate(); err != nil {
				return err
			}
			if jsonOutput {
				return printJSON(map[string]any{"active_runtime_id": nil})
			}
			fmt.Println("Deactivated runtime selection")
			return nil
		},
	})

	return cmd
}

func selectCommand(manager func() *aclruntime.Manager, jsonOutput *bool) *cobra.Command {
	var arch string
	var abi string
	var activate bool

	cmd := &cobra.Command{
		Use:     "select",
		Short:   "Select the best compatible runtime",
		Args:    cobra.NoArgs,
		Example: "  acl-runtime select --arch x86_64 --abi android-x86_64",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := manager().SelectCompatible(aclruntime.SelectionRequest{
				Architecture: arch,
				ABI:          abi,
			})
			if err != nil {
				return err
			}
			if activate {
				if err := manager().Activate(rt.ID); err != nil {
					return err
				}
				rt.Active = true
			}
			if *jsonOutput {
				return printJSON(rt)
			}
			fmt.Printf("%s\n", rt.ID)
			fmt.Printf("  path: %s\n", filepath.Clean(rt.Path))
			fmt.Printf("  version: %s\n", rt.Manifest.RuntimeVersion)
			fmt.Printf("  arch: %s\n", rt.Manifest.Architecture)
			return nil
		},
	}

	cmd.Flags().StringVar(&arch, "arch", "", "Requested architecture")
	cmd.Flags().StringVar(&abi, "abi", "", "Requested ABI")
	cmd.Flags().BoolVar(&activate, "activate", false, "Activate the selected runtime")
	return cmd
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
