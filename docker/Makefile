ENV ?= dev
HOSTNAME ?= fedbox
PORT ?= 4000
STORAGE ?= $(shell realpath .cache)
STORAGE_OBJECTS = $(STORAGE)/objects

APPSOURCES := $(wildcard ../app/*.go ../activitypub/*.go ../internal/*/*.go ../cli/*/main.go ../cmd/*.go ../storage/*/*.go)

HAVE_BUILDAH := $(shell buildah1 -v dot 2> /dev/null)
ifdef HAVE_BUILDAH
	BUILD=buildah bud
	RUN=docker run
	TAG=buildah tag
	PUSH=buildah push
else
	BUILD=docker build
	RUN=docker run
	TAG=docker tag
	PUSH=docker push
endif

.PHONY: clean build run cert

$(HOSTNAME).pem:
	./gen-certs.sh $(HOSTNAME)

cert: $(HOSTNAME).pem

clean:
	@-$(RM) $(HOSTNAME).{key,crt,pem}

build:
	$(BUILD) --build-arg ENV=$(ENV) --build-arg PORT=$(PORT) -f Dockerfile -t $(HOSTNAME)/app:$(ENV) ..

storage: $(STORAGE)

$(STORAGE):
	echo $(STORAGE)
	echo $(STORAGE_OBJECTS)
	mkdir -p $(STORAGE)

$(STORAGE_OBJECTS): bootstrap

bootstrap: storage
	$(RUN) -v $(STORAGE):/storage \
		-e ENV=$(ENV) \
		-e HOSTNAME=$(HOSTNAME) \
		$(HOSTNAME)/app:$(ENV) \
		/bin/ctl bootstrap

run: build $(STORAGE_OBJECTS)
	$(RUN) -v $(STORAGE):/storage \
		-v $(realpath ./.env.default):/.env \
		-p $(PORT):$(PORT) \
		-e ENV=$(ENV) \
		-e LISTEN=:$(PORT) \
		-e HOSTNAME=$(HOSTNAME) \
		$(HOSTNAME)/app:$(ENV)

push: build
	$(TAG) $(HOSTNAME)/app:$(ENV) quay.io/fedbox/fedbox:latest
	$(TAG) $(HOSTNAME)/app:$(ENV) quay.io/fedbox/fedbox:$(ENV)
	$(PUSH) quay.io/fedbox/fedbox:latest
	$(PUSH) quay.io/fedbox/fedbox:$(ENV)
