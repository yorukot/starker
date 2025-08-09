BINARY_NAME=starker

build:
	go build -o tmp/$(BINARY_NAME) cmd/main.go

run: build
	./tmp/$(BINARY_NAME)

dev:
	air --build.cmd "go build -o tmp/$(BINARY_NAME) cmd/main.go" --build.bin "./tmp/$(BINARY_NAME)"

test:
	go test ./...

lint:
	go fmt ./...
	go vet ./...
	golint ./...

generate-docs:
	swag init -g cmd/main.go -o ./docs

clean:
	rm -rf tmp/

.PHONY: build run test clean
