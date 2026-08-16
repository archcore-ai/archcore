---
title: "Install Provenance Receipt"
status: draft
tags:
  - "cli"
  - "release"
  - "update"
---

## Purpose & Scope

**Deferred whole (decision 2026-08-15).** Nothing in this spec is active in the current release — neither the reads nor the writes. The spec activates with the first package-manager channel: a Homebrew tap, a Scoop manifest, or a winget package. Until then it records the agreed shape, so activation is a decision, not a design session.

This spec defines the **install provenance receipt**: the local file that records which channel put this binary at this path, and the only evidence the unattended update policy accepts as authorization to replace it. Dependents: `@install.sh`, `@install.ps1`, the `update` command (`@cmd/update.go`), and any future package-manager channel, which must not write it.

Out of scope: `install-id`, which answers "which machine" for analytics and carries no authorization meaning. The two files share a directory and nothing else.

## Surface

- Path: `${XDG_STATE_HOME:-$HOME/.local/state}/archcore/install-receipt.json` on Unix, `%USERPROFILE%\.local\state\archcore\install-receipt.json` on Windows — the directory `install_id_path()` in `@install.sh` already creates.
- Fields: `channel` (one of `install.sh`, `install.ps1`, `self-update`), `binary` (absolute, symlink-resolved path of the installed executable), `version` (the version placed at that path).
- Writers: both installers on a successful install; the `update` command on a successful replacement.
- Reader: the unattended update policy in `internal/update` [planned].
- Resolution used for comparison: `os.Executable()` followed by `filepath.EvalSymlinks()`, the pair `Apply` already uses (`@internal/update/update.go:308`).

## Normative Behavior

The requirements below bind implementations only after activation.

1. WHEN an installer completes an install, the installer MUST write the receipt with `channel` set to its own script name, `binary` set to the installed path, and `version` set to the installed version.
2. WHEN `archcore update` replaces the binary successfully, the CLI MUST write the receipt with `channel` set to `self-update` if no receipt exists, and MUST otherwise update only `version` and `binary`.
3. The CLI MUST NOT overwrite an existing `channel` value.
4. WHEN the unattended update policy reads the receipt, the CLI MUST compare `binary` against the resolved path of the running executable.
5. IF `binary` and the resolved path differ, THEN the CLI MUST treat the binary as unauthorized for unattended replacement.
6. The receipt MUST stay on the machine that wrote it, and the CLI MUST NOT transmit any field of it.

## Constraints & Invariants

- Constraint: a package manager MUST NOT write this receipt. Rationale: its presence is the statement "a channel that owns this path installed it", and Homebrew, Scoop, and winget own their own trees.
- Constraint: writing the receipt MUST NOT depend on a telemetry opt-out. Rationale: the receipt records provenance for the update path, not identity for analytics; conflating them would make an opt-out silently disable updates.
- Constraint: the receipt MUST NOT carry a user name, a host name, or a project path. The executable path is the one path it carries, and requirement 6 keeps it local.
- Constraint: activation must answer the no-receipt fleet: machines that only ever updated unattended carry no receipt at activation time, and the fail-closed rule below would switch them off. The activation plan owns the migration; this spec does not prescribe it.
- Invariant: `channel` records who installed the binary first, never who last replaced it. A machine installed by `install.sh` and later updated keeps `channel: install.sh`.
- Invariant: a machine with no receipt is authorized for manual updates and refused for unattended ones.

## Failure Behavior

1. IF the receipt is absent, THEN the CLI MUST treat the binary as unauthorized for unattended replacement.
2. IF the receipt is unreadable or malformed, THEN the CLI MUST treat it as absent. This read is **fail-closed**: an unreadable receipt denies, it does not allow.
3. IF the receipt cannot be written after a successful replacement, THEN the CLI MUST NOT fail the update. The next successful update retries the write.
4. IF the user relocates the binary after installation, THEN requirement 5 refuses unattended replacement until a manual `archcore update` rewrites `binary`.

## Conformance

An implementation is conformant when, after activation, the installers and the `update` command write the fields in requirements 1–3, unattended replacement consults the receipt per requirements 4–5, and every read failure denies per the fail-closed rule.

Given a binary installed by Homebrew and never touched by `install.sh`, when the unattended update policy runs, then no receipt names that path, and the policy refuses without contacting the network.