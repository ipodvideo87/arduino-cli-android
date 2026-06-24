package patcher

import (
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// Minimal ELF fixture builder
//
// We synthesise tiny aarch64 ELF files in-process so the tests have zero
// external dependencies and run on every platform (Linux, macOS, Android).
//
// Layout (all ELF64 little-endian):
//
//   Offset 0         : ELF64 header (64 bytes)
//   Offset 64        : Program headers  (3 × 56 = 168 bytes)
//   Offset 232       : PT_INTERP data (interp string + NUL, may be 0 bytes)
//   Offset 232+interpSz : .dynamic section (Dyn entries)
//   ...              : .dynstr section (string table)
//   ...              : Section header table (4 entries: NULL, .dynamic, .dynstr, .shstrtab)
//   ...              : .shstrtab (section name strings)
//
// The section headers let debug/elf reliably resolve DT_NEEDED, DT_RUNPATH,
// and DT_RPATH without needing to follow DT_STRTAB virtual addresses.
// ──────────────────────────────────────────────────────────────────────────────

type elfFixtureSpec struct {
	interp  string
	runpath string
	rpath   string
	libs    []string
	machine elf.Machine
}

// buildELF creates a minimal ELF64 file with the given dynamic fields.
func buildELF(t *testing.T, dir, name string, spec elfFixtureSpec) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data, err := makeELF(spec)
	if err != nil {
		t.Fatalf("buildELF %s: %v", name, err)
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
	return path
}

// makeELF assembles the ELF binary into a byte slice.
func makeELF(spec elfFixtureSpec) ([]byte, error) {
	const (
		elfHdrSz  = 64
		phEntrySz = 56
		shEntrySz = 64
		dynEntrySz = 16
		numPHdrs  = 3
	)

	le := binary.LittleEndian

	machine := spec.machine
	if machine == 0 {
		machine = elf.EM_AARCH64
	}

	// ── 1. Build .dynstr (string table) ──────────────────────────────────────
	// Layout: NUL byte, then each string NUL-terminated.
	var dynstr []byte
	dynstr = append(dynstr, 0) // index 0 is always empty

	addStr := func(s string) uint32 {
		if s == "" {
			return 0
		}
		idx := uint32(len(dynstr))
		dynstr = append(dynstr, append([]byte(s), 0)...)
		return idx
	}

	interpIdx := addStr(spec.interp)
	_ = interpIdx // used in PT_INTERP data, not in dynstr

	libIdxs := make([]uint32, len(spec.libs))
	for i, lib := range spec.libs {
		libIdxs[i] = addStr(lib)
	}
	runpathIdx := addStr(spec.runpath)
	rpathIdx := addStr(spec.rpath)

	// Section name string table (.shstrtab).
	var shstrtab []byte
	shstrtab = append(shstrtab, 0)
	addShStr := func(s string) uint32 {
		idx := uint32(len(shstrtab))
		shstrtab = append(shstrtab, append([]byte(s), 0)...)
		return idx
	}
	shNull := addShStr("")
	_ = shNull
	shDynamic := addShStr(".dynamic")
	shDynstr  := addShStr(".dynstr")
	shShstrtab := addShStr(".shstrtab")

	// ── 2. Build .dynamic section ────────────────────────────────────────────
	type dynEntry struct{ tag, val uint64 }
	var dynEntries []dynEntry

	for _, idx := range libIdxs {
		dynEntries = append(dynEntries, dynEntry{uint64(elf.DT_NEEDED), uint64(idx)})
	}
	if spec.runpath != "" {
		dynEntries = append(dynEntries, dynEntry{uint64(elf.DT_RUNPATH), uint64(runpathIdx)})
	}
	if spec.rpath != "" {
		dynEntries = append(dynEntries, dynEntry{uint64(elf.DT_RPATH), uint64(rpathIdx)})
	}
	dynEntries = append(dynEntries, dynEntry{uint64(elf.DT_NULL), 0})

	dynBytes := make([]byte, len(dynEntries)*dynEntrySz)
	for i, e := range dynEntries {
		le.PutUint64(dynBytes[i*dynEntrySz:], e.tag)
		le.PutUint64(dynBytes[i*dynEntrySz+8:], e.val)
	}

	// ── 3. File layout calculation ────────────────────────────────────────────
	phStart := uint64(elfHdrSz)
	phEnd   := phStart + uint64(numPHdrs)*phEntrySz // = 232

	interpDataOff := phEnd
	interpDataSz  := uint64(0)
	if spec.interp != "" {
		interpDataSz = uint64(len(spec.interp)) + 1
	}

	dynOff  := interpDataOff + interpDataSz
	dynSz   := uint64(len(dynBytes))
	strOff  := dynOff + dynSz
	strSz   := uint64(len(dynstr))
	shsOff  := strOff + strSz
	shsSz   := uint64(len(shstrtab))

	// Section header table starts after all section data, aligned to 8 bytes.
	shTableOff := shsOff + shsSz
	if shTableOff%8 != 0 {
		shTableOff += 8 - (shTableOff % 8)
	}

	numSHdrs := uint16(4) // NULL, .dynamic, .dynstr, .shstrtab
	shTableSz := uint64(numSHdrs) * shEntrySz
	fileSize  := shTableOff + shTableSz

	// ── 4. Program headers ────────────────────────────────────────────────────
	phBytes := make([]byte, uint64(numPHdrs)*phEntrySz)

	writePH := func(idx int, pt elf.ProgType, off, vaddr, filesz, memsz uint64,
		flags elf.ProgFlag, align uint64) {
		b := idx * phEntrySz
		le.PutUint32(phBytes[b:], uint32(pt))
		le.PutUint32(phBytes[b+4:], uint32(flags))
		le.PutUint64(phBytes[b+8:], off)
		le.PutUint64(phBytes[b+16:], vaddr)
		le.PutUint64(phBytes[b+24:], vaddr) // paddr = vaddr
		le.PutUint64(phBytes[b+32:], filesz)
		le.PutUint64(phBytes[b+40:], memsz)
		le.PutUint64(phBytes[b+48:], align)
	}

	if spec.interp != "" {
		writePH(0, elf.PT_INTERP, interpDataOff, interpDataOff,
			interpDataSz, interpDataSz, elf.PF_R, 1)
	}
	writePH(1, elf.PT_DYNAMIC, dynOff, dynOff, dynSz, dynSz,
		elf.PF_R|elf.PF_W, 8)
	writePH(2, elf.PT_LOAD, 0, 0, fileSize, fileSize,
		elf.PF_R|elf.PF_X, 0x1000)

	// ── 5. Section headers ────────────────────────────────────────────────────
	shBytes := make([]byte, uint64(numSHdrs)*shEntrySz)

	writeSH := func(idx int,
		nameOff uint32,
		shType elf.SectionType,
		flags elf.SectionFlag,
		addr, off, size uint64,
		link, info uint32,
		addralign, entsize uint64) {
		b := idx * shEntrySz
		le.PutUint32(shBytes[b:], nameOff)
		le.PutUint32(shBytes[b+4:], uint32(shType))
		le.PutUint64(shBytes[b+8:], uint64(flags))
		le.PutUint64(shBytes[b+16:], addr)
		le.PutUint64(shBytes[b+24:], off)
		le.PutUint64(shBytes[b+32:], size)
		le.PutUint32(shBytes[b+40:], link)
		le.PutUint32(shBytes[b+44:], info)
		le.PutUint64(shBytes[b+48:], addralign)
		le.PutUint64(shBytes[b+56:], entsize)
	}

	// SHN_UNDEF (index 0) — null section
	writeSH(0, 0, elf.SHT_NULL, 0, 0, 0, 0, 0, 0, 0, 0)
	// .dynamic — index 1, linked to .dynstr (index 2)
	writeSH(1, shDynamic, elf.SHT_DYNAMIC, elf.SHF_ALLOC|elf.SHF_WRITE,
		dynOff, dynOff, dynSz,
		2, 0, // link=.dynstr
		8, uint64(dynEntrySz))
	// .dynstr — index 2
	writeSH(2, shDynstr, elf.SHT_STRTAB, elf.SHF_ALLOC,
		strOff, strOff, strSz,
		0, 0, 1, 0)
	// .shstrtab — index 3
	writeSH(3, shShstrtab, elf.SHT_STRTAB, 0,
		shsOff, shsOff, shsSz,
		0, 0, 1, 0)

	// ── 6. ELF header ─────────────────────────────────────────────────────────
	hdr := make([]byte, elfHdrSz)
	copy(hdr[0:], []byte{0x7f, 'E', 'L', 'F'}) // magic
	hdr[4] = 2                                   // ELFCLASS64
	hdr[5] = 1                                   // ELFDATA2LSB
	hdr[6] = 1                                   // EV_CURRENT
	hdr[7] = 0                                   // ELFOSABI_SYSV
	le.PutUint16(hdr[16:], uint16(elf.ET_EXEC))
	le.PutUint16(hdr[18:], uint16(machine))
	le.PutUint32(hdr[20:], 1)           // e_version
	le.PutUint64(hdr[24:], 0)           // e_entry
	le.PutUint64(hdr[32:], phStart)     // e_phoff
	le.PutUint64(hdr[40:], shTableOff)  // e_shoff
	le.PutUint32(hdr[48:], 0)           // e_flags
	le.PutUint16(hdr[52:], elfHdrSz)    // e_ehsize
	le.PutUint16(hdr[54:], phEntrySz)   // e_phentsize
	le.PutUint16(hdr[56:], numPHdrs)    // e_phnum
	le.PutUint16(hdr[58:], shEntrySz)   // e_shentsize
	le.PutUint16(hdr[60:], numSHdrs)    // e_shnum
	le.PutUint16(hdr[62:], 3)           // e_shstrndx = index of .shstrtab

	// ── 7. Assemble ───────────────────────────────────────────────────────────
	buf := make([]byte, fileSize)
	copy(buf[0:], hdr)
	copy(buf[phStart:], phBytes)
	if spec.interp != "" {
		copy(buf[interpDataOff:], spec.interp)
		buf[interpDataOff+interpDataSz-1] = 0 // NUL terminator
	}
	copy(buf[dynOff:], dynBytes)
	copy(buf[strOff:], dynstr)
	copy(buf[shsOff:], shstrtab)
	copy(buf[shTableOff:], shBytes)

	return buf, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────────────────────────────────

func TestComputePlan_NoPatch_AlreadyAndroid(t *testing.T) {
	dir := t.TempDir()
	path := buildELF(t, dir, "native", elfFixtureSpec{
		interp: "/system/bin/linker64",
		libs:   []string{"liblog.so"},
	})

	plan, err := ComputePlan(path, PlanOptions{RuntimeDir: "/acl/runtime"})
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if plan.Skipped {
		t.Fatalf("unexpected skip: %s", plan.SkipReason)
	}
	if plan.NeedsPatching() {
		t.Errorf("expected no patching for native-Android binary, got edits: %v", plan.Edits)
	}
}

func TestComputePlan_NoPatch_NoInterp_NoGlibcLibs(t *testing.T) {
	dir := t.TempDir()
	// A shared library with no PT_INTERP and no glibc-world DT_NEEDED.
	path := buildELF(t, dir, "android.so", elfFixtureSpec{
		libs: []string{"liblog.so", "libc.so"}, // "libc.so" ≠ "libc.so.6"
	})

	plan, err := ComputePlan(path, PlanOptions{RuntimeDir: "/acl/runtime"})
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if plan.NeedsPatching() {
		t.Errorf("expected no patching, got: %v", plan.Edits)
	}
}

func TestComputePlan_SetInterpreterAndRunpath(t *testing.T) {
	dir := t.TempDir()
	path := buildELF(t, dir, "gcc", elfFixtureSpec{
		interp: "/lib/ld-linux-aarch64.so.1",
		libs:   []string{"libc.so.6", "libgcc_s.so.1"},
	})

	opts := PlanOptions{RuntimeDir: "/acl/runtime"}
	plan, err := ComputePlan(path, opts)
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if plan.Skipped {
		t.Fatalf("unexpected skip: %s", plan.SkipReason)
	}
	if !plan.NeedsPatching() {
		t.Fatal("expected patching to be required")
	}

	if !hasEditForField(plan, "PT_INTERP") {
		t.Error("expected PT_INTERP edit")
	}
	if !hasEditForField(plan, "RUNPATH") {
		t.Error("expected RUNPATH edit")
	}

	interpEdit := findEdit(plan, "PT_INTERP")
	if interpEdit.Current != "/lib/ld-linux-aarch64.so.1" {
		t.Errorf("PT_INTERP current = %q, want /lib/ld-linux-aarch64.so.1", interpEdit.Current)
	}
	if interpEdit.Proposed != "/acl/runtime/ld-linux-aarch64.so.1" {
		t.Errorf("PT_INTERP proposed = %q, want /acl/runtime/ld-linux-aarch64.so.1", interpEdit.Proposed)
	}

	rpathEdit := findEdit(plan, "RUNPATH")
	if rpathEdit.Proposed != "/acl/runtime" {
		t.Errorf("RUNPATH proposed = %q, want /acl/runtime", rpathEdit.Proposed)
	}
	if rpathEdit.Current != "" {
		t.Errorf("RUNPATH current = %q, want empty (absent)", rpathEdit.Current)
	}
}

func TestComputePlan_RunpathOnly_GlibcLibsNoInterp(t *testing.T) {
	dir := t.TempDir()
	// No PT_INTERP but glibc-world DT_NEEDED entries.
	path := buildELF(t, dir, "libfoo.so", elfFixtureSpec{
		libs: []string{"libc.so.6", "libm.so.6"},
	})

	opts := PlanOptions{RuntimeDir: "/acl/runtime"}
	plan, err := ComputePlan(path, opts)
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if !plan.NeedsPatching() {
		t.Fatal("expected RUNPATH edit for glibc-world library deps")
	}
	if hasEditForField(plan, "PT_INTERP") {
		t.Error("should NOT have a PT_INTERP edit when interp is absent")
	}
	if !hasEditForField(plan, "RUNPATH") {
		t.Error("expected RUNPATH edit")
	}
}

func TestComputePlan_SkipsNonAArch64(t *testing.T) {
	dir := t.TempDir()
	path := buildELF(t, dir, "x86_64", elfFixtureSpec{
		machine: elf.EM_X86_64,
		interp:  "/lib64/ld-linux-x86-64.so.2",
		libs:    []string{"libc.so.6"},
	})

	plan, err := ComputePlan(path, PlanOptions{RuntimeDir: "/acl/runtime"})
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if !plan.Skipped {
		t.Error("expected skip for non-aarch64 binary")
	}
	if !strings.Contains(plan.SkipReason, "EM_AARCH64") {
		t.Errorf("SkipReason %q should mention EM_AARCH64", plan.SkipReason)
	}
}

func TestComputePlan_SkipsNonELF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	plan, err := ComputePlan(path, PlanOptions{RuntimeDir: "/acl/runtime"})
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if !plan.Skipped {
		t.Error("expected skip for non-ELF file")
	}
}

func TestComputePlan_GCCLibexecSkipped(t *testing.T) {
	dir := t.TempDir()
	libexecDir := filepath.Join(dir, "libexec", "gcc", "aarch64-linux-gnu", "12.0.0")
	if err := os.MkdirAll(libexecDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := buildELF(t, libexecDir, "cc1", elfFixtureSpec{
		interp: "/lib/ld-linux-aarch64.so.1",
		libs:   []string{"libc.so.6"},
	})

	plan, err := ComputePlan(path, PlanOptions{RuntimeDir: "/acl/runtime"})
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if !plan.Skipped {
		t.Errorf("expected GCC libexec binary to be skipped; got edits: %v", plan.Edits)
	}
	if !strings.Contains(plan.SkipReason, "wrapper-launch") {
		t.Errorf("SkipReason %q should mention wrapper-launch", plan.SkipReason)
	}
}

func TestComputePlan_LegacyRPATH_AlsoSetsRunpath(t *testing.T) {
	dir := t.TempDir()
	// Binary with glibc interp + legacy RPATH.
	path := buildELF(t, dir, "legacy", elfFixtureSpec{
		interp: "/lib/ld-linux-aarch64.so.1",
		rpath:  "/some/old/path",
		libs:   []string{"libc.so.6"},
	})

	opts := PlanOptions{RuntimeDir: "/acl/runtime"}
	plan, err := ComputePlan(path, opts)
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if !plan.NeedsPatching() {
		t.Fatal("expected patching needed")
	}
	// When we set RUNPATH (via the PT_INTERP+RUNPATH path), patchelf --set-rpath
	// implicitly clears RPATH — so we should NOT also emit a separate RPATH edit.
	if hasEditForField(plan, "RUNPATH") && hasEditForField(plan, "RPATH") {
		t.Error("should not emit both RUNPATH and RPATH edits simultaneously")
	}
}

func TestComputePlan_LegacyRPATH_NoInterp_NoGlibcLibs(t *testing.T) {
	dir := t.TempDir()
	// Binary with only a legacy RPATH, no glibc interp, no glibc libs.
	path := buildELF(t, dir, "rpathonly", elfFixtureSpec{
		rpath: "/usr/lib",
		libs:  []string{"liblog.so"},
	})

	opts := PlanOptions{RuntimeDir: "/acl/runtime"}
	plan, err := ComputePlan(path, opts)
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if !plan.NeedsPatching() {
		t.Fatal("expected RPATH removal edit for binary with only legacy RPATH")
	}
	if !hasEditForField(plan, "RPATH") {
		t.Error("expected RPATH edit for legacy RPATH removal")
	}
	if hasEditForField(plan, "RUNPATH") {
		t.Error("should NOT have RUNPATH edit alongside RPATH removal when no RUNPATH is being set")
	}
	rpathEdit := findEdit(plan, "RPATH")
	if rpathEdit.Proposed != "" {
		t.Errorf("RPATH proposed should be empty (removal), got %q", rpathEdit.Proposed)
	}
}

func TestComputePlans_MultiFile(t *testing.T) {
	dir := t.TempDir()

	p1 := buildELF(t, dir, "bin1", elfFixtureSpec{
		interp: "/lib/ld-linux-aarch64.so.1",
		libs:   []string{"libc.so.6"},
	})
	p2 := buildELF(t, dir, "bin2", elfFixtureSpec{
		interp: "/system/bin/linker64",
		libs:   []string{"liblog.so"},
	})
	p3 := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(p3, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	plans := ComputePlans([]string{p1, p2, p3}, PlanOptions{RuntimeDir: "/acl/runtime"})
	if len(plans) != 3 {
		t.Fatalf("expected 3 plans, got %d", len(plans))
	}
	if !plans[0].NeedsPatching() {
		t.Error("plan[0] (glibc binary) should need patching")
	}
	if plans[1].NeedsPatching() {
		t.Error("plan[1] (native Android) should not need patching")
	}
	if !plans[2].Skipped {
		t.Error("plan[2] (script) should be skipped")
	}

	summary := Summarise(plans)
	if summary.NeedPatching != 1 {
		t.Errorf("NeedPatching = %d, want 1", summary.NeedPatching)
	}
	if summary.AlreadyOK != 1 {
		t.Errorf("AlreadyOK = %d, want 1", summary.AlreadyOK)
	}
	if summary.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", summary.Skipped)
	}
}

func TestSummarise_Empty(t *testing.T) {
	s := Summarise(nil)
	if s.Total != 0 || s.NeedPatching != 0 || s.AlreadyOK != 0 || s.Skipped != 0 {
		t.Errorf("unexpected summary for nil input: %+v", s)
	}
}

func TestCustomLoaderName(t *testing.T) {
	dir := t.TempDir()
	path := buildELF(t, dir, "bin", elfFixtureSpec{
		interp: "/lib/ld-linux-aarch64.so.1",
		libs:   []string{"libc.so.6"},
	})

	opts := PlanOptions{
		RuntimeDir: "/acl/rt",
		LoaderName: "ld-custom.so.1",
	}
	plan, err := ComputePlan(path, opts)
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	edit := findEdit(plan, "PT_INTERP")
	if edit.Proposed != "/acl/rt/ld-custom.so.1" {
		t.Errorf("proposed interp = %q, want /acl/rt/ld-custom.so.1", edit.Proposed)
	}
}

func TestPlanPath_PreservedExactly(t *testing.T) {
	dir := t.TempDir()
	path := buildELF(t, dir, "mybin", elfFixtureSpec{
		interp: "/lib/ld-linux-aarch64.so.1",
		libs:   []string{"libc.so.6"},
	})

	plan, err := ComputePlan(path, PlanOptions{RuntimeDir: "/acl/runtime"})
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	if plan.Path != path {
		t.Errorf("plan.Path = %q, want %q", plan.Path, path)
	}
}

func TestFieldEdit_ReasonIsNonEmpty(t *testing.T) {
	dir := t.TempDir()
	path := buildELF(t, dir, "bin", elfFixtureSpec{
		interp: "/lib/ld-linux-aarch64.so.1",
		libs:   []string{"libc.so.6"},
	})

	plan, err := ComputePlan(path, PlanOptions{RuntimeDir: "/acl/runtime"})
	if err != nil {
		t.Fatalf("ComputePlan: %v", err)
	}
	for _, edit := range plan.Edits {
		if strings.TrimSpace(edit.Reason) == "" {
			t.Errorf("edit for %s has empty Reason", edit.Field)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

func hasEditForField(plan PatchPlan, field string) bool {
	for _, e := range plan.Edits {
		if e.Field == field {
			return true
		}
	}
	return false
}

func findEdit(plan PatchPlan, field string) FieldEdit {
	for _, e := range plan.Edits {
		if e.Field == field {
			return e
		}
	}
	return FieldEdit{}
}
