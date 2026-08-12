// Command archcore is the Archcore CLI: it manages a local .archcore/ directory
// of structured Markdown documents and serves them to AI coding agents over MCP
// and lifecycle hooks. This file owns process concerns only — signal handling
// and the exit code; every command lives in the cmd package.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"archcore-cli/cmd"
)

var version = "dev"

func resolveVersion() {
	if version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	resolveVersion()

	if err := cmd.NewRootCmd(version).ExecuteContext(ctx); err != nil {
		switch {
		case errors.Is(err, cmd.ErrAlreadyReported):
			// The command already printed its own failure summary.
		default:
			if msg := cmd.FormatExecuteError(err); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		os.Exit(1)
	}
}
