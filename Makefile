
# project info
PACKAGE		:= $(shell go list)
BINARY_NAME := $(shell basename $(PACKAGE))

fmt: ## fmt
	@go fmt ./...
	@goimports -l -w .
