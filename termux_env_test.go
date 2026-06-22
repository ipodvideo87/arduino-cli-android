package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScrubUnsafeLoaderEnvForNativeTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "0.118.0")
	t.Setenv("TERMUX_PREFIX", "/data/data/com.termux/files/usr")
	t.Setenv("TERMUX_HOME", "/data/data/com.termux/files/home")
	t.Setenv("LD_LIBRARY_PATH", "/data/data/com.termux/files/usr/glibc/lib")
	t.Setenv("LD_PRELOAD", "/tmp/boom.so")
	t.Setenv("LD_AUDIT", "/tmp/audit.so")

	removed := scrubUnsafeLoaderEnvForNativeTermux()

	require.Contains(t, removed, "LD_LIBRARY_PATH=/data/data/com.termux/files/usr/glibc/lib")
	require.Contains(t, removed, "LD_PRELOAD=/tmp/boom.so")
	require.Contains(t, removed, "LD_AUDIT=/tmp/audit.so")
	require.Empty(t, os.Getenv("LD_LIBRARY_PATH"))
	require.Empty(t, os.Getenv("LD_PRELOAD"))
	require.Empty(t, os.Getenv("LD_AUDIT"))
	require.Equal(t, "0.118.0", os.Getenv("TERMUX_VERSION"))
	require.True(t, strings.HasPrefix(os.Getenv("TERMUX_PREFIX"), "/data/data/com.termux/files"))
}

func TestScrubUnsafeLoaderEnvSkipsNonTermux(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "/tmp/termux-glibc")
	t.Setenv("LD_PRELOAD", "/tmp/boom.so")

	removed := scrubUnsafeLoaderEnvForNativeTermux()

	require.Empty(t, removed)
	require.Equal(t, "/tmp/termux-glibc", os.Getenv("LD_LIBRARY_PATH"))
	require.Equal(t, "/tmp/boom.so", os.Getenv("LD_PRELOAD"))
}
