# Research release validation

The release candidate implements `research` in vision, `evidence` in knowledge, three additional relation values, and unknown-frontmatter retention. Publication remains pending.

## Implementation evidence

- The registry has 21 document types and seven relation values.
- Registered MCP schemas and lifecycle tests cover create, get, list, search, update, and removal, including rejected documents and category filters.
- Relation tests cover direction, readback, duplicate handling, local-only endpoints, canonical paths, legacy removal, and unchanged invalid manifests.
- Metadata tests compare decoded YAML values and extra-key order against the original fixture. Cases include scalar types, nested collections, multiline text, aliases, merges, tag clearing, and consecutive updates.
- Original `rnd` tests continue to pin its vision category and recommendation-based status behavior.

## Verification

| Check | Result |
|---|---|
| `go test -p 1 ./... -count=1` | Passed after the review corrections |
| `go test -race -p 1 ./... -count=1` | Passed after the review corrections |
| `go test -race ./internal/mcp/tools -count=1` | Passed with the final follow-up test corrections |
| `golangci-lint run ./...` | 0 issues |
| `go vet ./...` | Passed |
| `go build -o /private/tmp/archcore-research-verified .` | Passed |
| Native binary `status` and `doctor` | Accepted seven relation values; manifest bytes unchanged |
| Native binary, real stdio MCP | Created both types, filtered their categories, added `supports`, and returned incoming readback |
| Go review | No open P0–P3 findings after corrections |
| Archcore review | Three declared capability deltas reconciled; stale descriptions corrected |

The first full runs exposed an existing cache-policy test that unnecessarily executed the staged binary under a three-second health-probe timeout. That test now uses the existing `withInstantProbe` seam. Production probe behavior and its dedicated tests are unchanged.

## Test-health-check proof

Contracts come from the three research specs and the accepted category revision. Baselines passed with uncached execution, two repetitions, and shuffle seeds 42 and 137. Expected values are explicit fixtures or independently decoded persisted state.

Each probe changed one behavior in an isolated worktree. Every defect below caused a test failure for the intended reason. Each file was restored byte-for-byte, and its original test then passed. Build-cache and sandbox failures were excluded and rerun with the required permissions.

| Probe | Contract violation detected |
|---|---|
| metadata-reachability | Serializer sentinel reached by the update matrix |
| metadata-old-owned-only | Former owned-fields-only reconstruction loses custom values and order |
| metadata-readback-loss | Metadata loss is visible through registered MCP readback |
| alias-rebinding | Removing anchor-identity validation permits unsafe rewriting |
| merged-tags-resurrection | Removing the explicit empty tag list restores merged tags |
| template-reachability | Dispatcher sentinel reached by template tests |
| template-wrong-dispatch | Research receives the wrong section contract |
| coverage-not-required | Missing Coverage stops producing an advisory |
| relations-reachability | Relation writer sentinel reached through MCP |
| relation-reversed | Persisted direction is reversed |
| relation-value-rejected | `supports` is rejected by manifest validation |
| relation-noncanonical | Equivalent paths bypass self-edge validation or persist different keys |
| global-guard-removed | Existing global documents become linkable |
| legacy-removal-lost | An older stored triple cannot be removed |
| cascade-includes-supports | An evidential edge triggers a content cascade |
| restatement-includes-supports | An evidential edge triggers a restatement finding |
| alignment-includes-research | Research is injected as a code constraint |
| invalid-manifest-erased | A refused operation overwrites invalid manifest bytes |
| list-relations-lost | Stored new values disappear from filtered readback |
| research-wrong-category | Template registry returns Knowledge for research |
| research-wrong-discovery | MCP lifecycle returns the wrong research category |
| evidence-wrong-category | MCP lifecycle returns Vision for evidence |

Diagnosis: `HEALTHY` for the sampled contracts. This is targeted fault evidence, not a mutation score or proof against every possible defect. Exact probe diffs, commands, timings, and logs are retained in the local `research-health-proof.tar.gz` proof bundle.

## Review corrections and limitations

Go review identified endpoint canonicalization and a vacuous global-refusal fixture. Follow-up review required preservation of legacy removability and independent single-tool fixtures. Archcore review found a stale search relation enum and missing invalid-manifest byte assertions. These findings are resolved.

Formatter advisories on unchanged portions of the existing building and search specifications predate this feature. This change corrects their affected vocabulary without claiming a full prose rewrite.

Older binaries reject the new relation values. Plugin routing and published-version handoff remain pending release. The shared source is updated as an accepted vocabulary; that update does not claim deployed runtime support.

Go source snapshot SHA-256: `b711111710a52767eeb63f2b94808ac6b80de4c88d73b40f3e5ae2e624e6e4dd`.

## Follow-up Go review

The follow-up corrected test names, table fixtures, the schema-test property list, and an unchecked MCP response assertion. Parser expectations now use explicit fixture fields instead of subtest names.

The second YAML parse remains necessary. An existing `status: "draft: bad"` is valid YAML, but the manual serializer emits the preserved status without quotes. The final parse refuses that malformed reconstruction before replacement. A new regression reaches this branch and checks unchanged file bytes. The explanatory comment now names this case.

Reconstruction failures now advise manual frontmatter repair without assuming an alias problem. Raw YAML errors remain private because an unknown-anchor error can quote document content. The first privacy fixture used an invalid anchor name and survived a disclosure probe. The corrected fixture uses a valid sensitive marker and detects that disclosure.

An agreement test compares the owned YAML schema, parser exclusions, and serialized key counts. Its fixture comes from the schema and asserts that every owned key is present. This detects a newly added owned field whose parser exclusion was forgotten. Production does not expose a mutable ownership map.

| Additional probe | Contract violation detected |
|---|---|
| reparse-reachability | The preserved-status regression reaches final YAML validation |
| reparse-removed | Skipping validation permits an invalid reconstruction to be written |
| yaml-error-disclosed | Returning the raw reconstruction error exposes the fixture marker |
| owned-status-leaks | Parser retention incorrectly includes an existing owned key |
| new-owned-field-leaks | A new schema field lacks a parser exclusion |
| extra-field-lost | Parser discards the independently expected custom field |

All six distinct probes were detected after the oracle corrections. Every probe was restored, and its original test passed again. The proof bundle includes the follow-up logs and exact diffs. Final independent Go review found no remaining issues in these corrections.

Concurrent full ordinary and race runs hit subprocess deadlines in plugin and updater tests. Both full suites subsequently passed with `-p 1`. These timing failures do not establish a feature regression; they limit claims about reliability under concurrent machine load. No production timeout changed in this follow-up.

The allocation figures supplied in the external review were not independently reproduced. Cache retention was left unchanged; this follow-up makes no performance claim.
