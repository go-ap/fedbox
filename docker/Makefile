ENV ?= dev
HOSTNAME ?= fedbox
PORT ?= 4000
STORAGE ?= $(shell realpath .cache)
STORAGE_OBJECTS = $(STORAGE)/objects

APPSOURCES := $(wildcard ../app/*.go ../activitypub/*.go ../internal/*/*.go ../cli/*/main.go ../cmd/*.go ../storage/*/*.go)

#BUILD=buildah build-using-dockerfile
#RUN=buildah run
#TAG=buildah tag
#PUSH=buildah push
BUILD=docker build
RUN=docker run
TAG=docker tag
PUSH=docker push

.PHONY: clean build run cert

$(HOSTNAME).key $(HOSTNAME).crt:
	openssl req -subj "/C=AQ/ST=Omond/L=Omond/O=$(HOSTNAME)/OU=none/CN=$(HOSTNAME)" \
	-newkey rsa:2048 -sha256 -keyout $(HOSTNAME).key -nodes -x509 -days 365 -out $(HOSTNAME).crt

$(HOSTNAME).pem: $(HOSTNAME).key $(HOSTNAME).crt
	cat $(HOSTNAME).key $(HOSTNAME).crt > $(HOSTNAME).pem

cert: $(HOSTNAME).pem

clean:
	@-$(RM) $(HOSTNAME).{key,crt,pem}

build: cert
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
		-v $(realpath ../.env.dev):/.env \
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
