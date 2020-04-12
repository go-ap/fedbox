export CGO_ENABLED=0
GO := go
TEST := $(GO) test -v

.PHONY: test integration clean

clean:
	@-$(RM) -rf ./.cache/test*

test: clean
	mkdir -p .cache
	$(TEST) ./...

integration: test
