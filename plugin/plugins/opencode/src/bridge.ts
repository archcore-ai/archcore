import path from "node:path"
import { agents } from "./agents"
import { MISSING, run } from "./spawn"

// Bridge between OpenCode's plugin hooks and the shared launchers in bin/
// (opencode-adapter.spec). No guard or validation decision is made here.

// apply_patch is the OpenCode 1.x name of the patch tool; 2.x calls it patch.
export const PRE_MUTATION = new Set(["write", "edit", "apply_patch", "patch"])
export const POST_MUTATION = new Set([
  "archcore_create_document",
  "archcore_update_document",
  "archcore_remove_document",
  "archcore_add_relation",
  "archcore_remove_relation",
])

export const BUDGET_MS = { gate: 2000, pre: 2000, post: 4000, session: 10000 }

// The first CLI release with the plain-text OpenCode session envelope and the
// archcore_* fold (plugin-cli-compatibility.rule, requirement 9).
export const MIN_CLI = "0.10.0"

export type Gate = "yes" | "no" | "__NO_CLI__"

type Input = { directory: string }
type ToolInput = { tool: string; sessionID: string; callID: string; args?: unknown }

export function warn(message: string) {
  process.stderr.write(`[archcore] ${message}\n`)
}

export function createServer(root: string) {
  const bin = (name: string) => path.join(root, "bin", name)

  return async (input: Input) => {
    const directory = input.directory
    const launch = (name: string, payload: unknown, budget: number) =>
      run(bin(name), [], directory, JSON.stringify(payload), budget)

    const gate = await checkGate(bin("cli-gte"), directory)
    const tools = gate === "yes"
    // Without a CLI, bin/session-start still prints the install instructions.
    const session = gate !== "no"

    const held = new Map<string, string>()
    const recaps = new Map<string, Promise<string>>()

    return {
      config: async (cfg: any) => {
        try {
          cfg.mcp ??= {}
          if (!cfg.mcp.archcore) cfg.mcp.archcore = { type: "local", command: ["archcore", "mcp"] }
          cfg.skills ??= {}
          cfg.skills.paths = [...(cfg.skills.paths ?? []), path.join(root, "skills")]
          cfg.agent ??= {}
          for (const [name, agent] of Object.entries(agents(path.join(root, "agents")))) {
            if (!cfg.agent[name]) cfg.agent[name] = agent
          }
        } catch (error) {
          warn(`config: ${error instanceof Error ? error.message : String(error)}`)
        }
      },

      "tool.execute.before": async (call: ToolInput, output: { args: unknown }) => {
        if (!tools || !PRE_MUTATION.has(call.tool)) return
        const result = await launch("pre-tool-use", { tool_name: call.tool, tool_input: output.args }, BUDGET_MS.pre)
        if (result.code === 2) throw new Error(result.stderr.replace(/\r?\n$/, ""))
        if (result.code === 0 && result.stdout) held.set(call.callID, result.stdout)
      },

      "tool.execute.after": async (call: ToolInput, output: any) => {
        try {
          const context = held.get(call.callID)
          held.delete(call.callID)
          if (context) output.output = output.output ? `${output.output}\n\n${context}` : context
          if (!tools || !POST_MUTATION.has(call.tool)) return
          const result = await launch("post-tool-use", { tool_name: call.tool, tool_input: call.args }, BUDGET_MS.post)
          if (result.code === 0 && result.stdout && Array.isArray(output.content)) {
            output.content.push({ type: "text", text: result.stdout })
          }
        } catch {}
      },

      "experimental.chat.system.transform": async (request: { sessionID?: string }, output: { system: string[] }) => {
        try {
          if (!session || !request.sessionID) return
          let recap = recaps.get(request.sessionID)
          if (!recap) {
            recap = launch("session-start", { cwd: directory }, BUDGET_MS.session).then((result) =>
              result.code === 0 ? result.stdout : "",
            )
            recaps.set(request.sessionID, recap)
          }
          const text = await recap
          if (text) output.system.push(text)
        } catch {}
      },
    }
  }
}

export async function checkGate(file: string, directory: string): Promise<Gate> {
  const result = await run(file, [MIN_CLI], directory, "", BUDGET_MS.gate)
  const token = result.stdout.trim()
  if (result.code === 0 && (token === "yes" || token === "__NO_CLI__")) return token
  if (result.code === 0 && token === "no") warn(`archcore CLI older than ${MIN_CLI}; the OpenCode hooks are off`)
  else if (result !== MISSING) warn("the CLI version check did not run; the OpenCode hooks are off")
  return "no"
}
