ENV ?= dev
HOSTNAME ?= fedbox

APPSOURCES := $(wildcard ../app/*.go ../activitypub/*.go ../internal/*/*.go ../cli/*/main.go ../cmd/*.go ../storage/*/*.go)

#BUILD=buildah build-using-dockerfile
#RUN=buildah run
BUILD=docker build
RUN=docker run

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
	$(BUILD) --build-arg ENV=$(ENV) -f Dockerfile -t $(HOSTNAME)/app:$(ENV) ..

run: build
	$(RUN) -v $(realpath ../.cache):/storage -e ENV=$(ENV) -e HOSTNAME=$(HOSTNAME) $(HOSTNAME)/app:$(ENV) /bin/ctl bootstrap
	$(RUN) -v $(realpath ../.cache):/storage -e ENV=$(ENV) -e HOSTNAME=$(HOSTNAME) -p 4000:4000 $(HOSTNAME)/app:$(ENV)

push: build
	docker tag $(HOSTNAME)/app:$(ENV) quay.io/fedbox/fedbox:latest
	docker tag $(HOSTNAME)/app:$(ENV) quay.io/fedbox/fedbox:$(ENV)
	docker push quay.io/fedbox/fedbox:latest
	docker push quay.io/fedbox/fedbox:$(ENV)
