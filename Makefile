BINARY_NAME=starker

build:
	go build -o tmp/$(BINARY_NAME) ./cmd

run: build
	./tmp/$(BINARY_NAME)

web:
	cd website && pnpm run dev

dev:
	air --build.cmd "go build -o tmp/$(BINARY_NAME) ./cmd" --build.bin "./tmp/$(BINARY_NAME)"

api:
	air --build.cmd "go build -o tmp/$(BINARY_NAME) ./cmd" --build.bin "./tmp/$(BINARY_NAME) api"

worker:
	air --build.cmd "go build -o tmp/$(BINARY_NAME) ./cmd" --build.bin "./tmp/$(BINARY_NAME) worker"

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
