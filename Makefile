VERSION := 0.0.5
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git branch --show-current)

.PHONY: build
build:
	@echo "==> Building ..."
	@go build -ldflags="-X github.com/prometheus/common/version.Version=$(VERSION) -X github.com/prometheus/common/version.Revision=$(GIT_COMMIT) -X github.com/prometheus/common/version.Branch=$(GIT_BRANCH)" .

.PHONY: upgrade-deps
upgrade-deps:
	@echo "==> Upgrading dependencies..."
	@go get -t -u ./...
	@go mod tidy

.PHONY: dependencies
dependencies:
	@echo "==> Downloading dependencies..."
	@go mod download -x
