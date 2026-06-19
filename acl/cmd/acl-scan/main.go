package main

import (
	"debug/elf"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: acl-scan <elf-file>")
		os.Exit(1)
	}

	file := os.Args[1]

	f, err := elf.Open(file)
	if err != nil {
		fmt.Printf("Not an ELF: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Println("===================================")
	fmt.Println("Android Compatibility Layer Scanner")
	fmt.Println("===================================")
	fmt.Println()

	fmt.Printf("File: %s\n", file)
	fmt.Printf("Class: %s\n", f.Class)
	fmt.Printf("Machine: %s\n", f.Machine)
	fmt.Printf("Type: %s\n", f.Type)
	fmt.Println()

	fmt.Println("Shared Libraries:")

	libs, err := f.ImportedLibraries()
	if err != nil {
		fmt.Println("  (unable to read)")
	} else {
		for _, lib := range libs {
			fmt.Printf("  - %s\n", lib)
		}
	}
}
