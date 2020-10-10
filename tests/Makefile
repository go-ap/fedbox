export CGO_ENABLED=0
GO := go
TEST := $(GO) test -v
STORAGE ?= all

.PHONY: test integration clean

clean:
	@-$(RM) -rf ./.cache/test*

c2s: clean
	mkdir -p .cache
	$(TEST) -tags "storage_$(STORAGE) integration c2s" ./...

s2s: clean
	mkdir -p .cache
	$(TEST) -tags "storage_$(STORAGE) integration s2s" ./...

test: c2s s2s

integration: test
