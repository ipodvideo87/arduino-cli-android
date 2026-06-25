// Package verifier implements pre-flight runtime assumption checks for the
// Android Compatibility Layer (ACL).
//
// # Overview
//
// Before any glibc-linked tool is launched on Android/Termux, the verifier
// suite confirms that the execution environment meets the minimum requirements:
//
//   - Termux PREFIX directory exists and its sub-directories are accessible
//   - SELinux is not in enforcing mode with a dangerous process context
//   - /proc/self/exe is readable (needed for self-path detection by many tools)
//   - Termux bin/lib directories do not have simultaneous W+X bits (W^X)
//   - patchelf is installed and executable (required for ACL apply-mode)
//   - A suitable dynamic linker is present under PREFIX
//
// # Exit Codes
//
// Each failed check carries an [ExitCode] that categorises the problem:
//
//	ExitOK (0)          All checks passed
//	ExitMissingDep (2)  Required tool or package is absent
//	ExitSELinux (3)     SELinux enforcing with dangerous context
//	ExitFilesystem (4)  PREFIX or sub-directory not accessible
//	ExitWX (5)          W^X restriction detected
//	ExitProcFS (6)      /proc restricted or inaccessible
//	ExitUnknown (99)    Unexpected internal error
//
// # Usage
//
// Run all checks and act on the worst exit code:
//
//	results := verifier.RunAll()
//	code := verifier.OverallExitCode(results)
//	for _, r := range results {
//	    fmt.Println(r)
//	}
//	os.Exit(int(code))
//
// Run a selected subset:
//
//	results := verifier.RunSelected([]string{"patchelf-present", "selinux-mode"})
//
// # Adding New Checks
//
// Append a [Check] to the [All] slice. Each Check must have a unique Name, a
// Description, and a Run function that returns a [Result]. The Run function
// must not call os.Exit; the caller is responsible for exit-code handling.
package verifier
