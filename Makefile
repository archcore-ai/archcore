SHELL := /bin/sh
REPO_ROOT := $(shell pwd)
PLUGIN_ROOT := $(REPO_ROOT)/plugins/archcore
PLUGIN_REL := plugins/archcore
BIN_SCRIPTS := $(PLUGIN_REL)/bin/session-start $(PLUGIN_REL)/bin/pre-tool-use $(PLUGIN_REL)/bin/post-tool-use $(PLUGIN_REL)/bin/detect-host $(PLUGIN_REL)/bin/cli-gte
LIB_SCRIPTS := $(PLUGIN_REL)/bin/lib/normalize-stdin.sh $(PLUGIN_REL)/bin/lib/plugin-cache-guard.sh
ALL_SCRIPTS := $(BIN_SCRIPTS) $(LIB_SCRIPTS)
TEST_SH_SCRIPTS := test/behavioral/route-bench.sh
TEST_BASH_SCRIPTS := test/helpers/mcp.bash
ARCHCORE_BIN ?= archcore
# Marketplace catalogs stay at repo root; plugin manifests/hooks/mcp live under plugins/archcore/.
JSON_FILES := .agents/plugins/marketplace.json .claude-plugin/marketplace.json .cursor-plugin/marketplace.json \
              $(PLUGIN_REL)/.claude-plugin/plugin.json $(PLUGIN_REL)/.cursor-plugin/plugin.json \
              $(PLUGIN_REL)/.codex-plugin/plugin.json $(PLUGIN_REL)/.plugin/plugin.json $(PLUGIN_REL)/.codex.mcp.json \
              $(PLUGIN_REL)/hooks/hooks.json $(PLUGIN_REL)/hooks/cursor.hooks.json $(PLUGIN_REL)/hooks/codex.hooks.json \
              $(PLUGIN_REL)/hooks/copilot.hooks.json \
              $(PLUGIN_REL)/.claude.mcp.json docs/cursor.mcp.example.json

.PHONY: test test-unit test-structure test-integration test-routing-bench test-research-agent test-codex-smoke test-copilot-smoke lint check-json check-perms verify all

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

# Requires the installed CLI. No host account or model calls are needed.
test-integration:
	@command -v "$(ARCHCORE_BIN)" >/dev/null 2>&1 || { echo "Install Archcore CLI >= 0.8.3 or set ARCHCORE_BIN"; exit 1; }
	@ARCHCORE_BIN="$(ARCHCORE_BIN)" PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/integration/research-vocabulary.bats test/integration/cursor-post-tool-use.bats

# LLM-in-the-loop routing bench — spends model tokens; on demand only, never CI.
test-routing-bench:
	@sh test/behavioral/route-bench.sh

# Live model + MCP behavior, explicitly requested and never part of CI.
test-research-agent:
	@command -v "$(ARCHCORE_BIN)" >/dev/null 2>&1 || { echo "Install Archcore CLI >= 0.8.3 or set ARCHCORE_BIN"; exit 1; }
	@ARCHCORE_BIN="$(ARCHCORE_BIN)" PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/integration/research-agent.bats

test-codex-smoke:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/integration/codex-plugin-smoke.bats

test-copilot-smoke:
	@command -v bats >/dev/null 2>&1 || { echo "bats-core not found. Install: brew install bats-core"; exit 1; }
	@PLUGIN_ROOT=$(PLUGIN_ROOT) REPO_ROOT=$(REPO_ROOT) bats test/integration/copilot-plugin-smoke.bats

lint:
	@command -v shellcheck >/dev/null 2>&1 || { echo "shellcheck not found, skipping"; exit 0; }
	@cd $(PLUGIN_ROOT)/bin && shellcheck -s sh -x $(addprefix $(REPO_ROOT)/,$(ALL_SCRIPTS))
	@shellcheck -s sh $(TEST_SH_SCRIPTS)
	@shellcheck -s bash $(TEST_BASH_SCRIPTS)
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

verify: all test-integration
	@echo "All checks passed"
