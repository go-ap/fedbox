export CGO_ENABLED=0
GO := go
TEST := $(GO) test -v
STORAGE ?= boltdb

.PHONY: test integration clean

clean:
	@-$(RM) -rf ./.cache/test*

test: clean
	mkdir -p .cache
	STORAGE=$(STORAGE) $(TEST) ./...

integration: test
