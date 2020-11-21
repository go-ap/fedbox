ENV ?= dev
HOSTNAME ?= fedbox
PORT ?= 4000
STORAGE ?= all
STORAGE_PATH ?= $(shell realpath .cache)
STORAGE_OBJECTS = $(STORAGE_PATH)/objects
TAG ?= $(ENV)

APPSOURCES := $(wildcard ../app/*.go ../activitypub/*.go ../internal/*/*.go ../cli/*/main.go ../cmd/*.go ../storage/*/*.go)

HAVE_BUILDAH := $(shell buildah -v dot 2> /dev/null)
ifdef HAVE_BUILDAH
	BUILD_CMD=buildah bud
	RUN_CMD=podman run
	TAG_CMD=buildah tag
	PUSH_CMD=buildah push
else
	BUILD_CMD=docker build
	RUN_CMD=docker run
	TAG_CMD=docker tag
	PUSH_CMD=docker push
endif

ifneq ($(STORAGE), all)
	TAG=$(ENV)-$(STORAGE)
endif

.PHONY: clean build run cert

$(HOSTNAME).pem:
	./gen-certs.sh $(HOSTNAME)

cert: $(HOSTNAME).pem

clean:
	@-$(RM) $(HOSTNAME).{key,crt,pem}

build:
	$(BUILD_CMD) --build-arg ENV=$(ENV) --build-arg STORAGE=$(STORAGE) --build-arg PORT=$(PORT) -f Dockerfile -t $(HOSTNAME)/app:$(TAG) ..

storage: $(STORAGE_PATH)

$(STORAGE_PATH):
	echo $(STORAGE_PATH)
	echo $(STORAGE_OBJECTS)
	mkdir -p $(STORAGE_PATH)

$(STORAGE_OBJECTS): bootstrap

bootstrap: storage
	$(RUN_CMD) -v $(STORAGE_PATH):/storage \
		-e ENV=$(ENV) \
		-e HOSTNAME=$(HOSTNAME) \
		$(HOSTNAME)/app:$(TAG) \
		/bin/ctl bootstrap

run: build $(STORAGE_OBJECTS)
	$(RUN_CMD) -v $(STORAGE_PATH):/storage \
		-v $(realpath ./.env.default):/.env \
		-p $(PORT):$(PORT) \
		-e ENV=$(ENV) \
		-e LISTEN=:$(PORT) \
		-e HOSTNAME=$(HOSTNAME) \
		$(HOSTNAME)/app:$(TAG)

push: build
	$(TAG_CMD) $(HOSTNAME)/app:$(TAG) quay.io/fedbox/fedbox:$(TAG)
	$(PUSH_CMD) quay.io/fedbox/fedbox:$(TAG)
ifeq ($(TAG), dev)
	$(TAG_CMD) $(HOSTNAME)/app:$(TAG) quay.io/fedbox/fedbox:latest || true
	$(PUSH_CMD) quay.io/fedbox/fedbox:latest || true
endif
ifneq ($(VERSION), )
	$(TAG_CMD) $(HOSTNAME)/app:$(TAG) quay.io/fedbox/fedbox:$(VERSION)-$(TAG) || true
	$(PUSH_CMD) quay.io/fedbox/fedbox:$(VERSION)-$(TAG) || true
endif
