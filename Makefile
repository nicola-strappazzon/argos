SHELL := /bin/bash
VERSION := `date +%Y%m%d%H%M`
GREEN := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET := $(shell tput -Txterm sgr0)

.PHONY: help tests deps build api web clean

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "${YELLOW}%-16s${GREEN}%s${RESET}\n", $$1, $$2}' $(MAKEFILE_LIST)

tests: ## Run the tests of the project.
	@go test ./... -v

deps: ## Download dependencies
	@go mod tidy
	@go mod download

build: ## Build binary for local operating system
	@go generate ./...
	@go build -o argos main.go

run: ## Run code.
	@go run main.go

clean: ## Remove build related file
	@go clean
