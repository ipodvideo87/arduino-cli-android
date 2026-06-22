package main

import (
	"os"
	"strings"
)

// scrubUnsafeLoaderEnvForNativeTermux removes dynamic loader variables that
// would otherwise leak into child processes and redirect them to the Termux
// glibc runtime.
func scrubUnsafeLoaderEnvForNativeTermux() []string {
	if !looksLikeNativeTermuxEnv(os.Environ()) {
		return nil
	}

	removed := make([]string, 0, 12)
	for _, key := range []string{
		"LD_LIBRARY_PATH",
		"LD_PRELOAD",
		"LD_AUDIT",
		"GCC_EXEC_PREFIX",
		"COMPILER_PATH",
		"GCC_SPECS",
		"LIBRARY_PATH",
		"CPATH",
		"C_INCLUDE_PATH",
		"CPLUS_INCLUDE_PATH",
		"OBJC_INCLUDE_PATH",
		"OBJCPLUS_INCLUDE_PATH",
		"LD_RUN_PATH",
	} {
		if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
			removed = append(removed, key+"="+value)
			_ = os.Unsetenv(key)
		}
	}
	return removed
}

func looksLikeNativeTermuxEnv(env []string) bool {
	values := make(map[string]string, len(env))
	for _, kv := range env {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		values[key] = value
	}

	if v := strings.TrimSpace(values["TERMUX_VERSION"]); v != "" {
		return true
	}
	if isNativeTermuxPrefix(values["TERMUX_PREFIX"]) || isNativeTermuxPrefix(values["TERMUX__PREFIX"]) {
		return true
	}
	if isNativeTermuxHome(values["TERMUX_HOME"]) || isNativeTermuxHome(values["TERMUX__HOME"]) {
		return true
	}
	if v := strings.TrimSpace(values["TERMUX__ROOTFS_DIR"]); v != "" && strings.Contains(v, "/data/data/com.termux/files") {
		return true
	}
	return false
}

func isNativeTermuxPrefix(value string) bool {
	value = strings.TrimSpace(value)
	return value == "/data/data/com.termux/files/usr"
}

func isNativeTermuxHome(value string) bool {
	value = strings.TrimSpace(value)
	return value == "/data/data/com.termux/files/home"
}
