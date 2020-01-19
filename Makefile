SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
export VERSION=(unknown)
GO := go
ENV ?= dev
LDFLAGS ?= -X main.version=$(VERSION)
BUILDFLAGS ?= -a -ldflags '$(LDFLAGS)'
APPSOURCES := $(wildcard app/*.go storage/*/*.go activitypub/*.go internal/*/*.go cmd/*.go)
PROJECT_NAME := $(shell basename $(PWD))

ifneq ($(ENV), dev)
	LDFLAGS += -s -w -extldflags "-static"
endif

ifeq ($(shell git describe --always > /dev/null 2>&1 ; echo $$?), 0)
export VERSION = $(shell git describe --always --dirty="-git")
endif
ifeq ($(shell git describe --tags > /dev/null 2>&1 ; echo $$?), 0)
export VERSION = $(shell git describe --tags)
endif

BUILD := $(GO) build $(BUILDFLAGS)
TEST := $(GO) test $(BUILDFLAGS)

.PHONY: all run clean test coverage integration

all: fedbox ctl

fedbox: bin/fedbox
bin/fedbox: go.mod cli/app/main.go $(APPSOURCES)
	$(BUILD) -tags $(ENV) -o $@ ./cli/app/main.go

ctl: bin/ctl
bin/ctl: go.mod cli/control/main.go $(APPSOURCES)
	$(BUILD) -tags $(ENV) -o $@ ./cli/control/main.go

run: fedbox
	@./bin/fedbox

clean:
	-$(RM) bin/*
	$(MAKE) -C tests $@


test: TEST_TARGET := ./{activitypub,app,storage,internal}/...
test:
	$(TEST) $(TEST_FLAGS) $(TEST_TARGET)

coverage: TEST_TARGET := .
coverage: TEST_FLAGS += -covermode=count -coverprofile $(PROJECT_NAME).coverprofile
coverage: test

integration:
	$(MAKE) -C tests $@
