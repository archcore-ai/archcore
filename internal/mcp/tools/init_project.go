package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"archcore-cli/internal/config"
)

func NewInitProjectTool() mcp.Tool {
	return mcp.NewTool("init_project",
		mcp.WithDescription(`Initialize the .archcore/ knowledge base for the current project.

Call this tool ONCE per project, before creating any documents, if list_documents reports the project is not yet initialized (empty result with no .archcore/ directory). It is safe to call on an already-initialized project — in that case the existing configuration is preserved and returned.

What it does:
- Creates the .archcore/ directory
- Writes .archcore/settings.json with the chosen sync mode and language

What it does NOT do:
- Does not install hooks or MCP configs for other agents (use 'archcore hooks install' or 'archcore mcp install' from the shell for that)
- Does not register the MCP server — that is handled by the host plugin or by 'archcore mcp install'

Returns: JSON with { initialized: true, settings: {...}, already_initialized: bool }.`),
		mcp.WithString("language",
			mcp.Description(`Optional BCP-47 language code (e.g. "en", "ru", "ja") for generated document content. Frontmatter keys and status values remain English. Defaults to "en" when omitted.`),
		),
		mcp.WithString("sync_mode",
			mcp.Description(`Sync mode for the project. "none" keeps everything local (default). "cloud" and "on-prem" require additional setup via the CLI and are typically configured later — prefer "none" for in-session initialization.`),
			mcp.Enum(config.ValidSyncTypeStrings()...),
		),
		mcp.WithString("archcore_url",
			mcp.Description(`Required only when sync_mode="on-prem". The URL of the on-prem Archcore server.`),
		),
		mcp.WithTitleAnnotation("Initialize Project"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
}

func HandleInitProject(baseDir string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		language := strings.TrimSpace(request.GetString("language", ""))
		if strings.Contains(language, " ") {
			return errorResult(`invalid language: must not contain spaces`), nil
		}
		if language == "en" {
			language = ""
		}

		syncMode := config.SyncType(strings.TrimSpace(request.GetString("sync_mode", "")))
		if syncMode == "" {
			syncMode = config.SyncTypeNone
		}

		existing, err := config.Load(baseDir)
		switch {
		case err == nil:
			return initResultPayload(existing, true), nil
		case !errors.Is(err, os.ErrNotExist):
			return errorResult(fmt.Sprintf("existing settings unreadable: %v", err)), nil
		}

		var settings *config.Settings
		switch syncMode {
		case config.SyncTypeNone:
			settings = config.NewNoneSettings()
		case config.SyncTypeCloud:
			settings = config.NewCloudSettings()
		case config.SyncTypeOnPrem:
			url := strings.TrimSpace(request.GetString("archcore_url", ""))
			if url == "" {
				return errorResult(`sync_mode="on-prem" requires archcore_url`), nil
			}
			settings = config.NewOnPremSettings(url)
		default:
			return errorResult(fmt.Sprintf("unknown sync_mode %q (valid: %s)", syncMode, strings.Join(config.ValidSyncTypeStrings(), ", "))), nil
		}

		settings.Language = language

		if err := config.Save(baseDir, settings); err != nil {
			return errorResult(fmt.Sprintf("saving settings: %v", err)), nil
		}

		return initResultPayload(settings, false), nil
	}
}

func initResultPayload(s *config.Settings, alreadyInitialized bool) *mcp.CallToolResult {
	payload := map[string]any{
		"initialized":         true,
		"already_initialized": alreadyInitialized,
		"settings":            s,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return errorResult(fmt.Sprintf("marshaling result: %v", err))
	}
	return mcp.NewToolResultText(string(data))
}
