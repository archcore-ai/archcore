SHELL := /bin/sh
PLUGIN_ROOT := $(shell pwd)
BIN_SCRIPTS := $(wildcard bin/check-* bin/validate-* bin/session-start) bin/archcore
LIB_SCRIPTS := bin/lib/normalize-stdin.sh
ALL_SCRIPTS := $(BIN_SCRIPTS) $(LIB_SCRIPTS)
JSON_FILES := .claude-plugin/plugin.json .claude-plugin/marketplace.json \
              .cursor-plugin/plugin.json .cursor-plugin/marketplace.json \
              .codex-plugin/plugin.json .codex.mcp.json .agents/plugins/marketplace.json \
              hooks/hooks.json hooks/cursor.hooks.json hooks/codex.hooks.json .mcp.json

.PHONY: test test-codex-smoke lint check-json check-perms verify all

all: check-json check-perms lint test

test:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) bats test/unit/ test/structure/

test-unit:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) bats test/unit/

test-structure:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) bats test/structure/

test-codex-smoke:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) bats test/integration/codex-plugin-smoke.bats

lint:
	@command -v shellcheck >/dev/null 2>&1 || { echo "shellcheck not found, skipping"; exit 0; }
	@cd $(PLUGIN_ROOT)/bin && shellcheck -s sh -x $(addprefix $(PLUGIN_ROOT)/,$(ALL_SCRIPTS))
	@echo "ShellCheck: all clean"

check-json:
	@fail=0; for f in $(JSON_FILES); do \
	  jq . < "$$f" > /dev/null 2>&1 || { echo "FAIL: $$f is not valid JSON"; fail=1; }; \
	done; \
	[ $$fail -eq 0 ] && echo "JSON: all valid" || exit 1

check-perms:
	@fail=0; for f in $(BIN_SCRIPTS); do \
	  [ -x "$$f" ] || { echo "FAIL: $$f not executable"; fail=1; }; \
	done; \
	[ $$fail -eq 0 ] && echo "Permissions: all OK" || exit 1

verify: all
	@echo "All checks passed"
