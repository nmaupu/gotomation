BIN_DIR ?= bin
BIN_NAME=gotomation
GOOS ?= darwin
GOARCH ?= amd64
CIRCLE_TAG ?= main
VERSION = $(CIRCLE_TAG)
PKG_NAME = github.com/nmaupu/gotomation
LDFLAGS = -ldflags="-X '$(PKG_NAME)/app.ApplicationVersion=$(VERSION)' -X '$(PKG_NAME)/app.BuildDate=$(shell date)'"
GHR ?= ghr

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: build
build $(BIN_DIR)/$(BIN_NAME): $(BIN_DIR)
	env GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -o $(BIN_DIR)/$(BIN_NAME)-$(GOOS)_$(GOARCH)-$(VERSION) $(LDFLAGS)

.PHONY: clean
clean:
	go clean -i
	rm -rf $(BIN)

.PHONY: test
test:
	go test ./...

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

.PHONY: CI-process-release
CI-process-release:
	@echo "Version to be released: $(CIRCLE_TAG)"
	$(GHR) -t $(GITHUB_TOKEN) \
		   -u $(CIRCLE_PROJECT_USERNAME) \
		   -r $(CIRCLE_PROJECT_REPONAME) \
		   -c $(CIRCLE_SHA1) \
		   -n "Release v$(CIRCLE_TAG)" \
		   -b "$(shell git log --format=%B -n1 $(CIRCLE_SHA1))" \
		   -delete \
		   $(CIRCLE_TAG) $(BIN_DIR)/
