BINARY_NAME=stargo

build:
	go build -o tmp/$(BINARY_NAME) cmd/main.go

run: build
	./tmp/$(BINARY_NAME)

test:
	go test ./...

lint:
	go fmt ./...
	go vet ./...
	golint ./...

clean:
	rm -rf tmp/

.PHONY: build run test clean
