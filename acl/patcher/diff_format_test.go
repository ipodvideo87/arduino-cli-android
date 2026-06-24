package patcher

import (
	"strings"
	"testing"
)

func TestWritePlanText_NeedsPatch(t *testing.T) {
	plans := []PatchPlan{
		{
			Path: "/tmp/gcc",
			Edits: []FieldEdit{
				{
					Field:    "PT_INTERP",
					Current:  "/lib/ld-linux-aarch64.so.1",
					Proposed: "/acl/runtime/ld-linux-aarch64.so.1",
					Reason:   "replace glibc loader with ACL loader",
				},
				{
					Field:    "RUNPATH",
					Current:  "",
					Proposed: "/acl/runtime",
					Reason:   "set RUNPATH to ACL runtime directory",
				},
			},
		},
	}

	out := PlanToString(plans, false)

	mustContain(t, out, "=== /tmp/gcc ===")
	mustContain(t, out, "--- PT_INTERP")
	mustContain(t, out, "+++ PT_INTERP")
	mustContain(t, out, "/lib/ld-linux-aarch64.so.1")
	mustContain(t, out, "/acl/runtime/ld-linux-aarch64.so.1")
	mustContain(t, out, "--- RUNPATH")
	mustContain(t, out, "<absent>")
	mustContain(t, out, "/acl/runtime")
	mustContain(t, out, "replace glibc loader with ACL loader")
	mustContain(t, out, "1 need patching")
}

func TestWritePlanText_AlreadyOK_VerboseShowsEntry(t *testing.T) {
	plans := []PatchPlan{
		{Path: "/tmp/native", Edits: nil},
	}

	outVerbose := PlanToString(plans, true)
	mustContain(t, outVerbose, "=== /tmp/native ===")
	mustContain(t, outVerbose, "[ok] no patching required")

	outQuiet := PlanToString(plans, false)
	if strings.Contains(outQuiet, "=== /tmp/native ===") {
		t.Error("quiet mode should omit already-OK entries")
	}
}

func TestWritePlanText_Skipped_VerboseShowsReason(t *testing.T) {
	plans := []PatchPlan{
		{Path: "/tmp/x86", Skipped: true, SkipReason: "unsupported machine EM_X86_64"},
	}

	outVerbose := PlanToString(plans, true)
	mustContain(t, outVerbose, "[skipped]")
	mustContain(t, outVerbose, "EM_X86_64")

	outQuiet := PlanToString(plans, false)
	if strings.Contains(outQuiet, "[skipped]") {
		t.Error("quiet mode should omit skipped entries")
	}
}

func TestWritePlanText_SummaryLine(t *testing.T) {
	plans := []PatchPlan{
		{
			Path: "/tmp/a",
			Edits: []FieldEdit{
				{Field: "PT_INTERP", Current: "/old", Proposed: "/new", Reason: "test"},
			},
		},
		{Path: "/tmp/b", Edits: nil},
		{Path: "/tmp/c", Skipped: true, SkipReason: "non-ELF"},
	}

	out := PlanToString(plans, false)
	mustContain(t, out, "Dry-run summary")
	mustContain(t, out, "3 file(s)")
	mustContain(t, out, "1 need patching")
	mustContain(t, out, "1 already OK")
	mustContain(t, out, "1 skipped")
}

func TestDisplayValue_EmptyIsAbsent(t *testing.T) {
	if displayValue("") != "<absent>" {
		t.Error("empty string should display as <absent>")
	}
	if displayValue("/foo") != "/foo" {
		t.Error("non-empty string should display as-is")
	}
}

func TestWritePlanText_RPATHRemoval(t *testing.T) {
	plans := []PatchPlan{
		{
			Path: "/tmp/lib.so",
			Edits: []FieldEdit{
				{
					Field:    "RPATH",
					Current:  "/some/old/path",
					Proposed: "",
					Reason:   "remove legacy RPATH",
				},
			},
		},
	}

	out := PlanToString(plans, false)
	mustContain(t, out, "--- RPATH")
	mustContain(t, out, "+++ RPATH")
	mustContain(t, out, "<absent>")
	mustContain(t, out, "/some/old/path")
}

func TestWritePlanText_MultipleFiles(t *testing.T) {
	plans := []PatchPlan{
		{
			Path: "/tmp/bin1",
			Edits: []FieldEdit{
				{Field: "PT_INTERP", Current: "/old/ld", Proposed: "/new/ld", Reason: "r1"},
			},
		},
		{
			Path: "/tmp/bin2",
			Edits: []FieldEdit{
				{Field: "RUNPATH", Current: "", Proposed: "/rt", Reason: "r2"},
			},
		},
	}

	out := PlanToString(plans, false)
	mustContain(t, out, "=== /tmp/bin1 ===")
	mustContain(t, out, "=== /tmp/bin2 ===")
	mustContain(t, out, "2 need patching")
}

func TestNeedsPatching(t *testing.T) {
	noEdit := PatchPlan{Path: "/a"}
	if noEdit.NeedsPatching() {
		t.Error("empty edits should not need patching")
	}

	skipped := PatchPlan{Path: "/b", Skipped: true, Edits: []FieldEdit{
		{Field: "PT_INTERP"},
	}}
	if skipped.NeedsPatching() {
		t.Error("skipped plan should never need patching")
	}

	withEdit := PatchPlan{Path: "/c", Edits: []FieldEdit{
		{Field: "PT_INTERP", Current: "/old", Proposed: "/new"},
	}}
	if !withEdit.NeedsPatching() {
		t.Error("plan with edits should need patching")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper
// ──────────────────────────────────────────────────────────────────────────────

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("output does not contain %q\n\nFull output:\n%s", needle, haystack)
	}
}
