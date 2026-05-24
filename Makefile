.PHONY: test benchmark format
test:
	go test -v -count=1 -cover -race ./...
	cd benchmark && go test .

lint: format
	golangci-lint run
	go vet ./...

benchmark:
	cd benchmark && go test -bench=. -benchmem -count=5

format:
	gofmt -w .
	go fix ./...
