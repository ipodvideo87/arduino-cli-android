# ACL Scanner

The scanner module inspects ELF binaries and extracts the runtime metadata ACL needs:
the ELF class, machine type, SONAME, program interpreter, RPATH/RUNPATH, imported libraries,
and suspicious absolute paths embedded in the file.

The shell wrappers in this directory call the Go scanner in `cmd/acl-scan` so the same
logic is shared by all higher-level checks.
