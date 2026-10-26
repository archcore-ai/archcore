import { expect, test } from "bun:test"
import { readFileSync } from "node:fs"
import path from "node:path"
import { agents } from "./agents"

const SOURCE = path.resolve(import.meta.dir, "../../archcore/agents")
const defined = agents(SOURCE)

const READ = ["archcore_get_document", "archcore_list_documents", "archcore_list_relations", "archcore_search_documents"]
const WRITE = [
  "archcore_add_relation",
  "archcore_create_document",
  "archcore_remove_document",
  "archcore_remove_relation",
  "archcore_update_document",
]

function allowed(name: string): string[] {
  return Object.entries(defined[name].permission)
    .filter(([, rule]) => rule === "allow" || (typeof rule === "object" && rule["*"] === "allow"))
    .map(([tool]) => tool)
    .sort()
}

test("both source agents become OpenCode subagents", () => {
  expect(Object.keys(defined).sort()).toEqual(["archcore-assistant", "archcore-auditor"])
  for (const agent of Object.values(defined)) expect(agent.mode).toBe("subagent")
})

test("steps come from maxTurns", () => {
  expect(defined["archcore-assistant"].steps).toBe(20)
  expect(defined["archcore-auditor"].steps).toBe(40)
})

test("the description is the folded source description", () => {
  expect(defined["archcore-auditor"].description).toBe(
    "Read-only documentation auditor. Use proactively for reviewing documentation health: missing or unsupported relations, stale statuses, coverage gaps, and consistency checks across the .archcore/ knowledge base.",
  )
})

test("the prompt is the source body below the frontmatter", () => {
  const source = readFileSync(path.join(SOURCE, "archcore-auditor.md"), "utf8")
  expect(source.endsWith(defined["archcore-auditor"].prompt)).toBe(true)
  expect(defined["archcore-auditor"].prompt.startsWith("You are the Archcore documentation auditor")).toBe(true)
})

test("every other tool is denied first, so the allows after it win", () => {
  for (const agent of Object.values(defined)) {
    expect(Object.entries(agent.permission)[0]).toEqual(["*", "deny"])
  }
})

test("reading env files still asks, as OpenCode's default rules do", () => {
  for (const agent of Object.values(defined)) {
    expect(agent.permission.read).toEqual({ "*": "allow", "*.env": "ask", "*.env.*": "ask", "*.env.example": "allow" })
  }
})

test("the auditor gets only the read tools", () => {
  expect(allowed("archcore-auditor")).toEqual([...READ, "glob", "grep", "read"].sort())
})

test("the assistant gets the read and write document tools, and no edit or bash", () => {
  expect(allowed("archcore-assistant")).toEqual([...READ, ...WRITE, "glob", "grep", "read"].sort())
})
