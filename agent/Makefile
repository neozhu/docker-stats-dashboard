BINARY ?= bin/docker-agent
GO_FLAGS ?=

.PHONY: build run test tidy clean

build:
	go build $(GO_FLAGS) -o $(BINARY) .

run:
	go run $(GO_FLAGS) .

test:
	go test $(GO_FLAGS) ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BINARY)
