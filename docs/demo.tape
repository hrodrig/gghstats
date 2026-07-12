# gghstats terminal demo — run from repository root:
#   make install && PATH="$(go env GOPATH)/bin:$PATH" bash -c "vhs docs/demo.tape"
#
# Prepend GOPATH/bin when invoking vhs so a stale Homebrew (or PATH) binary
# does not shadow the VERSION-built binary from `make install`.
# Use bash -c (not bare vhs from zsh).
#
# Requires: vhs (https://github.com/charmbracelet/vhs), make, Go toolchain.

Output docs/demo.gif

Require make

Set Shell "bash"
Set FontSize 14
Set Width 960
Set Height 640
Set Padding 20

Enter
Sleep 300ms

# Install VERSION-built binary to GOPATH/bin
Type "make install"
Enter
Sleep 2s

# Root help — serve, fetch, report, export, backup, restore
Type "gghstats -h"
Enter
Sleep 2s

# Version / build metadata
Type "gghstats version"
Enter
Sleep 1s

# Dashboard serve flags (incl. --demo)
Type "gghstats serve -h"
Enter
Sleep 1500ms

# SQLite snapshot subcommand
Type "gghstats backup -h"
Enter
Sleep 1500ms

# Sample env (truncated for a short GIF)
Type "gghstats --print-sample-config 2>&1 | head -n 22"
Enter
Sleep 2s

Type "exit"
Enter
Sleep 500ms
