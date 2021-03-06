ci:
	test -z "$$(gofmt -d -s .)"
	go vet -composites=false ./...
	go test ./...

fix:
	gofmt -w -s .
	go generate ./...

.PHONY: ci fix
