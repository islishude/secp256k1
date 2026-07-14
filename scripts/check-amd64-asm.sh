#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="${1:-$(mktemp -d)}"
mkdir -p "${output_dir}"

binary="${output_dir}/field-amd64.test"
disassembly="${output_dir}/field-amd64.objdump.txt"
symbols="${output_dir}/field-amd64.symbols.txt"
source_asm="${repo_dir}/internal/field/montgomery_amd64.s"

cd "${repo_dir}"
GOOS=linux GOARCH=amd64 GOAMD64="${GOAMD64:-v1}" \
  go test -c -tags=secp256k1_asm -o "${binary}" ./internal/field

go tool objdump -s 'github.com/islishude/secp256k1/internal/field\..*ADXAsm' "${binary}" >"${disassembly}"
go tool nm -size -sort address "${binary}" >"${symbols}"

for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm squareMontgomeryNADXAsm mulByB3MontgomeryADXAsm; do
  grep -q "${symbol}" "${symbols}"
done

grep -Eq '^[[:space:]]*MULXQ[[:space:]]' "${source_asm}"
grep -Eq '^[[:space:]]*ADCXQ[[:space:]]' "${source_asm}"
grep -Eq '^[[:space:]]*ADOXQ[[:space:]]' "${source_asm}"

for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm mulByB3MontgomeryADXAsm; do
  symbol_source="${output_dir}/${symbol}.source.txt"
  awk -v symbol="${symbol}" '
    $0 ~ "^TEXT ·" symbol "\\(" { active = 1 }
    active && $0 ~ "^TEXT ·" && $0 !~ ("^TEXT ·" symbol "\\(") { exit }
    active { print }
  ' "${source_asm}" >"${symbol_source}"
  if grep -Eq '^[[:space:]]*J[A-Z]+[[:space:]]' "${symbol_source}"; then
    echo "secret-independent kernel ${symbol} contains a branch" >&2
    exit 1
  fi
done

square_n_source="${output_dir}/squareMontgomeryNADXAsm.source.txt"
awk '
  /^TEXT ·squareMontgomeryNADXAsm\(/ { active = 1 }
  active && /^TEXT ·/ && !/^TEXT ·squareMontgomeryNADXAsm\(/ { exit }
  active { print }
' "${source_asm}" >"${square_n_source}"
if grep -E '^[[:space:]]*J[A-Z]+[[:space:]]' "${square_n_source}" | grep -Ev '^[[:space:]]*J(E|NE)[[:space:]]'; then
  echo "SquareN contains a branch other than its public loop control" >&2
  exit 1
fi
