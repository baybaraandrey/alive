GO           ?= go
GOFMT        ?= $(GO)fmt
GOOPTS       ?=
GO111MODULE  :=
pkgs          = ./...


.PHONY: build
build:
	@echo ">> building alive"
	GO111MODULE=$(GO111MODULE) $(GO) build $(GOOPTS) ./cmd/alive.go
