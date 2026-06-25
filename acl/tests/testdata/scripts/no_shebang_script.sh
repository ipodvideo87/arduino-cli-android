# No shebang line — this script relies on the parent shell to execute it.
# Test fixture: script without a "#!" line.
# Expected scanner result: interpreter_status absent (nil).
echo "no shebang here"
