import { readdirSync, readFileSync } from "node:fs"
import path from "node:path"

type Meta = Record<string, string | string[]>

export type AgentConfig = {
  description: string
  mode: "subagent"
  steps?: number
  prompt: string
  permission: Record<string, Rule>
}

// Reads the subset of YAML the agent files under plugins/archcore/agents/ use:
// scalars, one folded `>` scalar, and `- item` lists.
export function frontmatter(text: string): { meta: Meta; body: string } {
  const match = /^---\n([\s\S]*?)\n---\n/.exec(text)
  if (!match) return { meta: {}, body: text }
  const meta: Meta = {}
  let key = ""
  for (const line of match[1].split("\n")) {
    const pair = /^(\w+):\s*(.*)$/.exec(line)
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

export function agents(dir: string): Record<string, AgentConfig> {
  const out: Record<string, AgentConfig> = {}
  for (const file of readdirSync(dir).filter((name) => name.endsWith(".md")).sort()) {
    const { meta, body } = frontmatter(readFileSync(path.join(dir, file), "utf8"))
    const name = typeof meta.name === "string" ? meta.name : path.basename(file, ".md")
    const steps = Number(meta.maxTurns)
    out[name] = {
      description: typeof meta.description === "string" ? meta.description : "",
      mode: "subagent",
      ...(Number.isInteger(steps) && steps > 0 ? { steps } : {}),
      prompt: body,
      permission: permission(Array.isArray(meta.tools) ? meta.tools : []),
    }
  }
  return out
}
