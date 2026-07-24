SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

ENV ?= dev
STORAGE ?= all

TAGS ?=

ifneq ($(STORAGE),)
TAGS := $(TAGS),storage_$(STORAGE),c2s,s2s,ssh
endif

TEST_FLAGS ?= -timeout 40s -count=1 -tags '$(TAGS)'


download:
	go mod download
	go mod tidy

test:
	@export DOCKER_HOST=unix:///run/user/$(shell id -u)/podman/podman.sock
	go test $(TEST_FLAGS) -json ./ -build | go tool tparse -all -progress -trimpath github.com/go-ap/
