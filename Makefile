export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
export VERSION=(unknown)
GO := go
ENV ?= dev
LDFLAGS ?= -X main.version=$(VERSION)
BUILDFLAGS ?= -a -ldflags '$(LDFLAGS)'
APPSOURCES := $(wildcard ./app/*.go storage/*/*.go activitypub/*.go internal/*/*.go)
PROJECT_NAME := $(shell basename $(PWD))

ifneq ($(ENV), dev)
	LDFLAGS += -s -w -extldflags "-static"
endif

ifeq ($(shell git describe --always > /dev/null 2>&1 ; echo $$?), 0)
export VERSION = $(shell git describe --always)
endif
ifeq ($(shell git describe --tags > /dev/null 2>&1 ; echo $$?), 0)
export VERSION = $(shell git describe --tags)
endif

BUILD := $(GO) build $(BUILDFLAGS)
TEST := $(GO) test $(BUILDFLAGS)

.PHONY: all run clean test coverage integration

all: app boltbootstrap

app: bin/app
bin/app: go.mod main.go $(APPSOURCES)
	$(BUILD) -tags $(ENV) -o $@ ./main.go

boltbootstrap: bin/boltbootstrap $(wildcard storage/boltdb/*.go)
bin/boltbootstrap: go.mod cli/boltbootstrap/main.go
	$(BUILD) -tags $(ENV) -o $@ cli/boltbootstrap/main.go

run: app
	@./bin/app

clean:
	-$(RM) bin/*
	$(MAKE) -C tests $@

test:
	$(TEST) ./{app,internal,storage,cli}/...

coverage: TEST_FLAGS += -covermode=count -coverprofile $(PROJECT_NAME).coverprofile
coverage: test

integration:
	$(MAKE) -C tests $@
