package main

import (
	"context"
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
		if msg := cmd.FormatExecuteError(err); msg != "" {
			fmt.Println(msg)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
