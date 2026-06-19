# ACL Scanner

The scanner module now has two related responsibilities:

- inspect individual ELF binaries and extract runtime metadata ACL needs
- scan Arduino package installations and produce structured tool compatibility reports

For ELF inspection, the scanner extracts:

- ELF class
- machine type
- SONAME
- program interpreter
- RPATH/RUNPATH
- imported libraries
- suspicious absolute paths embedded in the file

For tool compatibility reporting, `acl-scan compat` and `acl-scan compat-json` walk an
installed Arduino packages tree, classify executable candidates, and report:

- executable type such as ELF, shell script, Python, or Java archive
- architecture
- interpreter
- shared library dependencies
- RPATH/RUNPATH
- hardcoded absolute paths
- compatibility category such as native Android compatible, Linux/glibc executable, script, unknown, or unsupported

Compatibility is currently determined by a conservative set of rules:

- ELF magic, PT_INTERP, imported libraries, RPATH/RUNPATH, and embedded absolute paths
- shebang detection for shell, Python, and similar scripts
- archive detection for Java `.jar` tools
- executable-bit and file-shape checks to avoid treating ordinary data files as tools

Compatibility classification is currently heuristic and intentionally conservative. It
is meant to show what Arduino CLI has installed and which tools are likely runtime
candidates, not to prove that execution is already safe.

Machine-readable output is emitted by `acl-scan compat-json`. Human-readable output is
emitted by `acl-scan compat`.

Compatibility heuristics currently live in `internal/acl/toolcompat`. Future
compatibility rules should be added there in a structured way first, and then promoted
into versioned ACL metadata under `acl/database/` as the rule set stabilizes.

The shell wrappers in this directory call the Go scanner in `cmd/acl-scan` so the same
logic is shared by all higher-level checks.
