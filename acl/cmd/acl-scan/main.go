package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/arduino/arduino-cli/internal/acl/elfscan"
	"github.com/arduino/arduino-cli/internal/acl/toolcompat"
)

const (
	exitOK            = 0
	exitUsage         = 2
	exitOpen          = 3
	exitNotELF        = 4
	exitNoInterpreter = 5
	exitCompat        = 6
)

type mode int

const (
	modeScan mode = iota
	modeDeps
	modeInterpreter
	modeSymbols
	modeCompat
	modeCompatJSON
	modeValidateCompat
	modeValidateCompatJSON
)

func main() {
	runMode, file, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printUsage()
		os.Exit(exitUsage)
	}

	if runMode == modeCompat || runMode == modeCompatJSON || runMode == modeValidateCompat || runMode == modeValidateCompatJSON {
		root := file
		if root == "" {
			root, err = toolcompat.DefaultPackagesRoot()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(exitCompat)
			}
		}
		scanner := toolcompat.NewScanner()
		if runMode == modeValidateCompat || runMode == modeValidateCompatJSON {
			validation, err := scanner.Validate(root)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(exitCompat)
			}
			if runMode == modeValidateCompatJSON {
				data, err := validation.JSON()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(exitCompat)
				}
				fmt.Println(string(data))
			} else {
				fmt.Print(toolcompat.FormatValidationReport(validation))
			}
			if validation.Summary.Passed {
				os.Exit(exitOK)
			}
			os.Exit(exitCompat)
		}
		report, err := scanner.Scan(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(exitCompat)
		}
		if runMode == modeCompatJSON {
			data, err := report.JSON()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(exitCompat)
			}
			fmt.Println(string(data))
		} else {
			fmt.Print(toolcompat.FormatReport(report))
		}
		os.Exit(exitOK)
	}

	result, err := elfscan.Inspect(file)
	if err != nil {
		exitCode := exitOpen
		if errors.Is(err, elfscan.ErrNotELF) {
			exitCode = exitNotELF
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode)
	}

	switch runMode {
	case modeDeps:
		for _, lib := range result.ImportedLibraries {
			fmt.Println(lib)
		}
	case modeInterpreter:
		if result.Interpreter == "" {
			fmt.Fprintln(os.Stderr, "no PT_INTERP entry present")
			os.Exit(exitNoInterpreter)
		}
		fmt.Println(result.Interpreter)
	case modeSymbols:
		fmt.Println("symbol scanning is not implemented yet")
	default:
		fmt.Print(elfscan.Format(result))
	}
}

func parseArgs(args []string) (mode, string, error) {
	if len(args) == 0 {
		return modeScan, "", fmt.Errorf("acl-scan expects a file path, or a mode plus a file path")
	}
	if len(args) == 1 {
		switch args[0] {
		case "compat":
			return modeCompat, "", nil
		case "compat-json":
			return modeCompatJSON, "", nil
		case "validate-compat":
			return modeValidateCompat, "", nil
		case "validate-compat-json":
			return modeValidateCompatJSON, "", nil
		}
		return modeScan, args[0], nil
	}

	if len(args) != 2 {
		return modeScan, "", fmt.Errorf("acl-scan expects a file path, or a mode plus a file path")
	}

	switch args[0] {
	case "scan":
		return modeScan, args[1], nil
	case "deps":
		return modeDeps, args[1], nil
	case "interpreter":
		return modeInterpreter, args[1], nil
	case "symbols":
		return modeSymbols, args[1], nil
	case "compat":
		return modeCompat, args[1], nil
	case "compat-json":
		return modeCompatJSON, args[1], nil
	case "validate-compat":
		return modeValidateCompat, args[1], nil
	case "validate-compat-json":
		return modeValidateCompatJSON, args[1], nil
	default:
		return modeScan, "", fmt.Errorf("unknown mode %q", args[0])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: acl-scan [scan|deps|interpreter|symbols] <elf-file>")
	fmt.Fprintln(os.Stderr, "       acl-scan compat [packages-root]")
	fmt.Fprintln(os.Stderr, "       acl-scan compat-json [packages-root]")
	fmt.Fprintln(os.Stderr, "       acl-scan validate-compat [packages-root]")
	fmt.Fprintln(os.Stderr, "       acl-scan validate-compat-json [packages-root]")
}
