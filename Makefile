SHELL := /bin/sh
REPO_ROOT := $(shell pwd)
PLUGIN_ROOT := $(REPO_ROOT)/plugins/archcore
PLUGIN_REL := plugins/archcore
BIN_SCRIPTS := $(wildcard $(PLUGIN_REL)/bin/check-* $(PLUGIN_REL)/bin/validate-* $(PLUGIN_REL)/bin/session-start) $(PLUGIN_REL)/bin/git-scope
LIB_SCRIPTS := $(PLUGIN_REL)/bin/lib/normalize-stdin.sh
ALL_SCRIPTS := $(BIN_SCRIPTS) $(LIB_SCRIPTS)
# Marketplace catalogs stay at repo root; plugin manifests/hooks/mcp live under plugins/archcore/.
JSON_FILES := .agents/plugins/marketplace.json .claude-plugin/marketplace.json .cursor-plugin/marketplace.json \
              $(PLUGIN_REL)/.claude-plugin/plugin.json $(PLUGIN_REL)/.cursor-plugin/plugin.json \
              $(PLUGIN_REL)/.codex-plugin/plugin.json $(PLUGIN_REL)/.plugin/plugin.json $(PLUGIN_REL)/.codex.mcp.json \
              $(PLUGIN_REL)/hooks/hooks.json $(PLUGIN_REL)/hooks/cursor.hooks.json $(PLUGIN_REL)/hooks/codex.hooks.json \
              $(PLUGIN_REL)/hooks/copilot.hooks.json \
              $(PLUGIN_REL)/.mcp.json docs/cursor.mcp.example.json

.PHONY: test test-codex-smoke lint check-json check-perms verify all

all: check-json check-perms lint test

test:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/unit/ test/structure/

test-unit:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/unit/

test-structure:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/structure/

test-codex-smoke:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/integration/codex-plugin-smoke.bats

lint:
	@command -v shellcheck >/dev/null 2>&1 || { echo "shellcheck not found, skipping"; exit 0; }
	@cd $(PLUGIN_ROOT)/bin && shellcheck -s sh -x $(addprefix $(REPO_ROOT)/,$(ALL_SCRIPTS))
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
