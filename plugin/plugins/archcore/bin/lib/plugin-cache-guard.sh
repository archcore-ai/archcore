#!/bin/sh
# Plugin-cache cwd guard, shared by the hook launchers.
#
# Hosts can misroute hook cwd into a plugin install directory — Cursor stdio
# hooks (forum #99215) land in ~/.cursor/plugins/cache/.../<sha>/, and Copilot
# launches plugin children inside ~/.copilot/installed-plugins/
# (github/copilot-cli#4234) — instead of the user workspace. A hook that runs
# from there would treat the plugin's own files as the user's project.
#
# Two layers, both silent-exit (hosts must see no noise from a misrouted cwd),
# mirroring the inline guard in bin/session-start:
#   1. install-cache path fragments in $PWD — catches any depth inside a
#      host's plugin cache;
#   2. bounded upward walk for plugin manifests — catches installs outside the
#      known cache locations (cwd may sit in a SUBDIRECTORY of the install,
#      e.g. <install>/skills/init/, where no manifest is a direct sibling).
archcore_plugin_cache_guard() {
  case "/$PWD/" in
    */.cursor/plugins/* | */.claude/plugins/* | */.codex/plugins/* | */.copilot/installed-plugins/* | */plugins/cache/*)
      exit 0 ;;
  esac

  _ac_guard_dir="$PWD"
  _ac_guard_depth=0
  while [ -n "$_ac_guard_dir" ] && [ "$_ac_guard_dir" != "/" ] && [ "$_ac_guard_depth" -lt 12 ]; do
    if [ -f "$_ac_guard_dir/.cursor-plugin/plugin.json" ] \
      || [ -f "$_ac_guard_dir/.claude-plugin/plugin.json" ] \
      || [ -f "$_ac_guard_dir/.codex-plugin/plugin.json" ] \
      || [ -f "$_ac_guard_dir/.plugin/plugin.json" ]; then
      exit 0
    fi
    _ac_guard_dir=$(dirname "$_ac_guard_dir")
    _ac_guard_depth=$((_ac_guard_depth + 1))
  done
}
