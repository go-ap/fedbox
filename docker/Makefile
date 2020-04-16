ENV ?= dev
HOSTNAME ?= fedbox
LISTEN ?= 4000

APPSOURCES := $(wildcard ../app/*.go ../activitypub/*.go ../internal/*/*.go ../cli/*/main.go ../cmd/*.go ../storage/*/*.go)

BUILD=buildah build-using-dockerfile
RUN=buildah run
TAG=buildah tag
PUSH=buildah push
#BUILD=docker build
#RUN=docker run
#TAG=docker tag
#PUSH=docker push

.PHONY: clean build run cert

$(HOSTNAME).key $(HOSTNAME).crt:
	openssl req -subj "/C=AQ/ST=Omond/L=Omond/O=$(HOSTNAME)/OU=none/CN=$(HOSTNAME)" \
	-newkey rsa:2048 -sha256 -keyout $(HOSTNAME).key -nodes -x509 -days 365 -out $(HOSTNAME).crt

$(HOSTNAME).pem: $(HOSTNAME).key $(HOSTNAME).crt
	cat $(HOSTNAME).key $(HOSTNAME).crt > $(HOSTNAME).pem

cert: $(HOSTNAME).pem

clean:
	@-$(RM) $(HOSTNAME).{key,crt,pem}

build:
	$(BUILD) --build-arg ENV=$(ENV) --build-arg LISTEN=:$(LISTEN) -f Dockerfile -t $(HOSTNAME)/app:$(ENV) ..

run: build
	$(RUN) -v $(realpath ../.cache):/storage -e ENV=$(ENV) -e HOSTNAME=$(HOSTNAME) $(HOSTNAME)/app:$(ENV) /bin/ctl bootstrap
	$(RUN) -v $(realpath ../.cache):/storage -e ENV=$(ENV) -e HOSTNAME=$(HOSTNAME) -p $(LISTEN):$(LISTEN) $(HOSTNAME)/app:$(ENV)

push: build
	$(TAG) $(HOSTNAME)/app:$(ENV) quay.io/fedbox/fedbox:latest
	$(TAG) $(HOSTNAME)/app:$(ENV) quay.io/fedbox/fedbox:$(ENV)
	$(PUSH) quay.io/fedbox/fedbox:latest
	$(PUSH) quay.io/fedbox/fedbox:$(ENV)
