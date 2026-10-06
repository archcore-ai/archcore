package advisory

import (
	"strings"
	"testing"

	"archcore-cli/templates"
)

func TestPrecisionFindings_KeywordAndLineAnchor(t *testing.T) {
	t.Parallel()
	longBody := strings.Repeat("This body is long enough to clear the placeholder floor. ", 6)
	fm := templates.Frontmatter{Title: "T", Status: templates.StatusDraft}
	spec := func(clause string) string {
		return "## Purpose\n" + longBody + "\n## Surface\ns\n## Normative Behavior\n" + clause + "\n## Conformance\nc\n"
	}
	adr := func(context string) string {
		return "## Context\n" + longBody + " " + context + "\n## Decision\nd\n## Alternatives Considered\na\n## Consequences\nc\n"
	}

	tests := []struct {
		name      string
		docType   templates.DocumentType
		body      string
		wantHit   string
		wantNoHit string
	}{
		{
			name: "russian clause with должен names the word", docType: templates.TypeSpec,
			body:    spec("1. Компонент должен отправлять событие `view`."),
			wantHit: `requirement with no BCP 14 keyword ("должен" in`,
		},
		{
			name: "german clause with muss names the word", docType: templates.TypeSpec,
			body:    spec("1. Der Dienst muss die Quittung senden."),
			wantHit: `"muss" in`,
		},
		{
			name: "japanese clause names the word without spaces", docType: templates.TypeSpec,
			body:    spec("1. サービスは領収書を送信しなければならない。"),
			wantHit: `"しなければならない" in`,
		},
		{
			name: "lowercase english must is named", docType: templates.TypeRule,
			body:    "## Rule\n1. Every handler in `src/**/*.ts` must start with `handle`.\n## Rationale\n" + longBody + "\n## Enforcement\nmanual review\n",
			wantHit: `"must" in`,
		},
		{
			name: "descriptive clause without any modal is reported", docType: templates.TypeSpec,
			body:    spec("1. Обработчики именуются с префиксом `handle`."),
			wantHit: "requirement with no BCP 14 keyword",
		},
		{
			name: "english keyword inside russian clause passes", docType: templates.TypeSpec,
			body:      spec("1. WHEN заказ оплачен, сервис оплаты MUST отправить чек."),
			wantNoHit: "requirement with no BCP 14 keyword",
		},
		{
			name: "a word containing a native modal is not one", docType: templates.TypeSpec,
			body:      spec("1. WHEN пользователь может выбрать, the form MUST показать список."),
			wantNoHit: "requirement with no BCP 14 keyword",
		},
		{
			name: "backticked must is named, not used", docType: templates.TypeSpec,
			body:      spec("1. The parser reads the `must` field."),
			wantHit:   "requirement with no BCP 14 keyword (1. The parser",
			wantNoHit: `"must" in`,
		},
		{
			name: "WHERE opens a clause like WHEN", docType: templates.TypeSpec,
			body:      spec("1. WHERE the beta flag is enabled, the page MUST show the banner if the user is new."),
			wantNoHit: "condition after the obligation",
		},
		{
			name: "file with a line number is reported", docType: templates.TypeADR,
			body:    adr("The handler lives in `server/headers-middleware.ts:34-40`."),
			wantHit: "code reference with a line number (server/headers-middleware.ts:34-40)",
		},
		{
			name: "at-path with a line is reported", docType: templates.TypeDoc,
			body:    "## Overview\n" + longBody + " See @internal/mcp/server.go:120.\n",
			wantHit: "@internal/mcp/server.go:120",
		},
		{
			name: "github-style fragment is reported", docType: templates.TypeDoc,
			body:    "## Overview\n" + longBody + " See src/app.tsx#L12-L20.\n",
			wantHit: "src/app.tsx#L12-L20",
		},
		{
			name: "file without a line passes", docType: templates.TypeDoc,
			body:      "## Overview\n" + longBody + " See @internal/mcp/server.go and `src/auth/`.\n",
			wantNoHit: "code reference with a line number",
		},
		{
			name: "host and port is not a file", docType: templates.TypeDoc,
			body:      "## Overview\n" + longBody + " Redis listens on `cache.example.com:6379`.\n",
			wantNoHit: "code reference with a line number",
		},
		{
			name: "a fenced sample keeps its line", docType: templates.TypeDoc,
			body:      "## Overview\n" + longBody + "\n```\npanic at main.go:12\n```\n",
			wantNoHit: "code reference with a line number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := strings.Join(PrecisionFindings(tt.docType, fm, tt.body), "\n")
			if tt.wantHit != "" && !strings.Contains(got, tt.wantHit) {
				t.Errorf("want a finding containing %q, got:\n%s", tt.wantHit, got)
			}
			if tt.wantNoHit != "" && strings.Contains(got, tt.wantNoHit) {
				t.Errorf("want no finding containing %q, got:\n%s", tt.wantNoHit, got)
			}
		})
	}
}
