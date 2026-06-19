#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/runtime.sh"

runtime_dir="$(acl::runtime_investigation_dir)"
required_files_text="$(acl::runtime_expected_files)"
required_files=()
while IFS= read -r file; do
	[[ -n "$file" ]] || continue
	required_files+=("$file")
done <<<"$required_files_text"

markers_text="$(acl::runtime_absolute_path_markers)"
markers=()
while IFS= read -r marker; do
	[[ -n "$marker" ]] || continue
	markers+=("$marker")
done <<<"$markers_text"

status=0
warned=0
experimental=0

echo "Runtime relocation report"
echo "Runtime directory: $runtime_dir"

if [[ ! -d "$runtime_dir" ]]; then
	echo "FAIL: runtime directory is missing"
	exit 1
fi

missing=()
for file in "${required_files[@]}"; do
	path="$runtime_dir/$file"
	if [[ ! -e "$path" ]]; then
		missing+=("$file")
	fi
done

if (( ${#missing[@]} > 0 )); then
	echo "FAIL: missing required runtime files: ${missing[*]}"
	status=1
fi

for file in "${required_files[@]}"; do
	path="$runtime_dir/$file"
	echo
	echo "File: $file"
	if [[ ! -e "$path" ]]; then
		echo "  exists: no"
		continue
	fi

	echo "  exists: yes"
	file_output="$(file "$path" 2>/dev/null || true)"
	echo "  file: ${file_output:-unknown}"

	if ! scan_output="$(acl::run_scanner scan "$path" 2>&1)"; then
		echo "  FAIL: ELF scan failed"
		echo "$scan_output" | sed 's/^/  /'
		status=1
		continue
	fi

	echo "$scan_output" | sed 's/^/  /'

	suspicious_found=()
	if command -v strings >/dev/null 2>&1; then
		strings_output="$(strings "$path" 2>/dev/null | sort -u || true)"
		while IFS= read -r line; do
			[[ -n "$line" ]] || continue
			for marker in "${markers[@]}"; do
				if [[ "$line" == *"$marker"* ]]; then
					suspicious_found+=("$line")
					break
				fi
			done
		done <<<"$strings_output"
	fi

	if (( ${#suspicious_found[@]} > 0 )); then
		warned=1
		experimental=1
		echo "  WARN: suspicious absolute paths found"
		for line in "${suspicious_found[@]}"; do
			echo "    - $line"
		done
	else
		echo "  PASS: no suspicious absolute paths"
	fi

	if grep -q 'Interpreter: /data/data/com.termux/files/usr/glibc/lib/ld-linux-aarch64.so.1' <<<"$scan_output"; then
		experimental=1
	fi
done

echo
if (( status != 0 )); then
	echo "FAIL: runtime relocation is not safe yet"
elif (( warned != 0 )); then
	echo "WARN: runtime contains hardcoded absolute paths"
else
	echo "PASS: no suspicious absolute paths"
fi

if (( experimental != 0 )); then
	echo "EXPERIMENTAL: runtime may work only inside the original Termux/glibc layout"
fi

exit "$status"
