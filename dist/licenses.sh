#!/usr/bin/env bash
# Regenerate dist/THIRD_PARTY_LICENSES.txt from the licence files of every
# module linked into the binary (plus Go's runtime), verbatim.
set -euo pipefail
cd "$(dirname "$0")/.."
go build -o /tmp/sprawl-licences-bin ./cmd/sprawl
out=dist/THIRD_PARTY_LICENSES.txt
{
  echo "Sprawl includes the following third-party software."
  echo
  gr=$(go env GOROOT)
  echo "================================================================"
  echo "Go standard library and runtime ($(go env GOVERSION))"
  echo "================================================================"
  cat "$gr/LICENSE"
  echo
  go version -m /tmp/sprawl-licences-bin | awk '$1=="dep"{print $2, $3}' | while read -r mod ver; do
    dir=$(go list -m -f '{{.Dir}}' "$mod")
    echo "================================================================"
    echo "$mod $ver"
    echo "================================================================"
    for f in LICENSE LICENSE.md LICENSE.txt COPYING NOTICE NOTICE.md; do
      if [[ -f "$dir/$f" ]]; then cat "$dir/$f"; echo; fi
    done
  done
} > "$out"
rm -f /tmp/sprawl-licences-bin
echo "wrote $out"
