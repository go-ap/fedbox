SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

ENV ?= dev
STORAGE ?= all

TAGS ?=

PODMAN_SOCKET ?= /run/user/$(shell id -u)/podman/podman.sock

ifneq ($(STORAGE),)
TAGS := $(TAGS),storage_$(STORAGE)
endif

TEST_FLAGS ?= -timeout 90s -count=1 -tags '$(TAGS)'

.PHONY: download test

download:
	go mod download
	go mod tidy

test:
	@export DOCKER_HOST="unix://$(PODMAN_SOCKET)"
	go test $(TEST_FLAGS) -json ./ -build | go tool tparse -all -progress -trimpath github.com/go-ap/
