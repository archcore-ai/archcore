// Package projectroot resolves the archcore project base directory in a
// host-agnostic way and applies sanity guards.
//
// Priority chain (highest first):
//
//  1. Options.Flag (the --base-dir flag value)
//  2. ARCHCORE_BASE_DIR environment variable
//  3. Walk-up from Options.Getwd looking for .archcore/, .git/, or a
//     generic project marker. Depth ≤ 8, stops at $HOME or fs root.
//  4. Getwd itself — only in ModeInit, where the caller is about to
//     create .archcore/ and lack of markers is expected.
//
// Guards (after resolution): reject $HOME, /, /tmp, /var/tmp, OS temp,
// paths without project markers (except ModeInit), and non-existent
// paths. Setting ARCHCORE_LEGACY_BASE_DIR=1 disables HOME/system/marker
// guards (existence is still checked) — this is a migration escape hatch
// only and the caller is expected to print a warning.
//
// Stable error codes (consumers parse these — never change):
//
//	ERR_NOT_PROJECT     base dir lacks project markers
//	ERR_HOME_REFUSED    base dir equals $HOME
//	ERR_NO_PROJECT      walk-up exhausted, nothing resolved
//	ERR_PATH_INVALID    path missing, not a directory, or a system path
//	ERR_TILDE_EXPAND    "~/..." expansion failed
//
// This package contains zero host-specific knowledge: no CLAUDE_*, CODEX_*,
// WORKSPACE_*, CURSOR_*, CLAUDE_PLUGIN_* environment variables, no plugin
// paths. Translating host signals into ARCHCORE_BASE_DIR is the plugin's
// job, not the CLI's.
package projectroot
