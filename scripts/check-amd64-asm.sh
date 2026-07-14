#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_dir="${1:-$(mktemp -d)}"
mkdir -p "${output_dir}"

binary="${output_dir}/field-amd64.test"
disassembly="${output_dir}/field-amd64.objdump.txt"
symbols="${output_dir}/field-amd64.symbols.txt"
source_asm="${repo_dir}/internal/field/montgomery_amd64.s"
scalar_binary="${output_dir}/scalar-amd64.test"
scalar_disassembly="${output_dir}/scalar-amd64.objdump.txt"
scalar_symbols="${output_dir}/scalar-amd64.symbols.txt"
scalar_source_asm="${repo_dir}/internal/scalar/montgomery_amd64.s"
selector_binary="${output_dir}/root-amd64.test"
selector_disassembly="${output_dir}/w6-amd64.objdump.txt"
selector_symbols="${output_dir}/w6-amd64.symbols.txt"
selector_asm="${repo_dir}/scalar_select_amd64.s"

cd "${repo_dir}"
GOOS=linux GOARCH=amd64 GOAMD64="${GOAMD64:-v1}" \
  go test -c -tags=secp256k1_asm -o "${binary}" ./internal/field

go tool objdump -s 'github.com/islishude/secp256k1/internal/field\..*ADXAsm' "${binary}" >"${disassembly}"
go tool nm -size -sort address "${binary}" >"${symbols}"

for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm; do
  grep -q "${symbol}" "${symbols}"
done

GOOS=linux GOARCH=amd64 GOAMD64="${GOAMD64:-v1}" \
  go test -c -tags=secp256k1_asm -o "${scalar_binary}" ./internal/scalar
go tool objdump -s 'github.com/islishude/secp256k1/internal/scalar\..*ADXAsm' \
  "${scalar_binary}" >"${scalar_disassembly}"
go tool nm -size -sort address "${scalar_binary}" >"${scalar_symbols}"

for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm squareMontgomeryNADXAsm invVartimeWordsADXAsm; do
  grep -q "${symbol}" "${scalar_symbols}"
done
grep -Eq '^[[:space:]]*MULXQ[[:space:]]' "${scalar_source_asm}"
grep -Eq '^[[:space:]]*ADCXQ[[:space:]]' "${scalar_source_asm}"
grep -Eq '^[[:space:]]*ADOXQ[[:space:]]' "${scalar_source_asm}"

for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm; do
  symbol_source="${output_dir}/scalar-${symbol}.source.txt"
  awk -v symbol="${symbol}" '
    $0 ~ "^TEXT ·" symbol "\\(" { active = 1 }
    active && $0 ~ "^TEXT ·" && $0 !~ ("^TEXT ·" symbol "\\(") { exit }
    active { print }
  ' "${scalar_source_asm}" >"${symbol_source}"
  if grep -Eq '^[[:space:]]*J[A-Z]+[[:space:]]' "${symbol_source}"; then
    echo "secret-independent scalar kernel ${symbol} contains a branch" >&2
    exit 1
  fi
  head -n 1 "${symbol_source}" | grep -Fq "TEXT ·${symbol}(SB), NOSPLIT, \$0-"
done

