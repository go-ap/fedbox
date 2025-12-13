SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

ENV ?= dev
FEDBOX_HOSTNAME ?= fedbox
PORT ?= 4000
STORAGE ?= all
STORAGE_PATH ?= $(shell realpath .cache)
STORAGE_OBJECTS = $(STORAGE_PATH)/objects
TAG ?= $(ENV)
VERSION ?= HEAD


TAG_CMD=podman tag
PUSH_CMD=podman push

ifeq ($(shell git describe --always > /dev/null 2>&1 ; echo $$?), 0)
	BRANCH=$(shell git rev-parse --abbrev-ref HEAD | tr '/' '-')
	HASH=$(shell git rev-parse --short HEAD)
	VERSION = $(shell printf "%s-%s" "$(BRANCH)" "$(HASH)")
endif
ifeq ($(shell git describe --tags > /dev/null 2>&1 ; echo $$?), 0)
	VERSION = $(shell git describe --tags | tr '/' '-')
endif

ifneq ($(STORAGE),all)
	TAG=$(ENV)-$(STORAGE)
endif

.PHONY: clean build builder push cert

build:
	ENV=$(ENV) VERSION=$(VERSION) STORAGE=$(STORAGE) PORT=$(PORT) HOSTNAME=$(FEDBOX_HOSTNAME) ./image.sh $(FEDBOX_HOSTNAME)/app:$(TAG)

push: build
	$(TAG_CMD) "$(FEDBOX_HOSTNAME)/app:$(TAG)" "quay.io/go-ap/fedbox:$(TAG)"
	$(PUSH_CMD) quay.io/go-ap/fedbox:$(TAG)
ifeq ($(TAG),dev)
	$(TAG_CMD) "$(FEDBOX_HOSTNAME)/app:$(TAG)" quay.io/go-ap/fedbox:latest || true
	$(PUSH_CMD) quay.io/go-ap/fedbox:latest || true
endif
ifneq ($(VERSION),HEAD)
	$(TAG_CMD) "$(FEDBOX_HOSTNAME)/app:$(TAG)" "quay.io/go-ap/fedbox:$(VERSION)-$(TAG)" || true
	$(PUSH_CMD) quay.io/go-ap/fedbox:$(VERSION)-$(TAG) || true
endif
