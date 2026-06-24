// make_fixture_elf.go — generate ELF test fixtures using the same approach as
// gen_test_elfs.go but as a convenience wrapper for manual use.
//
// Run with:
//
//	go run acl/tests/testdata/make_fixture_elf.go
//
// The generated fixtures are written to acl/tests/testdata/elfs/.
// They are tiny synthetic ELF64/aarch64 binaries that satisfy debug/elf
// parsing but are NOT executable.
//
//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := filepath.Join("acl", "tests", "testdata", "elfs")
	// Delegate to gen_test_elfs logic — print usage pointing to the canonical generator.
	fmt.Fprintf(os.Stderr, "Use gen_test_elfs.go with an explicit output directory:\n")
	fmt.Fprintf(os.Stderr, "  go run acl/tests/testdata/gen_test_elfs.go %s\n", outDir)
	os.Exit(1)
}
