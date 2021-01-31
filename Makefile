BIN_DIR ?= bin
BIN_NAME=gotomation
GOOS ?= linux
GOARCH ?= amd64
VERSION ?= $(CIRCLE_TAG)
PKG_NAME = github.com/nmaupu/gotomation
LDFLAGS = -ldflags="-X '$(PKG_NAME)/app.ApplicationVersion=$(VERSION)' -X '$(PKG_NAME)/app.BuildDate=$(shell date)'"

.PHONY: fmt build clean test all

all: build

fmt:
	go fmt ./...

build $(BIN_DIR)/$(BIN_NAME): $(BIN_DIR)
	# TODO: HTTP /version, /health and maybe other checking stuff
	env GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BIN_DIR)/$(BIN_NAME) $(LDFLAGS)

clean:
	go clean -i
	rm -rf $(BIN)

test:
	go test ./...

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

CI-process-release:
	@echo "Version to be released: $(CIRCLE_TAG)"
	ghr -t $(GITHUB_TOKEN) \
		-u $(CIRCLE_PROJECT_USERNAME) \
		-r $(CIRCLE_PROJECT_REPONAME) \
		-c $(CIRCLE_SHA1) \
		-n "Release v$(CIRCLE_TAG)" \
		-b "$(shell git log --format=%B -n1 $(CIRCLE_SHA1))" \
		-delete \
		$(CIRCLE_TAG) $(BIN_DIR)/
