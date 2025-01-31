BIN_DIR ?= bin
BIN_NAME=gotomation
GOOS ?= linux
GOARCH ?= amd64
CIRCLE_BRANCH ?= main
ifeq ($(CIRCLE_TAG),)
	VERSION := $(CIRCLE_BRANCH)
else
	VERSION = $(CIRCLE_TAG)
endif
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
	env GO111MODULE=on \
		CGO_ENABLED=0 \
		GOOS=$(GOOS) \
		GOARCH=$(GOARCH) \
		GOARM=$(GOARM) \
		go build -o $(BIN_DIR)/$(BIN_NAME)-$(GOOS)_$(GOARCH)-$(VERSION) $(LDFLAGS)

.PHONY: clean
clean:
	go clean -i
	rm -rf $(BIN)

.PHONY: test
test:
	go test ./...

.PHONY: docker
docker:
	docker build \
		--build-arg GOTOMATION_VERSION=$(VERSION) \
		--build-arg GOTOMATION_BIN_DIR=$(BIN_DIR) \
		-t docker.io/nmaupu/gotomation:$(VERSION) .
	docker push docker.io/nmaupu/gotomation:$(VERSION)

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
