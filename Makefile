.PHONY: test test-arm64-asm test-amd64-asm lint benchmark perf-check perf-check-arm64 perf-check-amd64 check-main-deps vartime-audit generate-asm check-asm-module generate-check amd64-asm-audit fuzz-smoke ct-smoke format
test:
	go test -v -count=1 -cover -race ./...
	cd benchmark && go test .

test-arm64-asm:
	go test -tags=secp256k1_asm ./...
	go test -race -tags=secp256k1_asm ./...

test-amd64-asm:
	@test "$$(go env GOARCH)" = amd64 || { printf '%s\n' 'test-amd64-asm requires an amd64 host' >&2; exit 1; }
	go test -tags=secp256k1_asm ./...
	go test -race -tags=secp256k1_asm ./...

lint: format
	golangci-lint run
	go vet ./...

benchmark:
	cd benchmark && go test -bench=. -benchmem -count=5

perf-check:
	go test -run '^$$' -bench '^Benchmark(VerifyHotPublicKey|SignRecoverable|RecoverDigest)$$' -benchmem -count=5 .

perf-check-arm64:
	go test -tags=secp256k1_asm -run '^$$' -bench '^Benchmark(VerifyHotPublicKey|SignRecoverable|RecoverDigest)$$' -benchmem -count=5 .

perf-check-amd64:
	@test "$$(go env GOARCH)" = amd64 || { printf '%s\n' 'perf-check-amd64 requires an amd64 host' >&2; exit 1; }
	go test -tags=secp256k1_asm -run '^$$' -bench '^Benchmark(VerifyHotPublicKey|SignRecoverable|RecoverDigest)$$' -benchmem -count=5 .

check-main-deps:
	@mods="$$(go list -m all)"; count="$$(printf '%s\n' "$$mods" | sed '/^$$/d' | wc -l | tr -d ' ')"; \
	if [ "$$count" -ne 1 ]; then printf '%s\n' "main module has external dependencies:" "$$mods" >&2; exit 1; fi

vartime-audit:
	./scripts/check-vartime.sh

generate-asm:
	cd asm && go generate ./...

check-asm-module:
	cd asm && go mod tidy -diff
	cd asm && go test ./...
	cd asm && go vet ./...

generate-check:
	go generate ./...
	$(MAKE) generate-asm
	git diff --exit-code

amd64-asm-audit:
	./scripts/check-amd64-asm.sh

fuzz-smoke:
	go test -run=^$$ -fuzz=FuzzParseDERSignature -fuzztime=10s .

ct-smoke:
	SECP256K1_CT_TEST=1 go test -run TestConstantTimeScalarBaseMultSmoke -count=1 .

format:
	gofmt -w .
	go fix ./...
