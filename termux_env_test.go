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
	t.Setenv("GCC_EXEC_PREFIX", "/tmp/gcc/")
	t.Setenv("COMPILER_PATH", "/tmp/compiler/")
	t.Setenv("GCC_SPECS", "/tmp/specs")
	t.Setenv("LIBRARY_PATH", "/tmp/libs")
	t.Setenv("CPATH", "/tmp/includes")
	t.Setenv("C_INCLUDE_PATH", "/tmp/c-headers")
	t.Setenv("CPLUS_INCLUDE_PATH", "/tmp/cxx-headers")
	t.Setenv("OBJC_INCLUDE_PATH", "/tmp/objc-headers")
	t.Setenv("OBJCPLUS_INCLUDE_PATH", "/tmp/objcxx-headers")
	t.Setenv("LD_RUN_PATH", "/tmp/runpath")

	removed := scrubUnsafeLoaderEnvForNativeTermux()

	require.Contains(t, removed, "LD_LIBRARY_PATH=/data/data/com.termux/files/usr/glibc/lib")
	require.Contains(t, removed, "LD_PRELOAD=/tmp/boom.so")
	require.Contains(t, removed, "LD_AUDIT=/tmp/audit.so")
	require.Contains(t, removed, "GCC_EXEC_PREFIX=/tmp/gcc/")
	require.Contains(t, removed, "COMPILER_PATH=/tmp/compiler/")
	require.Contains(t, removed, "GCC_SPECS=/tmp/specs")
	require.Contains(t, removed, "LIBRARY_PATH=/tmp/libs")
	require.Contains(t, removed, "CPATH=/tmp/includes")
	require.Contains(t, removed, "C_INCLUDE_PATH=/tmp/c-headers")
	require.Contains(t, removed, "CPLUS_INCLUDE_PATH=/tmp/cxx-headers")
	require.Contains(t, removed, "OBJC_INCLUDE_PATH=/tmp/objc-headers")
	require.Contains(t, removed, "OBJCPLUS_INCLUDE_PATH=/tmp/objcxx-headers")
	require.Contains(t, removed, "LD_RUN_PATH=/tmp/runpath")
	require.Empty(t, os.Getenv("LD_LIBRARY_PATH"))
	require.Empty(t, os.Getenv("LD_PRELOAD"))
	require.Empty(t, os.Getenv("LD_AUDIT"))
	require.Empty(t, os.Getenv("GCC_EXEC_PREFIX"))
	require.Empty(t, os.Getenv("COMPILER_PATH"))
	require.Empty(t, os.Getenv("GCC_SPECS"))
	require.Empty(t, os.Getenv("LIBRARY_PATH"))
	require.Empty(t, os.Getenv("CPATH"))
	require.Empty(t, os.Getenv("C_INCLUDE_PATH"))
	require.Empty(t, os.Getenv("CPLUS_INCLUDE_PATH"))
	require.Empty(t, os.Getenv("OBJC_INCLUDE_PATH"))
	require.Empty(t, os.Getenv("OBJCPLUS_INCLUDE_PATH"))
	require.Empty(t, os.Getenv("LD_RUN_PATH"))
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
