#!/usr/bin/env bash
set -euo pipefail

if (( $# != 2 )); then
  echo "usage: $(basename "$0") <environment|benchmarks|profiles|report|gate> <output-directory>" >&2
  exit 2
fi

action="$1"
output="$2"
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mkdir -p "${output}"
output="$(cd "${output}" && pwd)"
cd "${repo_dir}"

record_environment() {
  lscpu >"${output}/lscpu.txt"
  taskset -pc $$ >"${output}/affinity.txt"
  go env >"${output}/go-env.txt"
  go test -v -count=1 -tags=secp256k1_asm -run TestAMD64FeatureDispatch ./internal/field \
    | tee "${output}/cpuid.txt"
  grep -q 'CPUID ADX+BMI2=true' "${output}/cpuid.txt"
  for kernel in none mul square; do
    SECP256K1_AMD64_BENCH_KERNEL="${kernel}" \
      go test -count=1 -tags='secp256k1_asm,secp256k1_amd64_bench' \
        -run '^TestAMD64BenchmarkKernelSelection$' ./internal/field
  done
}

run_benchmarks() {
  for package in root field scalar; do
    : >"${output}/${package}-default.txt"
    : >"${output}/${package}-tagged.txt"
  done
  for kernel in none mul square; do
    : >"${output}/root-${kernel}.txt"
  done
  : >"${output}/root-w5.txt"

  run_mode() {
    local mode="$1"
    local -a tags=()
    if [[ "${mode}" == tagged ]]; then
      tags=(-tags=secp256k1_asm)
    fi
    GOMAXPROCS=1 taskset -c 0 go test "${tags[@]}" -run '^$' \
      -bench '^Benchmark(SignDigest|SignRecoverableDigest|VerifyDigest|RecoverDigest|VerifyHotPublicKey|VerifyParseCompressedCold|VerifyParseUncompressedCold|SignCompact|SignRecoverable|PublicKeyDerive|ScalarInv|ScalarInvVartime|ScalarBaseMultProjective)$' \
      -benchmem -benchtime=500ms -count=1 . >>"${output}/root-${mode}.txt"
    GOMAXPROCS=1 taskset -c 0 go test "${tags[@]}" -run '^$' \
      -bench '^BenchmarkField(Add|Sub|Mul|MulByB3|Square|SquareN)$' \
      -benchmem -benchtime=500ms -count=1 ./internal/field >>"${output}/field-${mode}.txt"
    GOMAXPROCS=1 taskset -c 0 go test "${tags[@]}" -run '^$' \
      -bench '^BenchmarkScalar(Mul|Square|SquareN|Inv)$' \
      -benchmem -benchtime=500ms -count=1 ./internal/scalar >>"${output}/scalar-${mode}.txt"
  }

  run_kernel() {
    local kernel="$1"
    SECP256K1_AMD64_BENCH_KERNEL="${kernel}" \
      GOMAXPROCS=1 taskset -c 0 go test -tags='secp256k1_asm,secp256k1_amd64_bench,secp256k1_amd64_w5_bench' -run '^$' \
        -bench '^Benchmark(SignRecoverable|VerifyHotPublicKey)$' \
        -benchmem -benchtime=500ms -count=1 . >>"${output}/root-${kernel}.txt"
  }

  run_w5() {
    GOMAXPROCS=1 taskset -c 0 go test -tags='secp256k1_asm,secp256k1_amd64_w5_bench' -run '^$' \
      -bench '^Benchmark(ScalarBaseMultProjective|SignRecoverable|VerifyHotPublicKey)$' \
      -benchmem -benchtime=500ms -count=1 . >>"${output}/root-w5.txt"
  }

  for iteration in $(seq 1 10); do
    if (( iteration % 2 == 1 )); then
      run_mode default
      run_w5
      run_mode tagged
    else
      run_mode tagged
      run_w5
      run_mode default
    fi
    kernels=(none mul square)
    for offset in 0 1 2; do
      index=$(( (iteration + offset - 1) % 3 ))
      run_kernel "${kernels[index]}"
    done
  done
}

capture_profiles() {
  capture_profile() {
    local benchmark="$1"
    GOMAXPROCS=1 taskset -c 0 go test -tags=secp256k1_asm -run '^$' \
      -bench "^Benchmark${benchmark}$" -benchtime=3s -count=1 \
      -cpuprofile="${output}/tagged-${benchmark}.cpu.pprof" . \
      >"${output}/tagged-${benchmark}.profile.txt"
  }
  capture_profile SignRecoverable
  capture_profile VerifyHotPublicKey
}

build_report() {
  go run ./cmd/benchcmp "${output}/root-default.txt" "${output}/root-tagged.txt" \
    2>&1 | tee "${output}/root-summary.txt"
  go run ./cmd/benchcmp "${output}/field-default.txt" "${output}/field-tagged.txt" \
    2>&1 | tee "${output}/field-summary.txt"
  go run ./cmd/benchcmp "${output}/scalar-default.txt" "${output}/scalar-tagged.txt" \
    2>&1 | tee "${output}/scalar-summary.txt"
  go run ./cmd/benchcmp "${output}/root-w5.txt" "${output}/root-tagged.txt" \
    2>&1 | tee "${output}/root-w6-stage-summary.txt"
  for kernel in mul square; do
    go run ./cmd/benchcmp "${output}/root-none.txt" "${output}/root-${kernel}.txt" \
      2>&1 | tee "${output}/root-${kernel}-summary.txt"
  done
}

check_gate() {
  go run ./cmd/benchcmp -gate=final-paired \
    "${output}/root-default.txt" "${output}/root-tagged.txt"
}

case "${action}" in
  environment)
    record_environment
    ;;
  benchmarks)
    run_benchmarks
    ;;
  profiles)
    capture_profiles
    ;;
  report)
    build_report
    ;;
  gate)
    check_gate
    ;;
  *)
    echo "unknown action: ${action}" >&2
    exit 2
    ;;
esac
