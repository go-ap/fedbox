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
STORAGE_TAG = storage_$(STORAGE),
endif

ifeq ($(TAGS),)
TAGS = c2s,s2s,ssh
endif

TEST_FLAGS ?= -timeout 180s -count=1 -tags '$(STORAGE_TAG)integration,$(TAGS)'

.PHONY: download test coverage

download:
	go mod download
	go mod tidy

test:
	DOCKER_HOST="unix://$(PODMAN_SOCKET)" \
	go test $(TEST_FLAGS) -json ./ -build -coverage -verbose | go tool tparse -all -progress -trimpath github.com/go-ap/

coverage: test
	go tool covdata percent -i="$(find ./ -iname "coverage" -exec ls -d {} + | tr ' ' ',')"
