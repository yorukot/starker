BINARY_NAME=starker
SWAGGER_PORT=8081

build:
	go build -o tmp/$(BINARY_NAME) cmd/main.go

run: build
	./tmp/$(BINARY_NAME)

web:
	cd website && pnpm run dev

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

swagger: generate-docs
	docker run --rm --name swagger-ui -p $(SWAGGER_PORT):8080 -e SWAGGER_JSON=/docs/swagger.json -v $(PWD)/docs:/docs swaggerapi/swagger-ui

clean:
	rm -rf tmp/

.PHONY: build run test clean generate-docs swagger-ui swagger-ui-stop
