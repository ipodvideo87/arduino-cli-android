#!/usr/bin/env notarealinterpreter
# Test fixture: env-style shebang pointing to an interpreter that is not
# known to ACL and is not installed under Termux PREFIX.
# Expected scanner result: interpreter_status = missing.
echo "this should never run on Termux without the interpreter"
