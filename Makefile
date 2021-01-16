BIN=bin
BIN_NAME=gotomation

.PHONY: fmt build clean test all

all: build

fmt:
	go fmt ./...

build $(BIN)/$(BIN_NAME): $(BIN)
	env CGO_ENABLED=0 go build -o $(BIN)/$(BIN_NAME)

clean:
	go clean -i
	rm -rf $(BIN)

test:
	go test ./...

$(BIN):
	mkdir -p $(BIN)

