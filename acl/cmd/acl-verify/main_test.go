package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/arduino/arduino-cli/acl/verifier"
)

// TestRootCmdList verifies that --list prints all check names without error.
func TestRootCmdList(t *testing.T) {
	cmd := rootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// --list writes to os.Stdout directly; capture via pipe.
	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	os.Stdout = w

	printCheckList()

	w.Close()
	os.Stdout = old

	var capBuf bytes.Buffer
	if _, readErr := capBuf.ReadFrom(r); readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	output := capBuf.String()

	for _, c := range verifier.All {
		if !strings.Contains(output, c.Name) {
			t.Errorf("--list output missing check name %q\noutput:\n%s", c.Name, output)
		}
	}
}

// TestRootCmdUnknownCheck verifies that requesting an unknown check name
// returns a descriptive error.
func TestRootCmdUnknownCheck(t *testing.T) {
	cmd := rootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--check", "this-check-does-not-exist"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown check name, got nil")
	}
	if !strings.Contains(err.Error(), "unknown check name") {
		t.Errorf("expected 'unknown check name' in error, got: %v", err)
	}
}

// TestEmitJSON verifies emitJSON-equivalent logic writes valid, well-structured
// JSON for a known set of results.
func TestEmitJSON(t *testing.T) {
	// Redirect stdout.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	results := []verifier.Result{
		{
			Name:    "fake-pass",
			Passed:  true,
			Code:    verifier.ExitOK,
			Message: "all good",
		},
		{
			Name:    "fake-fail",
			Passed:  false,
			Code:    verifier.ExitMissingDep,
			Message: "tool missing",
			Hint:    "install the tool",
		},
	}

	// Write JSON directly (avoiding os.Exit in emitJSON when code != OK).
	out := make([]jsonResult, len(results))
	for i, res := range results {
		out[i] = jsonResult{
			Name:    res.Name,
			Passed:  res.Passed,
			Code:    int(res.Code),
			Message: res.Message,
			Hint:    res.Hint,
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(out); encErr != nil {
		t.Fatalf("encode: %v", encErr)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("read: %v", readErr)
	}

	var parsed []jsonResult
	if jsonErr := json.Unmarshal(buf.Bytes(), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\n%s", jsonErr, buf.String())
	}
	if len(parsed) != 2 {
		t.Fatalf("expected 2 results, got %d", len(parsed))
	}

	pass := parsed[0]
	if pass.Name != "fake-pass" || !pass.Passed || pass.Code != 0 {
		t.Errorf("unexpected pass result: %+v", pass)
	}
	if pass.Hint != "" {
		t.Errorf("expected empty hint for passing result, got %q", pass.Hint)
	}

	fail := parsed[1]
	if fail.Name != "fake-fail" || fail.Passed || fail.Code != int(verifier.ExitMissingDep) {
		t.Errorf("unexpected fail result: %+v", fail)
	}
	if fail.Hint == "" {
		t.Error("expected non-empty hint for failing result")
	}
}

// TestPrintCheckList verifies that printCheckList includes all registered names.
func TestPrintCheckList(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	printCheckList()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	output := buf.String()
	for _, c := range verifier.All {
		if !strings.Contains(output, c.Name) {
			t.Errorf("printCheckList missing %q\noutput:\n%s", c.Name, output)
		}
		if !strings.Contains(output, c.Description) {
			t.Errorf("printCheckList missing description for %q\noutput:\n%s", c.Name, output)
		}
	}
}

// TestEmitText verifies that emitText shows PASS/FAIL and the summary line.
func TestEmitText(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	results := []verifier.Result{
		{Name: "a", Passed: true, Code: verifier.ExitOK, Message: "ok"},
		{Name: "b", Passed: false, Code: verifier.ExitFilesystem, Message: "bad", Hint: "fix it"},
	}
	emitText(results)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	output := buf.String()

	if !strings.Contains(output, "PASS") {
		t.Errorf("expected PASS in output: %s", output)
	}
	if !strings.Contains(output, "FAIL") {
		t.Errorf("expected FAIL in output: %s", output)
	}
	if !strings.Contains(output, "fix it") {
		t.Errorf("expected hint text in output: %s", output)
	}
	if !strings.Contains(output, "1/2 checks passed") {
		t.Errorf("expected summary '1/2 checks passed' in: %s", output)
	}
	if !strings.Contains(output, "1 failed") {
		t.Errorf("expected '1 failed' in summary: %s", output)
	}
}

// TestEmitTextAllPass verifies that the summary does not mention failures when
// all checks pass.
func TestEmitTextAllPass(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	results := []verifier.Result{
		{Name: "a", Passed: true, Code: verifier.ExitOK, Message: "ok"},
		{Name: "b", Passed: true, Code: verifier.ExitOK, Message: "also ok"},
	}
	emitText(results)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	output := buf.String()

	if strings.Contains(output, "failed") {
		t.Errorf("unexpected 'failed' in all-pass output: %s", output)
	}
	if !strings.Contains(output, "2/2 checks passed") {
		t.Errorf("expected '2/2 checks passed' in: %s", output)
	}
}

// TestJsonResultHintOmitEmpty verifies that the hint field is omitted from
// JSON when empty (omitempty).
func TestJsonResultHintOmitEmpty(t *testing.T) {
	r := jsonResult{
		Name:    "test",
		Passed:  true,
		Code:    0,
		Message: "ok",
		Hint:    "",
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hint") {
		t.Errorf("expected hint to be omitted when empty, got: %s", data)
	}
}

// TestJsonResultHintPresent verifies that the hint field appears when non-empty.
func TestJsonResultHintPresent(t *testing.T) {
	r := jsonResult{
		Name:    "test",
		Passed:  false,
		Code:    int(verifier.ExitMissingDep),
		Message: "missing",
		Hint:    "do something",
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hint") {
		t.Errorf("expected hint to be present, got: %s", data)
	}
	if !strings.Contains(string(data), "do something") {
		t.Errorf("expected hint value in JSON: %s", data)
	}
}
