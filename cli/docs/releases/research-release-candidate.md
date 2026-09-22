# Archcore CLI v0.8.3 — release candidate

This candidate adds stored research, reusable evidence, and three relation values. Publication is pending.

## Document types

- `research` belongs to `vision`. Its template records Goal, Scope, Coverage, Sources, Findings, Synthesis, and Open Gaps.
- `evidence` belongs to `knowledge`. Its template records Locator, Extract, and Notes.
- The registry contains 21 types: 12 vision, 7 knowledge, and 2 experience.
- `rnd` retains its recommendation-based workflow and vision category.

Both new types use the ISO prose profile. Missing required sections produce advisories; they do not block document writes.

## Relations

- `supports` points from material to the statement it backs.
- `contradicts` points from the challenger to the statement it disputes.
- `supersedes` points from the newer document to the document it replaces.

The seven relation values share the existing manifest format. Relations connect distinct local documents, without restrictions on their types or categories. Adding a relation does not change document status, create an inverse edge, or resolve a contradiction.

Relation writes normalize endpoint paths before comparing or storing them. Removal also accepts the exact unnormalized triples stored by older binaries.

## Frontmatter retention

`update_document` preserves unknown top-level YAML values and their relative order after the owned fields. The owned fields remain `title`, `status`, and `tags`. Custom metadata remains in raw document content; it does not become structured sync configuration.

Retention preserves YAML meaning, not original formatting or comments. If reconstructing an owned field would break a retained alias, the update fails before writing and requests manual repair. Clearing tags also overrides tags inherited through a retained YAML merge.

## Compatibility and handoff

Older CLI binaries reject manifests containing the three new relation values. This release provides no downgrade conversion. Update every CLI that reads a shared manifest before adding the new values.

The engine stores sources and findings. It does not fetch sources, verify authenticity, or execute a research method. Plugin routing and its minimum-engine gate require a separate plugin release after the CLI assets are published.

The selected release tag is `v0.8.3`. The release workflow in @.github/workflows/release.yml runs tests and publishes six platform archives plus checksums. The plugin handoff must name the actual published tag after verification.
