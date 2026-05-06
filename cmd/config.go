package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/internal/display"

	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config [get|set] [key] [value]",
		Short: "View or modify archcore configuration",
		RunE:  runConfig,
	}
	return cmd
}

func runConfig(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	settings, err := config.Load(cwd)
	if err != nil {
		fmt.Println(display.FailLine("Settings not found or invalid"))
		fmt.Println(display.HintLine("Run 'archcore init' to set up"))
		return fmt.Errorf("settings not found: %w", err)
	}

	if len(args) == 0 {
		fmt.Println(display.KeyValue("sync", settings.Sync))
		return nil
	}

	switch args[0] {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: archcore config get <key>")
		}
		// "get sync" is allowed (read-only, safe to expose); "set sync" is blocked below.
		if args[1] == "project_id" || args[1] == "archcore_url" {
			return fmt.Errorf("%s is not available yet — sync features are coming soon", args[1])
		}
		val, err := getSettingsValue(settings, args[1])
		if err != nil {
			return err
		}
		fmt.Println(val)

	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: archcore config set <key> <value>")
		}
		if args[1] == "sync" || args[1] == "project_id" || args[1] == "archcore_url" {
			return fmt.Errorf("%s is not available yet — sync features are coming soon", args[1])
		}
		if err := setSettingsValue(settings, args[1], strings.Join(args[2:], " ")); err != nil {
			return err
		}
		if err := config.Save(cwd, settings); err != nil {
			return fmt.Errorf("saving settings: %w", err)
		}
		fmt.Println(display.CheckLine(fmt.Sprintf("Set %s = %s", args[1], strings.Join(args[2:], " "))))

	default:
		return fmt.Errorf("unknown subcommand %q — use 'get' or 'set'", args[0])
	}

	return nil
}

func getSettingsValue(s *config.Settings, key string) (string, error) {
	switch key {
	case "sync":
		return s.Sync, nil
	case "project_id":
		if s.Sync == config.SyncTypeNone {
			return "", fmt.Errorf("project_id is not available for sync type %q", config.SyncTypeNone)
		}
		if s.ProjectID == nil {
			return "null", nil
		}
		return strconv.Itoa(*s.ProjectID), nil
	case "archcore_url":
		if s.Sync != config.SyncTypeOnPrem {
			return "", fmt.Errorf("archcore_url is only available for sync type %q", config.SyncTypeOnPrem)
		}
		return s.ArchcoreURL, nil
	case "language":
		if s.Language == "" {
			return "en", nil
		}
		return s.Language, nil
	default:
		return "", fmt.Errorf("unknown config key %q — valid keys: sync, project_id, archcore_url, language", key)
	}
}

func setSettingsValue(s *config.Settings, key, value string) error {
	switch key {
	case "sync":
		switch value {
		case config.SyncTypeNone:
			s.Sync = value
			s.ProjectID = nil
			s.ArchcoreURL = ""
		case config.SyncTypeCloud:
			s.Sync = value
			s.ArchcoreURL = ""
		case config.SyncTypeOnPrem:
			if s.ArchcoreURL == "" {
				return fmt.Errorf("cannot switch to %q without archcore_url — run 'archcore config set archcore_url <url>' instead", config.SyncTypeOnPrem)
			}
			s.Sync = value
		default:
			return fmt.Errorf("invalid sync type %q — use %q, %q, or %q",
				value, config.SyncTypeNone, config.SyncTypeCloud, config.SyncTypeOnPrem)
		}
	case "project_id":
		if s.Sync == config.SyncTypeNone {
			return fmt.Errorf("cannot set project_id when sync is %q", config.SyncTypeNone)
		}
		if value == "null" {
			s.ProjectID = nil
		} else {
			pid, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("project_id must be \"null\" or a number, got %q", value)
			}
			s.ProjectID = &pid
		}
	case "archcore_url":
		if s.Sync != config.SyncTypeOnPrem {
			// Setting archcore_url implies on-prem sync mode.
			s.Sync = config.SyncTypeOnPrem
		}
		value = strings.TrimRight(value, "/")
		if value == "" {
			return fmt.Errorf("archcore_url must not be empty")
		}
		s.ArchcoreURL = value
	case "language":
		if value == "" {
			return fmt.Errorf("language must not be empty")
		}
		s.Language = value
	default:
		return fmt.Errorf("unknown config key %q — valid keys: sync, project_id, archcore_url, language", key)
	}
	return nil
}
