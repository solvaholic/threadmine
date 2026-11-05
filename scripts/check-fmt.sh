#!/bin/sh
set -eu

formatter=${1:?usage: check-fmt.sh /path/to/gofmt}
if [ ! -x "$formatter" ]; then
    echo "Go formatter not found: $formatter. Check the selected Go installation." >&2
    exit 1
fi

if ! files=$(find . -type d \( -name .git -o -name vendor \) -prune -o \
    -type f -name '*.go' -exec "$formatter" -l {} +); then
    echo "Formatting check failed. Fix the Go source or formatter error above." >&2
    exit 1
fi
if [ -n "$files" ]; then
    printf '%s\n' "$files"
    echo "Go files need formatting. Run make fmt." >&2
    exit 1
fi
