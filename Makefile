SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
TEST_FLAGS ?= -count=1 -v

LOCAL_HOSTNAME ?= fedbox.git
STORAGE ?= all
VERSION ?= (unknown)
export CGO_ENABLED=0
export VERSION=$(VERSION)
GO := go
ENV ?= dev
LDFLAGS ?= -X main.version=$(VERSION)
BUILDFLAGS ?= -trimpath -a -ldflags '$(LDFLAGS)'
APPSOURCES := $(wildcard app/*.go storage/*/*.go activitypub/*.go internal/*/*.go cmd/*.go)
ASSETFILES := $(wildcard templates/*)
PROJECT_NAME := $(shell basename $(PWD))
APPSOURCES += internal/assets/assets.gen.go
TAGS := $(ENV) storage_$(STORAGE)

ifneq ($(ENV), dev)
	LDFLAGS += -s -w -extldflags "-static"
endif

ifeq ($(shell git describe --always > /dev/null 2>&1 ; echo $$?), 0)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD | tr '/' '-')
HASH=$(shell git rev-parse --short HEAD)
export VERSION = $(shell printf "%s-%s" "$(BRANCH)" "$(HASH)")
endif
ifeq ($(shell git describe --tags > /dev/null 2>&1 ; echo $$?), 0)
export VERSION = $(shell git describe --tags | tr '/' '-')
endif

BUILD := $(GO) build $(BUILDFLAGS)
TEST := $(GO) test $(BUILDFLAGS)

.PHONY: all run clean test coverage integration

all: fedbox ctl

assets: internal/assets/assets.gen.go

internal/assets/assets.gen.go: $(ASSETFILES)
	go generate -tags "$(TAGS)" ./assets.go

fedbox: bin/fedbox
bin/fedbox: go.mod cli/fedbox/main.go $(APPSOURCES)
	$(BUILD) -tags "$(TAGS)" -o $@ ./cli/fedbox/main.go

ctl: bin/ctl
bin/ctl: go.mod cli/control/main.go $(APPSOURCES)
	$(BUILD) -tags "$(TAGS)" -o $@ ./cli/control/main.go

run: fedbox
	@./bin/fedbox

clean:
	-$(RM) bin/*
	$(MAKE) -C tests $@


test: TEST_TARGET := ./{activitypub,app,storage,internal,cmd}/...
test:
	$(TEST) $(TEST_FLAGS) -tags "$(TAGS)" $(TEST_TARGET)

coverage: TEST_TARGET := .
coverage: TEST_FLAGS += -covermode=count -coverprofile $(PROJECT_NAME).coverprofile
coverage: test

integration:
	$(MAKE) -C tests $@

$(LOCAL_HOSTNAME).key $(LOCAL_HOSTNAME).crt:
	openssl req -subj "/C=AQ/ST=Omond/L=Omond/O=*.$(LOCAL_HOSTNAME)/OU=none/CN=*.$(LOCAL_HOSTNAME)" -newkey rsa:2048 -sha256 -keyout $(LOCAL_HOSTNAME).key -nodes -x509 -days 365 -out $(LOCAL_HOSTNAME).crt

$(LOCAL_HOSTNAME).pem: $(LOCAL_HOSTNAME).key $(LOCAL_HOSTNAME).crt
	cat $(LOCAL_HOSTNAME).key $(LOCAL_HOSTNAME).crt > $(LOCAL_HOSTNAME).pem

cert: $(LOCAL_HOSTNAME).key
