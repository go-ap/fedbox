SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:

FEDBOX_HOSTNAME ?= fedbox.git
STORAGE ?= all
ENV ?= dev
LDFLAGS ?= -X main.version=$(VERSION)
BUILDFLAGS ?= -trimpath -a -ldflags '$(LDFLAGS)'
TEST_FLAGS ?= -count=1
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

M4 = /usr/bin/m4
M4_FLAGS =

DESTDIR ?= /
INSTALL_PREFIX ?= usr/local

GO := go
APPSOURCES := $(wildcard app/*.go activitypub/*.go internal/*/*.go storage/*/*.go)
ASSETFILES := $(wildcard templates/*)
PROJECT_NAME := $(shell basename $(PWD))
APPSOURCES += internal/assets/assets.gen.go
TAGS := $(ENV) storage_$(STORAGE)

export CGO_ENABLED=0

ifneq ($(ENV), dev)
	LDFLAGS += -s -w -extldflags "-static"
endif

ifeq ($(VERSION), )
	ifeq ($(shell git describe --always > /dev/null 2>&1 ; echo $$?), 0)
		BRANCH=$(shell git rev-parse --abbrev-ref HEAD | tr '/' '-')
		HASH=$(shell git rev-parse --short HEAD)
		VERSION ?= $(shell printf "%s-%s" "$(BRANCH)" "$(HASH)")
	endif
	ifeq ($(shell git describe --tags > /dev/null 2>&1 ; echo $$?), 0)
		VERSION ?= $(shell git describe --tags | tr '/' '-')
	endif
endif

BUILD := $(GO) build $(BUILDFLAGS)
TEST := $(GO) test $(BUILDFLAGS)

.PHONY: all run clean test coverage integration install download

all: fedbox ctl

download:
	$(GO) mod download
	$(GO) mod tidy

assets: internal/assets/assets.gen.go

internal/assets/assets.gen.go: download $(ASSETFILES)
	$(GO) generate -tags "$(TAGS)" ./assets.go

fedbox: bin/fedbox assets
bin/fedbox: go.mod cmd/fedbox/main.go $(APPSOURCES)
	$(BUILD) -tags "$(TAGS)" -o $@ ./cmd/fedbox/main.go

systemd/fedbox.service: systemd/fedbox.service.in
	$(M4) -DWORKING_DIR=$(STORAGE_PATH) $< >$@

systemd/fedbox.socket: systemd/fedbox.socket.in
	$(M4) -DLISTEN_HOST=$(LISTEN_HOST) -DLISTEN_PORT=$(LISTEN_PORT) $< >$@

ctl: bin/ctl
bin/ctl: go.mod cmd/control/main.go $(APPSOURCES)
	$(BUILD) -tags "$(TAGS)" -o $@ ./cmd/control/main.go

run: fedbox
	@./bin/fedbox

clean:
	-$(RM) bin/*
	$(MAKE) -C tests $@

test: TEST_TARGET := ./{activitypub,app,storage,internal}/...
test: assets
	$(TEST) $(TEST_FLAGS) -tags "$(TAGS)" $(TEST_TARGET)

coverage: TEST_TARGET := .
coverage: TEST_FLAGS += -covermode=count -coverprofile $(PROJECT_NAME).coverprofile
coverage: test

integration: download
	$(MAKE) -C tests $@

$(FEDBOX_HOSTNAME).key $(FEDBOX_HOSTNAME).crt:
	openssl req -subj "/C=AQ/ST=Omond/L=Omond/O=*.$(FEDBOX_HOSTNAME)/OU=none/CN=*.$(FEDBOX_HOSTNAME)" -newkey rsa:2048 -sha256 -keyout $(FEDBOX_HOSTNAME).key -nodes -x509 -days 365 -out $(FEDBOX_HOSTNAME).crt

$(FEDBOX_HOSTNAME).pem: $(FEDBOX_HOSTNAME).key $(FEDBOX_HOSTNAME).crt
	cat $(FEDBOX_HOSTNAME).key $(FEDBOX_HOSTNAME).crt > $(FEDBOX_HOSTNAME).pem

cert: $(FEDBOX_HOSTNAME).key

install: ./bin/fedbox systemd/fedbox.service systemd/fedbox.socket $(FEDBOX_HOSTNAME).crt $(FEDBOX_HOSTNAME).key
	useradd -m -s /bin/false -u 2000 fedbox
	install bin/fedbox $(DESTDIR)$(INSTALL_PREFIX)/bin
	install -m 644 -o fedbox systemd/fedbox.service $(DESTDIR)/etc/systemd/system
	install -m 644 -o fedbox systemd/fedbox.socket $(DESTDIR)/etc/systemd/system
	install -m 600 -o fedbox .env.prod $(STORAGE_PATH)
	install -m 600 -o $(FEDBOX_HOSTNAME).crt $(STORAGE_PATH)
	install -m 600 -o $(FEDBOX_HOSTNAME).key $(STORAGE_PATH)
