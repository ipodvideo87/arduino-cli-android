// Package tests contains ACL integration-style tests.
//
// These tests exercise the ACL scanner, patcher, and verifier components
// against real fixture files in testdata/, including:
//
//   - shebang_fixtures_test.go — script shebang validation against fake
//     Termux PREFIX trees using the fixture scripts in testdata/scripts/.
//
// Run with:
//
//	go test ./acl/tests/...
package tests
