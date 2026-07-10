.PHONY: test lint benchmark perf-check check-main-deps vartime-audit generate-check fuzz-smoke ct-smoke format
test:
	go test -v -count=1 -cover -race ./...
	cd benchmark && go test .

lint: format
	golangci-lint run
	go vet ./...

benchmark:
	cd benchmark && go test -bench=. -benchmem -count=5

perf-check:
	go test -run '^$$' -bench '^Benchmark(VerifyHotPublicKey|SignRecoverable|RecoverDigest)$$' -benchmem -count=5 .

check-main-deps:
	@mods="$$(go list -m all)"; count="$$(printf '%s\n' "$$mods" | sed '/^$$/d' | wc -l | tr -d ' ')"; \
	if [ "$$count" -ne 1 ]; then printf '%s\n' "main module has external dependencies:" "$$mods" >&2; exit 1; fi

vartime-audit:
	@if grep -nH 'Vartime' sign.go privatekey.go rfc6979.go; then \
		printf '%s\n' 'Vartime function used in a secret path' >&2; exit 1; \
	fi

generate-check:
	go generate ./...
	git diff --exit-code

fuzz-smoke:
	go test -run=^$$ -fuzz=FuzzParseDERSignature -fuzztime=10s .

ct-smoke:
	SECP256K1_CT_TEST=1 go test -run TestConstantTimeScalarBaseMultSmoke -count=1 .

format:
	gofmt -w .
	go fix ./...
