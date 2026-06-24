// gen_test_elfs.go — standalone fixture generator called by integration tests.
//
// Usage:
//
//	go run acl/tests/testdata/gen_test_elfs.go <output-dir>
//
// Writes the following fixtures into <output-dir>:
//
//	needs-patch.elf   — glibc interp + glibc DT_NEEDED          → must be patched
//	already-ok.elf    — Android linker64 interp                  → no patch needed
//	glibc-rpath.elf   — glibc interp + legacy RPATH              → must be patched
//	x86_64.elf        — x86_64 machine                           → skipped (wrong arch)
//	libexec/gcc/aarch64-linux-gnu/12/cc1 — GCC libexec path      → skipped (wrapper-launch)
//
//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gen_test_elfs.go <output-dir>")
		os.Exit(1)
	}
	outDir := os.Args[1]

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatalf("mkdir %s: %v", outDir, err)
	}

	type spec struct {
		name    string
		interp  string
		runpath string
		rpath   string
		libs    []string
		machX86 bool
	}

	fixtures := []spec{
		{
			name:   "needs-patch.elf",
			interp: "/lib/ld-linux-aarch64.so.1",
			libs:   []string{"libc.so.6", "libgcc_s.so.1"},
		},
		{
			name:   "already-ok.elf",
			interp: "/system/bin/linker64",
			libs:   []string{"liblog.so"},
		},
		{
			name:   "glibc-rpath.elf",
			interp: "/lib/ld-linux-aarch64.so.1",
			rpath:  "/usr/lib:/usr/local/lib",
			libs:   []string{"libc.so.6"},
		},
		{
			name:    "x86_64.elf",
			interp:  "/lib64/ld-linux-x86-64.so.2",
			libs:    []string{"libc.so.6"},
			machX86: true,
		},
		{
			name:   filepath.Join("libexec", "gcc", "aarch64-linux-gnu", "12", "cc1"),
			interp: "/lib/ld-linux-aarch64.so.1",
			libs:   []string{"libc.so.6"},
		},
	}

	for _, f := range fixtures {
		path := filepath.Join(outDir, f.name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		data, err := buildELF(f.interp, f.runpath, f.rpath, f.libs, f.machX86)
		if err != nil {
			fatalf("build %s: %v", f.name, err)
		}
		if err := os.WriteFile(path, data, 0o755); err != nil {
			fatalf("write %s: %v", path, err)
		}
		fmt.Printf("wrote %s\n", path)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// ELF constants we need (avoid importing debug/elf in a //go:build ignore file
// to keep the generator self-contained).
const (
	elfMag0 = 0x7f
	elfMag1 = 'E'
	elfMag2 = 'L'
	elfMag3 = 'F'

	elfClass64   = 2
	elfData2LSB  = 1
	elfVersion   = 1
	elfOSABINone = 0

	etExec = 2

	emAArch64 = 0xB7
	emX86_64  = 0x3E

	ptLoad    = 1
	ptDynamic = 2
	ptInterp  = 3

	pfX = 1
	pfW = 2
	pfR = 4

	shtNull    = 0
	shtDynamic = 6
	shtStrtab  = 3

	shfAlloc = 2
	shfWrite = 1

	dtNeeded  = 5
	dtRpath   = 15
	dtStrtab  = 23
	dtStrsz   = 10
	dtRunpath = 29
	dtNull    = 0
)

// buildELF assembles a minimal ELF64 binary with section headers so that
// debug/elf can reliably read DT_NEEDED, DT_RUNPATH, and DT_RPATH.
func buildELF(interp, runpath, rpath string, libs []string, machX86 bool) ([]byte, error) {
	const (
		elfHdrSz   = 64
		phEntrySz  = 56
		shEntrySz  = 64
		dynEntrySz = 16
		numPHdrs   = 3
		numSHdrs   = 4 // NULL, .dynamic, .dynstr, .shstrtab
	)

	le := binary.LittleEndian

	machCode := uint16(emAArch64)
	if machX86 {
		machCode = emX86_64
	}

	// ── String table (.dynstr) ────────────────────────────────────────────────
	var dynstr []byte
	dynstr = append(dynstr, 0) // index 0 = empty

	addStr := func(s string) uint32 {
		if s == "" {
			return 0
		}
		idx := uint32(len(dynstr))
		dynstr = append(dynstr, append([]byte(s), 0)...)
		return idx
	}

	libIdxs := make([]uint32, len(libs))
	for i, lib := range libs {
		libIdxs[i] = addStr(lib)
	}
	runpathIdx := addStr(runpath)
	rpathIdx := addStr(rpath)

	// ── Section name string table (.shstrtab) ─────────────────────────────────
	var shstrtab []byte
	shstrtab = append(shstrtab, 0)
	addShStr := func(s string) uint32 {
		idx := uint32(len(shstrtab))
		shstrtab = append(shstrtab, append([]byte(s), 0)...)
		return idx
	}
	_ = addShStr("")
	shDynamicName := addShStr(".dynamic")
	shDynstrName  := addShStr(".dynstr")
	shShstrtabName := addShStr(".shstrtab")

	// ── .dynamic section ──────────────────────────────────────────────────────
	type dynEnt struct{ tag, val uint64 }
	var dynEntries []dynEnt

	for _, idx := range libIdxs {
		dynEntries = append(dynEntries, dynEnt{dtNeeded, uint64(idx)})
	}
	if runpath != "" {
		dynEntries = append(dynEntries, dynEnt{dtRunpath, uint64(runpathIdx)})
	}
	if rpath != "" {
		dynEntries = append(dynEntries, dynEnt{dtRpath, uint64(rpathIdx)})
	}
	dynEntries = append(dynEntries, dynEnt{dtNull, 0})

	dynBytes := make([]byte, len(dynEntries)*dynEntrySz)
	for i, e := range dynEntries {
		le.PutUint64(dynBytes[i*dynEntrySz:], e.tag)
		le.PutUint64(dynBytes[i*dynEntrySz+8:], e.val)
	}

	// ── File layout ───────────────────────────────────────────────────────────
	phStart := uint64(elfHdrSz)
	phEnd   := phStart + uint64(numPHdrs)*phEntrySz

	interpOff := phEnd
	interpSz  := uint64(0)
	if interp != "" {
		interpSz = uint64(len(interp)) + 1
	}

	dynOff  := interpOff + interpSz
	dynSz   := uint64(len(dynBytes))
	strOff  := dynOff + dynSz
	strSz   := uint64(len(dynstr))
	shsOff  := strOff + strSz
	shsSz   := uint64(len(shstrtab))

	shTableOff := shsOff + shsSz
	if shTableOff%8 != 0 {
		shTableOff += 8 - (shTableOff % 8)
	}
	shTableSz := uint64(numSHdrs) * shEntrySz
	fileSize  := shTableOff + shTableSz

	// ── Program headers ───────────────────────────────────────────────────────
	phBytes := make([]byte, uint64(numPHdrs)*phEntrySz)

	writePH := func(idx int, pt uint32, off, vaddr, filesz, memsz uint64, flags uint32, align uint64) {
		b := idx * phEntrySz
		le.PutUint32(phBytes[b:], pt)
		le.PutUint32(phBytes[b+4:], flags)
		le.PutUint64(phBytes[b+8:], off)
		le.PutUint64(phBytes[b+16:], vaddr)
		le.PutUint64(phBytes[b+24:], vaddr)
		le.PutUint64(phBytes[b+32:], filesz)
		le.PutUint64(phBytes[b+40:], memsz)
		le.PutUint64(phBytes[b+48:], align)
	}

	if interp != "" {
		writePH(0, ptInterp, interpOff, interpOff, interpSz, interpSz, pfR, 1)
	}
	writePH(1, ptDynamic, dynOff, dynOff, dynSz, dynSz, pfR|pfW, 8)
	writePH(2, ptLoad, 0, 0, fileSize, fileSize, pfR|pfX, 0x1000)

	// ── Section headers ───────────────────────────────────────────────────────
	shBytes := make([]byte, uint64(numSHdrs)*shEntrySz)

	writeSH := func(idx int, nameOff uint32, shType uint32, flags uint64,
		addr, off, size uint64, link, info uint32, addralign, entsize uint64) {
		b := idx * shEntrySz
		le.PutUint32(shBytes[b:], nameOff)
		le.PutUint32(shBytes[b+4:], shType)
		le.PutUint64(shBytes[b+8:], flags)
		le.PutUint64(shBytes[b+16:], addr)
		le.PutUint64(shBytes[b+24:], off)
		le.PutUint64(shBytes[b+32:], size)
		le.PutUint32(shBytes[b+40:], link)
		le.PutUint32(shBytes[b+44:], info)
		le.PutUint64(shBytes[b+48:], addralign)
		le.PutUint64(shBytes[b+56:], entsize)
	}

	writeSH(0, 0, shtNull, 0, 0, 0, 0, 0, 0, 0, 0)
	writeSH(1, shDynamicName, shtDynamic, shfAlloc|shfWrite,
		dynOff, dynOff, dynSz, 2, 0, 8, dynEntrySz)
	writeSH(2, shDynstrName, shtStrtab, shfAlloc,
		strOff, strOff, strSz, 0, 0, 1, 0)
	writeSH(3, shShstrtabName, shtStrtab, 0,
		shsOff, shsOff, shsSz, 0, 0, 1, 0)

	// ── ELF header ────────────────────────────────────────────────────────────
	hdr := make([]byte, elfHdrSz)
	hdr[0], hdr[1], hdr[2], hdr[3] = elfMag0, elfMag1, elfMag2, elfMag3
	hdr[4] = elfClass64
	hdr[5] = elfData2LSB
	hdr[6] = elfVersion
	hdr[7] = elfOSABINone
	le.PutUint16(hdr[16:], etExec)
	le.PutUint16(hdr[18:], machCode)
	le.PutUint32(hdr[20:], 1)
	le.PutUint64(hdr[24:], 0)
	le.PutUint64(hdr[32:], phStart)
	le.PutUint64(hdr[40:], shTableOff)
	le.PutUint32(hdr[48:], 0)
	le.PutUint16(hdr[52:], elfHdrSz)
	le.PutUint16(hdr[54:], phEntrySz)
	le.PutUint16(hdr[56:], numPHdrs)
	le.PutUint16(hdr[58:], shEntrySz)
	le.PutUint16(hdr[60:], numSHdrs)
	le.PutUint16(hdr[62:], 3) // e_shstrndx = index of .shstrtab

	// ── Assemble ──────────────────────────────────────────────────────────────
	buf := make([]byte, fileSize)
	copy(buf[0:], hdr)
	copy(buf[phStart:], phBytes)
	if interp != "" {
		copy(buf[interpOff:], interp)
		buf[interpOff+interpSz-1] = 0
	}
	copy(buf[dynOff:], dynBytes)
	copy(buf[strOff:], dynstr)
	copy(buf[shsOff:], shstrtab)
	copy(buf[shTableOff:], shBytes)

	return buf, nil
}