scalar_square_n_source="${output_dir}/scalar-squareMontgomeryNADXAsm.source.txt"
awk '
  /^TEXT ·squareMontgomeryNADXAsm\(/ { active = 1 }
  active && /^TEXT ·/ && !/^TEXT ·squareMontgomeryNADXAsm\(/ { exit }
  active { print }
' "${scalar_source_asm}" >"${scalar_square_n_source}"
grep -Eq '^TEXT ·squareMontgomeryNADXAsm\(SB\), NOSPLIT, \$0-' "${scalar_square_n_source}"
test "$(grep -Ec '^[[:space:]]*JEQ[[:space:]]' "${scalar_square_n_source}")" -eq 2
test "$(grep -Ec '^[[:space:]]*JMP[[:space:]]' "${scalar_square_n_source}")" -eq 1
if grep -Eq '^[[:space:]]*J[A-Z]+[[:space:]]' "${scalar_square_n_source}" | grep -Ev '^[[:space:]]*(JEQ|JMP)[[:space:]]'; then
  echo 'scalar SquareN contains a non-counter branch' >&2
  exit 1
fi

scalar_inv_source="${output_dir}/scalar-invVartimeWordsADXAsm.source.txt"
awk '
  /^TEXT ·invVartimeWordsADXAsm\(/ { active = 1 }
  active && /^TEXT ·/ && !/^TEXT ·invVartimeWordsADXAsm\(/ { exit }
  active { print }
' "${scalar_source_asm}" >"${scalar_inv_source}"
head -n 1 "${scalar_inv_source}" | grep -Fq 'TEXT ·invVartimeWordsADXAsm(SB), NOSPLIT, $0-'
grep -Eq '^[[:space:]]*BSFQ[[:space:]]' "${scalar_inv_source}"
grep -Eq '^[[:space:]]*SHRXQ[[:space:]]' "${scalar_inv_source}"
grep -Eq '^[[:space:]]*SHLXQ[[:space:]]' "${scalar_inv_source}"
if grep -Eq '^[[:space:]]*JMP[[:space:]]+\([^)]*\)' "${scalar_inv_source}"; then
  echo 'public InvVartime contains an indirect branch' >&2
  exit 1
fi
for rejected in squareMontgomeryNADXAsm mulByB3MontgomeryADXAsm; do
  if grep -q "${rejected}" "${symbols}"; then
    echo "rejected AMD64 kernel ${rejected} is still linked" >&2
    exit 1
  fi
done

grep -Eq '^[[:space:]]*MULXQ[[:space:]]' "${source_asm}"
grep -Eq '^[[:space:]]*ADCXQ[[:space:]]' "${source_asm}"
grep -Eq '^[[:space:]]*ADOXQ[[:space:]]' "${source_asm}"

for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm; do
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

mul_source="${output_dir}/mulMontgomeryADXAsm.source.txt"
test "$(grep -Ec '^[[:space:]]*MOVOU[[:space:]]' "${mul_source}")" -eq 2
test "$(grep -Ec '^[[:space:]]*PSRLDQ[[:space:]]' "${mul_source}")" -eq 2

GOOS=linux GOARCH=amd64 GOAMD64="${GOAMD64:-v1}" \
  go test -c -tags=secp256k1_asm -o "${selector_binary}" .
go tool objdump -s 'github.com/islishude/secp256k1\.selectGeneratorW6' \
  "${selector_binary}" >"${selector_disassembly}"
go tool nm -size -sort address "${selector_binary}" >"${selector_symbols}"
grep -q 'selectGeneratorW6' "${selector_symbols}"

selector_source="${output_dir}/selectGeneratorW6.source.txt"
awk '
  /^TEXT ·selectGeneratorW6\(/ { active = 1 }
  active && /^TEXT ·/ && !/^TEXT ·selectGeneratorW6\(/ { exit }
  active { print }
' "${selector_asm}" >"${selector_source}"
test "$(grep -Ec '^[[:space:]]*MOVOU[[:space:]]' "${selector_source}")" -eq 132
test "$(grep -Ec '^[[:space:]]*CMPQ[[:space:]]' "${selector_source}")" -eq 31
test "$(grep -Ec '^[[:space:]]*SETEQ[[:space:]]' "${selector_source}")" -eq 31
if grep -Eq '^[[:space:]]*J[A-Z]+[[:space:]]' "${selector_source}"; then
  echo "W6 selector contains a branch" >&2
  exit 1
fi

# Go's portable objdump does not decode ADX opcodes on every host toolchain.
# On the native Linux CI runner, also retain and validate GNU objdump output.
if command -v objdump >/dev/null && objdump --version | grep -q 'GNU objdump'; then
  native_disassembly="${output_dir}/field-amd64.gnu-objdump.txt"
  : >"${native_disassembly}"
  for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm; do
    symbol_dump="${output_dir}/${symbol}.gnu-objdump.txt"
    objdump -d --disassemble="github.com/islishude/secp256k1/internal/field.${symbol}.abi0" \
      "${binary}" >"${symbol_dump}"
    cat "${symbol_dump}" >>"${native_disassembly}"
  done

  scalar_native_disassembly="${output_dir}/scalar-amd64.gnu-objdump.txt"
  : >"${scalar_native_disassembly}"
  for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm squareMontgomeryNADXAsm invVartimeWordsADXAsm; do
    symbol_dump="${output_dir}/scalar-${symbol}.gnu-objdump.txt"
    objdump -d --disassemble="github.com/islishude/secp256k1/internal/scalar.${symbol}.abi0" \
      "${scalar_binary}" >"${symbol_dump}"
    cat "${symbol_dump}" >>"${scalar_native_disassembly}"
  done
  grep -Eiq '[[:space:]]mulxq?[[:space:]]' "${scalar_native_disassembly}"
  grep -Eiq '[[:space:]]adcxq?[[:space:]]' "${scalar_native_disassembly}"
  grep -Eiq '[[:space:]]adoxq?[[:space:]]' "${scalar_native_disassembly}"
  for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm; do
    if grep -Eiq '[[:space:]]j[a-z]+[[:space:]]' "${output_dir}/scalar-${symbol}.gnu-objdump.txt"; then
      echo "native scalar disassembly for ${symbol} contains a branch" >&2
      exit 1
    fi
  done

  grep -Eiq '[[:space:]]mulxq?[[:space:]]' "${native_disassembly}"
  grep -Eiq '[[:space:]]adcxq?[[:space:]]' "${native_disassembly}"
  grep -Eiq '[[:space:]]adoxq?[[:space:]]' "${native_disassembly}"
  for symbol in mulMontgomeryADXAsm squareMontgomeryADXAsm; do
    if grep -Eiq '[[:space:]]j[a-z]+[[:space:]]' "${output_dir}/${symbol}.gnu-objdump.txt"; then
      echo "native disassembly for ${symbol} contains a branch" >&2
      exit 1
    fi
  done

  selector_dump="${output_dir}/selectGeneratorW6.gnu-objdump.txt"
  objdump -d --disassemble='github.com/islishude/secp256k1.selectGeneratorW6.abi0' \
    "${selector_binary}" >"${selector_dump}"
  cat "${selector_dump}" >>"${native_disassembly}"
  grep -Eiq '[[:space:]]movdqu[[:space:]]' "${selector_dump}"
  if grep -Eiq '[[:space:]]j[a-z]+[[:space:]]' "${selector_dump}"; then
    echo "native W6 selector disassembly contains a branch" >&2
    exit 1
  fi
fi
