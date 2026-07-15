#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_dir}"

if grep -nHE 'Vartime|invVartime' sign.go privatekey.go rfc6979.go; then
  echo 'variable-time function used in a secret path' >&2
  exit 1
fi

list_production_go_files_with() {
  local pattern="$1"
  if command -v rg >/dev/null 2>&1; then
    rg -l --glob '*.go' --glob '!*_test.go' "${pattern}" . || true
    return
  fi
  find . -name '*.go' ! -name '*_test.go' -exec grep -lE "${pattern}" {} + || true
}

callers="$(list_production_go_files_with '\.InvVartime\(' | sort)"
expected="$(printf '%s\n' ./recover.go ./verify.go | sort)"
if [[ "${callers}" != "${expected}" ]]; then
  echo 'unexpected production InvVartime caller set:' >&2
  printf '%s\n' "${callers}" >&2
  exit 1
fi

asm_callers="$(list_production_go_files_with 'invVartimeWordsADXAsm\(' | sort)"
if [[ -n "${asm_callers}" ]]; then
  echo 'rejected public-input inversion assembly caller remains:' >&2
  printf '%s\n' "${asm_callers}" >&2
  exit 1
fi
