package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGoldenFiles verifies that the golden JSON fixtures in testdata/golden/
// are well-formed and schema-compliant (i.e. they decode into Report without
// error and have a non-empty schema_version).
func TestGoldenFiles(t *testing.T) {
	goldenDir := filepath.Join("testdata", "golden")
	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("ReadDir %s: %v", goldenDir, err)
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(goldenDir, name))
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}

			var r Report
			if err := json.Unmarshal(data, &r); err != nil {
				t.Fatalf("unmarshal %s: %v", name, err)
			}

			if r.SchemaVersion == "" {
				t.Errorf("%s: schema_version is empty", name)
			}

			// Entries that claim to be scripts must have patch_class set correctly.
			for i, entry := range r.Entries {
				if entry.Category == CategoryScript {
					if entry.PatchClass != PatchClassScriptNoELFPatch && entry.PatchClass != "" {
						t.Errorf("%s entry[%d]: script with unexpected patch_class %q",
							name, i, entry.PatchClass)
					}
					// If interpreter_status is present, status must be a known value.
					if entry.InterpreterStatus != nil {
						switch entry.InterpreterStatus.Status {
						case InterpreterFound, InterpreterMissing, InterpreterRemapped:
							// OK
						default:
							t.Errorf("%s entry[%d]: unknown interpreter_status.status %q",
								name, i, entry.InterpreterStatus.Status)
						}
					}
				}
			}

			// Summary totals must be consistent.
			if r.Summary.Total != len(r.Entries) {
				t.Errorf("%s: summary.total=%d but len(entries)=%d",
					name, r.Summary.Total, len(r.Entries))
			}
		})
	}
}
