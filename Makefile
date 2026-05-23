.PHONY: test benchmark format
test:
	go vet ./...
	go test -v -count=1 -cover -race ./...

benchmark:
	cd benchmark && go test -bench=. -benchmem -count=5

format:
	gofmt -w .
	go fix -diff ./...
