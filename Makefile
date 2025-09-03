SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

FEDBOX_HOSTNAME ?= fedbox.git
STORAGE ?= all
ENV ?= dev
PROJECT ?= fedbox
VERSION ?= HEAD

LDFLAGS ?= -X main.version=$(VERSION)
BUILDFLAGS ?= -a -ldflags '$(LDFLAGS)'
TEST_FLAGS ?= -count=1

UPX = upx
M4 = m4
M4_FLAGS =

DESTDIR ?= /
INSTALL_PREFIX ?= usr/local

GO ?= go
APPSOURCES := $(wildcard ./*.go activitypub/*.go internal/*/*.go storage/*/*.go)
ASSETFILES := $(wildcard templates/*)

TAGS := $(ENV) storage_$(STORAGE)

export CGO_ENABLED=0

ifeq ($(shell git describe --always > /dev/null 2>&1 ; echo $$?), 0)
	BRANCH=$(shell git rev-parse --abbrev-ref HEAD | tr '/' '-')
	HASH=$(shell git rev-parse --short HEAD)
	VERSION = $(shell printf "%s-%s" "$(BRANCH)" "$(HASH)")
endif
ifeq ($(shell git describe --tags > /dev/null 2>&1 ; echo $$?), 0)
	VERSION = $(shell git describe --tags | tr '/' '-')
endif

ifneq ($(ENV),dev)
	LDFLAGS += -s -w -extldflags "-static"
	BUILDFLAGS += -trimpath
endif

BUILD := $(GO) build $(BUILDFLAGS)
TEST := $(GO) test $(BUILDFLAGS)

.PHONY: all run clean test coverage integration install download help

.DEFAULT_GOAL := help

help: ## Help target that shows this message.
	@sed -rn 's/^([^:]+):.*[ ]##[ ](.+)/\1:\2/p' $(MAKEFILE_LIST) | column -ts: -l2

all: fedbox fedboxctl ##

download: go.sum ## Downloads dependencies and tidies the go.mod file.

go.sum: go.mod
	$(GO) mod download all
	$(GO) mod tidy

fedbox: bin/fedbox ## Builds the main FedBOX service binary.
bin/fedbox: go.mod go.sum cmd/fedbox/main.go $(APPSOURCES)
	$(BUILD) -tags "$(TAGS)" -o $@ ./cmd/fedbox/main.go
ifneq ($(ENV),dev)
	$(UPX) -q --mono --no-progress --best $@ || true
endif

fedboxctl: bin/fedboxctl ## Builds the control binary for the FedBOX service.
bin/fedboxctl: go.mod go.sum cmd/control/main.go $(APPSOURCES)
	$(BUILD) -tags "$(TAGS)" -o $@ ./cmd/control/main.go
ifneq ($(ENV),dev)
	$(UPX) -q --mono --no-progress --best $@ || true
endif

systemd/fedbox.service: systemd/fedbox.service.in ## Creates a systemd service file for the FedBOX service.
	$(M4) $(M4_FLAGS) -DWORKING_DIR=$(STORAGE_PATH) $< >$@

systemd/fedbox.socket: systemd/fedbox.socket.in ## Creates a socket systemd unit file to accompany the service file.
	$(M4) $(M4_FLAGS) -DLISTEN_HOST=$(LISTEN_HOST) -DLISTEN_PORT=$(LISTEN_PORT) $< >$@


run: fedbox ## Runs the FedBOX binary.
	@./bin/fedbox

clean: ## Cleanup the build workspace.
	-$(RM) bin/*
	$(MAKE) -C tests $@

test: TEST_TARGET := . ./{activitypub,storage,internal}/...
test: download ## Run unit tests for the service.
	$(TEST) $(TEST_FLAGS) -tags "$(TAGS)" $(TEST_TARGET)

coverage: TEST_TARGET := .
coverage: TEST_FLAGS += -covermode=count -coverprofile $(PROJECT).coverprofile
coverage: test ## Run unit tests for the service with coverage.

integration: download ## Run integration tests for the service.
	$(MAKE) -C tests $@

$(FEDBOX_HOSTNAME).key $(FEDBOX_HOSTNAME).crt:
	openssl req -subj "/C=AQ/ST=Omond/L=Omond/O=*.$(FEDBOX_HOSTNAME)/OU=none/CN=*.$(FEDBOX_HOSTNAME)" -newkey rsa:2048 -sha256 -keyout $(FEDBOX_HOSTNAME).key -nodes -x509 -days 365 -out $(FEDBOX_HOSTNAME).crt

$(FEDBOX_HOSTNAME).pem: $(FEDBOX_HOSTNAME).key $(FEDBOX_HOSTNAME).crt
	cat $(FEDBOX_HOSTNAME).key $(FEDBOX_HOSTNAME).crt > $(FEDBOX_HOSTNAME).pem

cert: $(FEDBOX_HOSTNAME).key ## Create a certificate.

install: ./bin/fedbox systemd/fedbox.service systemd/fedbox.socket $(FEDBOX_HOSTNAME).crt $(FEDBOX_HOSTNAME).key ## Install the application.
	useradd -m -s /bin/false -u 2000 fedbox
	install bin/fedbox $(DESTDIR)$(INSTALL_PREFIX)/bin
	install -m 644 -o fedbox systemd/fedbox.service $(DESTDIR)/etc/systemd/system
	install -m 644 -o fedbox systemd/fedbox.socket $(DESTDIR)/etc/systemd/system
	install -m 600 -o fedbox .env.prod $(STORAGE_PATH)
	install -m 600 -o $(FEDBOX_HOSTNAME).crt $(STORAGE_PATH)
	install -m 600 -o $(FEDBOX_HOSTNAME).key $(STORAGE_PATH)
