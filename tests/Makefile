export CGO_ENABLED=0
GO := go
TEST := $(GO) test -v
STORAGE ?= boltdb

.PHONY: test integration clean

clean:
	@-$(RM) -rf ./.cache/test*

c2s: clean
	mkdir -p .cache
	STORAGE=$(STORAGE) $(TEST) -tags "storage_all integration c2s" ./...

s2s: clean
	mkdir -p .cache
	STORAGE=$(STORAGE) $(TEST) -tags "storage_all integration s2s" ./...

test: c2s s2s

integration: test
