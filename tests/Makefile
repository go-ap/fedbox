GO := go
TEST_FLAGS ?= -count=1
STORAGE ?= all

ENV ?= test
BUILDFLAGS ?= -a
TAGS = $(ENV) storage_$(STORAGE) integration
TEST_TARGET = ./

TEST := $(GO) test $(BUILDFLAGS)

.PHONY: test integration clean

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

integration: test
