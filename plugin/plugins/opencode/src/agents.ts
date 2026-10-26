import { existsSync, readdirSync, readFileSync } from "node:fs"
import path from "node:path"

type Meta = Record<string, string | string[]>

export type AgentConfig = {
  description: string
  mode: "subagent"
  steps?: number
  prompt: string
  permission: Record<string, Rule>
}

// Reads the subset of YAML the agent and skill files under plugins/archcore/
// use: scalars, one folded `>` scalar, and `- item` lists.
export function frontmatter(text: string): { meta: Meta; body: string } {
  const match = /^---\n([\s\S]*?)\n---\n/.exec(text)
  if (!match) return { meta: {}, body: text }
  const meta: Meta = {}
  let key = ""
  for (const line of match[1].split("\n")) {
    const pair = /^([\w-]+):\s*(.*)$/.exec(line)
    if (pair) {
      key = pair[1]
      meta[key] = pair[2] === "" ? [] : pair[2] === ">" ? "" : pair[2]
      continue
    }
    const current = meta[key]
    const item = /^\s+-\s+(.+)$/.exec(line)
    if (item && Array.isArray(current)) current.push(item[1])
    else if (typeof current === "string" && line.trim()) meta[key] = `${current} ${line.trim()}`.trim()
  }
  return { meta, body: text.slice(match[0].length).replace(/^\n+/, "") }
}

type Action = "allow" | "deny" | "ask"
type Rule = Action | Record<string, Action>

// OpenCode's default agent rules ask before reading env files
// (packages/opencode/src/agent/agent.ts, v1.18.34). The read rule below comes
// after them, so it repeats that exception instead of allowing every path.
const READ_RULE: Rule = { "*": "allow", "*.env": "ask", "*.env.*": "ask", "*.env.example": "allow" }

const READ_TOOLS: Record<string, [string, Rule]> = {
  Read: ["read", READ_RULE],
  Grep: ["grep", "allow"],
  Glob: ["glob", "allow"],
}

// OpenCode applies the last matching rule, so the catch-all deny comes first.
export function permission(tools: string[]): Record<string, Rule> {
  const rules: Record<string, Rule> = { "*": "deny" }
  for (const tool of tools) {
    const read = READ_TOOLS[tool]
    if (read) rules[read[0]] = read[1]
    const mcp = /^archcore-(\w+)$/.exec(tool)
    if (mcp) rules[`archcore_${mcp[1]}`] = "allow"
  }
  return rules
}

// scalar reads a frontmatter value as text; a double-quoted value is unquoted.
export function scalar(value: string | string[] | undefined): string | undefined {
  if (typeof value !== "string") return undefined
  if (!/^".*"$/.test(value)) return value
  try {
    return JSON.parse(value) as string
  } catch {
    return value
  }
}

type AgentSource = { name: string; description: string; steps?: number; prompt: string; tools: string[] }

function readAgents(dir: string): AgentSource[] {
  return readdirSync(dir)
    .filter((name) => name.endsWith(".md"))
    .sort()
    .map((file) => {
      const { meta, body } = frontmatter(readFileSync(path.join(dir, file), "utf8"))
      const steps = Number(meta.maxTurns)
      return {
        name: scalar(meta.name) ?? path.basename(file, ".md"),
        description: scalar(meta.description) ?? "",
        ...(Number.isInteger(steps) && steps > 0 ? { steps } : {}),
        prompt: body,
        tools: Array.isArray(meta.tools) ? meta.tools : [],
      }
    })
}

export function agents(dir: string): Record<string, AgentConfig> {
  const out: Record<string, AgentConfig> = {}
  for (const { name, description, steps, prompt, tools } of readAgents(dir)) {
    out[name] = {
      description,
      mode: "subagent",
      ...(steps ? { steps } : {}),
      prompt,
      permission: permission(tools),
    }
  }
  return out
}

export type PermissionV2 = { action: string; resource: string; effect: Action }

export type AgentV2 = {
  name: string
  description: string
  mode: "subagent"
  steps?: number
  system: string
  permissions: PermissionV2[]
}

// The V2 form of permission(): a ruleset of {action, resource, effect}, where
// the last matching rule wins (packages/core/src/tool.ts, OpenCode 2.1.4).
export function permissionsV2(tools: string[]): PermissionV2[] {
  return Object.entries(permission(tools)).flatMap(([action, rule]) =>
    typeof rule === "string"
      ? [{ action, resource: "*", effect: rule }]
      : Object.entries(rule).map(([resource, effect]) => ({ action, resource, effect })),
  )
}

export function agentsV2(dir: string): Record<string, AgentV2> {
  const out: Record<string, AgentV2> = {}
  for (const { name, description, steps, prompt, tools } of readAgents(dir)) {
    out[name] = {
      name,
      description,
      mode: "subagent",
      ...(steps ? { steps } : {}),
      system: prompt,
      permissions: permissionsV2(tools),
    }
  }
  return out
}

export type SkillV2 = { id: string; name: string; description?: string; path: string; content: string }

// bundledSkills reads skills/<dir>/SKILL.md the way OpenCode 2.x parses a skill
// (packages/core/src/config/plugin/skill-file.ts), except that the id is the
// archcore- name, so it never replaces a user skill named plan or review.
export function bundledSkills(dir: string): SkillV2[] {
  return readdirSync(dir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && existsSync(path.join(dir, entry.name, "SKILL.md")))
    .map((entry) => entry.name)
    .sort()
    .map((name) => {
      const file = path.join(dir, name, "SKILL.md")
      const { meta, body } = frontmatter(readFileSync(file, "utf8"))
      const id = scalar(meta.name) ?? name
      const description = scalar(meta.description)
      return { id, name: id, ...(description ? { description } : {}), path: file, content: body }
    })
}
