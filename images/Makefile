ENV ?= prod
FEDBOX_HOSTNAME ?= fedbox
PORT ?= 4000
STORAGE ?= all
STORAGE_PATH ?= $(shell realpath .cache)
STORAGE_OBJECTS = $(STORAGE_PATH)/objects
TAG ?= $(ENV)

BUILD_CMD=buildah bud
RUN_CMD=podman run
TAG_CMD=podman tag
PUSH_CMD=podman push

ifneq ($(STORAGE), all)
	TAG=$(ENV)-$(STORAGE)
endif

.PHONY: clean build builder run push cert storage bootstrap

$(FEDBOX_HOSTNAME).pem:
	./gen-certs.sh $(FEDBOX_HOSTNAME)

cert: $(FEDBOX_HOSTNAME).pem

clean:
	@-$(RM) $(FEDBOX_HOSTNAME).{key,crt,pem}

builder:
	#$(BUILD_CMD) --build-arg HOSTNAME=$(FEDBOX_HOSTNAME) -f Dockerfile.build -t fedbox/builder ..
	./build.sh .. fedbox/builder

build:
	#$(BUILD_CMD) --build-arg ENV=$(ENV) --build-arg VERSION=$(VERSION) --build-arg STORAGE=$(STORAGE) --build-arg PORT=$(PORT) -f Dockerfile -t $(FEDBOX_HOSTNAME)/app:$(TAG) ..
	ENV=$(ENV) VERSION=$(VERSION) STORAGE=$(STORAGE) PORT=$(PORT) HOSTNAME=$(FEDBOX_HOSTNAME) ./image.sh $(FEDBOX_HOSTNAME)/app:$(TAG)

storage: $(STORAGE_PATH)

$(STORAGE_PATH):
	echo $(STORAGE_PATH)
	echo $(STORAGE_OBJECTS)
	mkdir -p $(STORAGE_PATH)

$(STORAGE_OBJECTS): bootstrap

bootstrap: storage
	$(RUN_CMD) -v $(STORAGE_PATH):/storage \
		-e ENV=$(ENV) \
		-e FEDBOX_HOSTNAME=$(FEDBOX_HOSTNAME) \
		$(FEDBOX_HOSTNAME)/app:$(TAG) \
		/bin/fedboxctl bootstrap

run: build $(STORAGE_OBJECTS)
	$(RUN_CMD) -v $(STORAGE_PATH):/storage \
		-v $(realpath ./.env.default):/.env \
		-p $(PORT):$(PORT) \
		-e ENV=$(ENV) \
		-e LISTEN=:$(PORT) \
		-e FEDBOX_HOSTNAME=$(FEDBOX_HOSTNAME) \
		$(FEDBOX_HOSTNAME)/app:$(TAG)

push: build
	$(TAG_CMD) $(FEDBOX_HOSTNAME)/app:$(TAG) quay.io/go-ap/fedbox:$(TAG)
	$(PUSH_CMD) quay.io/go-ap/fedbox:$(TAG)
ifeq ($(TAG), dev)
	$(TAG_CMD) $(FEDBOX_HOSTNAME)/app:$(TAG) quay.io/go-ap/fedbox:latest || true
	$(PUSH_CMD) quay.io/go-ap/fedbox:latest || true
endif
ifneq ($(VERSION), )
	$(TAG_CMD) $(FEDBOX_HOSTNAME)/app:$(TAG) quay.io/go-ap/fedbox:$(VERSION)-$(TAG) || true
	$(PUSH_CMD) quay.io/go-ap/fedbox:$(VERSION)-$(TAG) || true
endif
