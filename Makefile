# go-clean-arch-template — developer Makefile
# Run `make help` for the list of targets.

BINARY      := account-service
BIN_DIR     := bin
PKG         := ./...
GOBIN       := $(shell go env GOPATH)/bin

.DEFAULT_GOAL := help

## help: show this help
.PHONY: help
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed -e 's/## //'

## build: compile the binary to bin/
.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) .

## run-server: run the HTTP+gRPC server
.PHONY: run-server
run-server:
	go run main.go server

## run-worker: run the background worker
.PHONY: run-worker
run-worker:
	go run main.go worker

## migrate: apply database migrations
.PHONY: migrate
migrate:
	go run main.go migrate

## tidy: download + tidy modules
.PHONY: tidy
tidy:
	go mod tidy

## test: run all tests with race + coverage
.PHONY: test
test:
	go test -race -cover $(PKG)

## test-short: fast tests only
.PHONY: test-short
test-short:
	go test -short $(PKG)

## cover: write an HTML coverage report to coverage.html
.PHONY: cover
cover:
	go test -coverprofile=coverage.out $(PKG)
	go tool cover -html=coverage.out -o coverage.html

## vet: go vet
.PHONY: vet
vet:
	go vet $(PKG)

## fmt: gofmt the tree
.PHONY: fmt
fmt:
	gofmt -w -s .

## lint: golangci-lint (install: https://golangci-lint.run)
.PHONY: lint
lint:
	golangci-lint run

## check: fmt + vet + lint
.PHONY: check
check: fmt vet lint

## proto: regenerate protobuf/gRPC stubs (needs buf + protoc-gen-go[-grpc])
.PHONY: proto
proto:
	buf dep update
	buf generate

## sqlc: regenerate type-safe pgx queries (needs sqlc)
.PHONY: sqlc
sqlc:
	sqlc generate

## mocks: regenerate port mocks (needs go.uber.org/mock)
.PHONY: mocks
mocks:
	go generate ./internal/ports/outbound/mock/...

## swagger: regenerate the OpenAPI docs (needs swaggo/swag)
.PHONY: swagger
swagger:
	$(GOBIN)/swag init -g internal/apps/server/server.go -o docs/swagger --parseDependency --parseInternal

## generate: proto + sqlc + mocks + swagger
.PHONY: generate
generate: proto sqlc mocks swagger

## tools: install code-generation tools
.PHONY: tools
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install go.uber.org/mock/mockgen@latest

## docker-up: start postgres + mongo (replica set) + redis
.PHONY: docker-up
docker-up:
	docker compose up -d

## docker-down: stop infra containers
.PHONY: docker-down
docker-down:
	docker compose down

## docker-build: build the service image
.PHONY: docker-build
docker-build:
	docker build -t $(BINARY):latest .

## ci: tidy -> check -> test -> build
.PHONY: ci
ci: tidy check test build
