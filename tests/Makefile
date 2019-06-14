export CGO_ENABLED=0
GO := go
TEST := $(GO) test -v

.PHONY: test integration clean

clean:
	@-$(RM) -rf ./test-*.bdb

test: clean
	$(TEST) ./...

integration: test
