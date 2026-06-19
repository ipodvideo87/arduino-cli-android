package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseArgsDefaultsToScan(t *testing.T) {
	mode, file, err := parseArgs([]string{"/tmp/tool"})
	require.NoError(t, err)
	require.Equal(t, modeScan, mode)
	require.Equal(t, "/tmp/tool", file)
}

func TestParseArgsCompatDefaultsToArduinoPackagesRoot(t *testing.T) {
	mode, file, err := parseArgs([]string{"compat"})
	require.NoError(t, err)
	require.Equal(t, modeCompat, mode)
	require.Empty(t, file)
}

func TestParseArgsCompatJSONWithExplicitRoot(t *testing.T) {
	mode, file, err := parseArgs([]string{"compat-json", "/tmp/packages"})
	require.NoError(t, err)
	require.Equal(t, modeCompatJSON, mode)
	require.Equal(t, "/tmp/packages", file)
}

func TestParseArgsValidateCompatDefaultsToArduinoPackagesRoot(t *testing.T) {
	mode, file, err := parseArgs([]string{"validate-compat"})
	require.NoError(t, err)
	require.Equal(t, modeValidateCompat, mode)
	require.Empty(t, file)
}

func TestParseArgsValidateCompatJSONWithExplicitRoot(t *testing.T) {
	mode, file, err := parseArgs([]string{"validate-compat-json", "/tmp/packages"})
	require.NoError(t, err)
	require.Equal(t, modeValidateCompatJSON, mode)
	require.Equal(t, "/tmp/packages", file)
}

func TestParseArgsRejectsUnknownMode(t *testing.T) {
	_, _, err := parseArgs([]string{"unknown", "/tmp/file"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown mode")
}
