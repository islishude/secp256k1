.PHONY: test lint benchmark fuzz-smoke ct-smoke format
test:
	go test -v -count=1 -cover -race ./...
	cd benchmark && go test .

lint: format
	golangci-lint run
	go vet ./...

benchmark:
	cd benchmark && go test -bench=. -benchmem -count=5

fuzz-smoke:
	go test -run=^$$ -fuzz=FuzzParseDERSignature -fuzztime=10s .

ct-smoke:
	SECP256K1_CT_TEST=1 go test -run TestConstantTimeScalarBaseMultSmoke -count=1 .

format:
	gofmt -w .
	go fix ./...
