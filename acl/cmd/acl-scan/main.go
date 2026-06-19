package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/arduino/arduino-cli/internal/acl/elfscan"
)

const (
	exitOK            = 0
	exitUsage         = 2
	exitOpen          = 3
	exitNotELF        = 4
	exitNoInterpreter = 5
)

type mode int

const (
	modeScan mode = iota
	modeDeps
	modeInterpreter
	modeSymbols
)

func main() {
	runMode, file, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printUsage()
		os.Exit(exitUsage)
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
	if len(args) == 1 {
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
	default:
		return modeScan, "", fmt.Errorf("unknown mode %q", args[0])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: acl-scan [scan|deps|interpreter|symbols] <elf-file>")
}
