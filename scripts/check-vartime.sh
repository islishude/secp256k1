#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_dir}"

if rg -n 'Vartime|invVartime' sign.go privatekey.go rfc6979.go; then
  echo 'variable-time function used in a secret path' >&2
  exit 1
fi

callers="$({ rg -l --glob '*.go' --glob '!*_test.go' '\.InvVartime\(' . || true; } | sort)"
expected="$(printf '%s\n' ./recover.go ./verify.go | sort)"
if [[ "${callers}" != "${expected}" ]]; then
  echo 'unexpected production InvVartime caller set:' >&2
  printf '%s\n' "${callers}" >&2
  exit 1
fi

asm_callers="$({ rg -l --glob '*.go' --glob '!*_test.go' 'invVartimeWordsADXAsm\(' . || true; } | sort)"
expected_asm_callers="$(printf '%s\n' \
  ./internal/scalar/montgomery_amd64.go \
  ./internal/scalar/montgomery_amd64_stub.go | sort)"
if [[ "${asm_callers}" != "${expected_asm_callers}" ]]; then
  echo 'public-input inversion assembly escaped its scalar backend wrapper:' >&2
  printf '%s\n' "${asm_callers}" >&2
  exit 1
fi
