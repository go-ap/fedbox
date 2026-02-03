SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

ENV ?= dev
STORAGE ?= all

TAGS ?= ssh

ifneq ($(STORAGE),)
TAGS := storage_$(STORAGE),$(TAGS)
endif

TEST_FLAGS ?= -timeout 60s -count=1 -tags '$(TAGS)'

export DOCKER_HOST = unix:///run/user/$(shell id -u)/podman/podman.sock

download:
	go mod download
	go mod tidy

test:
	go test $(TEST_FLAGS) -json ./ -build | go tool tparse -all -progress -trimpath github.com/go-ap/
