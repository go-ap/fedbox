GO := go
TEST_FLAGS ?= -count=1
STORAGE ?= all

ENV ?= test
BUILDFLAGS ?= -a
TAGS = $(ENV) storage_$(STORAGE) integration
TEST_TARGET = ./

TEST := $(GO) test $(BUILDFLAGS)

.PHONY: test integration clean coverage

.cache:
	mkdir -p .cache

clean:
	@-$(RM) -rf ./.cache/$(ENV)
	@-$(RM) -rf ./.cache/*.bdb

c2s: clean .cache
	$(TEST) $(TEST_FLAGS) -tags "$(TAGS) c2s" $(TEST_TARGET)

s2s: clean .cache
	$(TEST) $(TEST_FLAGS) -tags "$(TAGS) s2s" $(TEST_TARGET)

test: c2s s2s

coverage:
	$(TEST) $(TEST_FLAGS) -tags "$(TAGS) c2s s2s" -coverpkg github.com/go-ap/fedbox,github.com/go-ap/fedbox/internal/assets,github.com/go-ap/fedbox/activitypub -covermode=count -args -test.gocoverdir="$(PWD)/tests/.cache" $(TEST_TARGET)

integration: coverage
