import path from "node:path"
import { agentsV2, bundledSkills } from "./agents"
import { BUDGET_MS, POST_MUTATION, PRE_MUTATION, checkGate, warn } from "./bridge"
import { run } from "./spawn"

// The OpenCode 2.x half of the bridge (opencode-plugin-serves-v1-and-v2-from-one-package.adr).
// It maps each V1 hook to its V2 destination and spawns the same launchers.
// The types below are the subset of @opencode/plugin 2.1.4 this file touches.

type Text = { type: "text"; text: string }
type ToolEvent = {
  tool: string
  sessionID: string
  id: string
  input: unknown
  status?: "completed" | "error"
  result?: { output?: unknown; content?: Text[]; metadata?: unknown }
}
type ContextEvent = { sessionID: string; system: Text[] }
type Prompt = { text: string; skills?: { id: string }[] }
type CommandInput = { sessionID: string; prompt: Prompt; delivery: unknown }
type Editor<T> = { get(id: string): T | undefined; set?(id: string, value: T): void; add?(value: T): void }

export type Context = {
  location: { directory: string }
  mcp: { transform(cb: (editor: Required<Pick<Editor<unknown>, "get" | "set">>) => void): Promise<unknown> }
  skill: { transform(cb: (editor: Required<Pick<Editor<unknown>, "get" | "add">>) => void): Promise<unknown> }
  command: { transform(cb: (editor: { add(command: unknown): void }) => void): Promise<unknown> }
  agent: { transform(cb: (editor: { update(id: string, fn: (agent: Record<string, unknown>) => void): void }) => void): Promise<unknown> }
  tool: { hook(name: "execute.before" | "execute.after", cb: (event: ToolEvent) => Promise<void>): Promise<unknown> }
  session: {
    hook(name: "context", cb: (event: ContextEvent) => Promise<void>): Promise<unknown>
    prompt(input: unknown): Promise<unknown>
  }
}

// Reads a bundled tree once, before any transform: V2 replays transforms, so
// they stay synchronous and cheap. A missing tree registers nothing.
function load<T>(what: string, read: () => T, empty: T): T {
  try {
    return read()
  } catch (error) {
    warn(`${what}: ${error instanceof Error ? error.message : String(error)}`)
    return empty
  }
}

export function createSetup(root: string) {
  const bin = (name: string) => path.join(root, "bin", name)

  return async (ctx: Context) => {
    const directory = ctx.location.directory
    const launch = (name: string, payload: unknown, budget: number) =>
      run(bin(name), [], directory, JSON.stringify(payload), budget)

    const gate = await checkGate(bin("cli-gte"), directory)
    const tools = gate === "yes"
    const session = gate !== "no"

    const skills = load("skills", () => bundledSkills(path.join(root, "skills")), [])
    const agents = load("agents", () => agentsV2(path.join(root, "agents")), {})

    // The config MCP plugin runs before package plugins, so a user's own
    // archcore entry is already here and stays (opencode-mcp-entry-ownership.adr).
    await ctx.mcp.transform((editor) => {
      if (!editor.get("archcore")) editor.set("archcore", { type: "local", command: ["archcore", "mcp"] })
    })
    await ctx.skill.transform((editor) => {
      for (const skill of skills) if (!editor.get(skill.id)) editor.add(skill)
    })
    // 2.x offers skills as @mentions only, so each one also gets its
    // /archcore-* command, which attaches the skill to the prompt.
    await ctx.command.transform((editor) => {
      for (const skill of skills) {
        editor.add({
          name: skill.name,
          description: skill.description,
          execute: (input: CommandInput) =>
            ctx.session.prompt({
              ...input.prompt,
              sessionID: input.sessionID,
              text: input.prompt.text.trim() || `/${skill.name}`,
              skills: [...(input.prompt.skills ?? []), { id: skill.id }],
              delivery: input.delivery,
            }),
        })
      }
    })
    // Config agents apply after package plugins and merge onto the same id,
    // so a user's own definition of these agents still wins.
    await ctx.agent.transform((editor) => {
      for (const [id, agent] of Object.entries(agents)) editor.update(id, (current) => Object.assign(current, agent))
    })

    const held = new Map<string, string>()
    const advice = new Map<string, string[]>()
    const recaps = new Map<string, Promise<string>>()

    await ctx.tool.hook("execute.before", async (event) => {
      if (!tools || !PRE_MUTATION.has(event.tool)) return
      const result = await launch("pre-tool-use", { tool_name: event.tool, tool_input: event.input }, BUDGET_MS.pre)
      if (result.code === 2) throw new Error(result.stderr.replace(/\r?\n$/, ""))
      if (result.code === 0 && result.stdout) held.set(event.id, result.stdout)
    })

    // A throw here would fail the session step, not the tool call, so nothing escapes.
    await ctx.tool.hook("execute.after", async (event) => {
      try {
        const context = held.get(event.id)
        held.delete(event.id)
        if (event.status !== "completed" || !event.result) return
        if (context && Array.isArray(event.result.content)) {
          event.result = { ...event.result, content: [...event.result.content, { type: "text", text: context }] }
        }
        if (!tools || !POST_MUTATION.has(event.tool)) return
        const result = await launch("post-tool-use", { tool_name: event.tool, tool_input: event.input }, BUDGET_MS.post)
        // MCP tools run inside Code Mode by default, where text added to their
        // result never reaches the model; the next model request carries it.
        if (result.code === 0 && result.stdout) {
          advice.set(event.sessionID, [...(advice.get(event.sessionID) ?? []), result.stdout])
        }
      } catch {}
    })

    await ctx.session.hook("context", async (event) => {
      try {
        if (!session) return
        let recap = recaps.get(event.sessionID)
        if (!recap) {
          recap = launch("session-start", { cwd: directory }, BUDGET_MS.session).then((result) =>
            result.code === 0 ? result.stdout : "",
          )
          recaps.set(event.sessionID, recap)
        }
        const text = await recap
        if (text) event.system.push({ type: "text", text })
        for (const pending of advice.get(event.sessionID) ?? []) event.system.push({ type: "text", text: pending })
        advice.delete(event.sessionID)
      } catch {}
    })
  }
}
