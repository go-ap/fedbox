SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

ENV ?= dev
STORAGE ?= all

PODMAN_SOCKET ?= /run/user/$(shell id -u)/podman/podman.sock

ifneq ($(STORAGE),)
STORAGE_TAG = storage_$(STORAGE),
endif

TEST_FLAGS ?= -timeout 180s -count=1 -tags '$(STORAGE_TAG)integration,$(TAGS)'

ifeq ($(origin GOCOVERDIR),undefined)
TEST_FLAGS += -test.artifacts
endif

ifeq ($(origin TAGS),undefined)
TAGS = c2s,s2s,ssh
endif

.PHONY: download test coverage

download:
	go mod download
	go mod tidy

test:
	DOCKER_HOST="unix://$(PODMAN_SOCKET)" \
	go test $(TEST_FLAGS) -json ./ -build -coverage -verbose | go tool tparse -all -progress -trimpath github.com/go-ap/

coverage:
	go tool covdata percent -i="$(find ./ -iname "coverage" -exec ls -d {} + | tr ' ' ',')"
