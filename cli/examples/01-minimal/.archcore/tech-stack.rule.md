---
title: "Fixed Technology Stack"
status: accepted
---

## Rule

This project uses, and does not deviate from:

- Frontend: React + TypeScript + Vite
- Backend: Go (standard `net/http`)
- Database: PostgreSQL via `sqlc`-generated code
- Tests: Go `testing`, Vitest on the frontend

Introducing a new framework or datastore requires an ADR first.

## Rationale

A small team moves faster with one obvious choice per layer.
