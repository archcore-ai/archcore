import { afterEach, beforeEach, describe, expect, test } from "bun:test"
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, realpathSync, rmSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import path from "node:path"
import { BUDGET_MS, createServer } from "./bridge"
import { createSetup, type Context } from "./v2"

const FIXTURES = path.resolve(import.meta.dir, "../../../test/fixtures/stdin/opencode")
const LAUNCHERS = ["cli-gte", "pre-tool-use", "post-tool-use", "session-start"]

// Each stub records what it received, then sources <name>.sh from the stub dir
// when a test wrote one, so a test scripts the exit code and the output.
const STUB = `#!/bin/sh
name=$(basename "$0")
cat > "$STUB_DIR/$name.stdin"
pwd -P > "$STUB_DIR/$name.cwd"
printf '%s' "$ARCHCORE_HOST" > "$STUB_DIR/$name.host"
printf '%s' "$*" > "$STUB_DIR/$name.args"
echo x >> "$STUB_DIR/$name.count"
[ -f "$STUB_DIR/$name.sh" ] && . "$STUB_DIR/$name.sh"
[ "$name" = cli-gte ] && echo yes
exit 0
`

let root: string
let stubs: string
let project: string

// Behavior tests do not measure time, and the first run of a freshly written
// stub can take seconds on a loaded machine, so they get budgets far above that.
const BUDGETS = { ...BUDGET_MS }

function behave(name: string, script: string) {
  writeFileSync(path.join(stubs, `${name}.sh`), script)
}

function count(name: string): number {
  const file = path.join(stubs, `${name}.count`)
  return existsSync(file) ? readFileSync(file, "utf8").trim().split("\n").length : 0
}

function received(name: string): unknown {
  return JSON.parse(readFileSync(path.join(stubs, `${name}.stdin`), "utf8"))
}

function fixture(name: string): { tool_name: string; tool_input: unknown } {
  return JSON.parse(readFileSync(path.join(FIXTURES, name), "utf8"))
}

// stderrOf collects what the bridge writes to stderr while fn runs.
async function stderrOf(fn: () => Promise<unknown>): Promise<string[]> {
  const lines: string[] = []
  const write = process.stderr.write
  process.stderr.write = ((chunk: string | Uint8Array) => lines.push(String(chunk)) > 0) as typeof write
  try {
    await fn()
  } finally {
    process.stderr.write = write
  }
  return lines
}

function alive(pid: number): boolean {
  try {
    process.kill(pid, 0)
    return true
  } catch {
    return false
  }
}

async function hooks() {
  return (await createServer(root)({ directory: project })) as any
}

beforeEach(() => {
  root = mkdtempSync(path.join(tmpdir(), "archcore-oc-"))
  stubs = path.join(root, "stubs")
  project = realpathSync(mkdtempSync(path.join(tmpdir(), "archcore-oc-project-")))
  for (const dir of ["bin", "skills", "agents", "stubs"]) mkdirSync(path.join(root, dir))
  for (const name of LAUNCHERS) {
    const file = path.join(root, "bin", name)
    writeFileSync(file, STUB)
    chmodSync(file, 0o755)
  }
  writeFileSync(
    path.join(root, "agents", "archcore-auditor.md"),
    "---\nname: archcore-auditor\ndescription: >\n  Read-only\n  auditor.\nmaxTurns: 40\ntools:\n  - archcore-get_document\n  - Read\n---\n\nYou audit.\n",
  )
  process.env.STUB_DIR = stubs
  Object.assign(BUDGET_MS, { gate: 15000, pre: 15000, post: 15000, session: 15000 })
})

afterEach(() => {
  Object.assign(BUDGET_MS, BUDGETS)
  rmSync(root, { recursive: true, force: true })
  rmSync(project, { recursive: true, force: true })
})

describe("config", () => {
  test("registers the MCP server, the skills path, and the agents on an empty config", async () => {
    const cfg: any = {}
    await (await hooks()).config(cfg)
    expect(cfg.mcp.archcore).toEqual({ type: "local", command: ["archcore", "mcp"] })
    expect(cfg.skills.paths).toEqual([path.join(root, "skills")])
    expect(cfg.agent["archcore-auditor"]).toEqual({
      description: "Read-only auditor.",
      mode: "subagent",
      steps: 40,
      prompt: "You audit.\n",
      permission: {
        "*": "deny",
        archcore_get_document: "allow",
        read: { "*": "allow", "*.env": "ask", "*.env.*": "ask", "*.env.example": "allow" },
      },
    })
  })

  test("keeps an existing mcp.archcore entry, existing skill paths, and an existing agent", async () => {
    const mine = { type: "local", command: ["archcore", "mcp"], environment: { A: "1" } }
    const cfg: any = { mcp: { archcore: mine }, skills: { paths: ["mine"] }, agent: { "archcore-auditor": { prompt: "x" } } }
    await (await hooks()).config(cfg)
    expect(cfg.mcp.archcore).toBe(mine)
    expect(cfg.skills.paths).toEqual(["mine", path.join(root, "skills")])
    expect(cfg.agent["archcore-auditor"]).toEqual({ prompt: "x" })
  })

  test("does not throw when the agents directory is missing", async () => {
    rmSync(path.join(root, "agents"), { recursive: true })
    const cfg: any = {}
    await (await hooks()).config(cfg)
    expect(cfg.mcp.archcore).toBeDefined()
  })
})

describe("CLI gate", () => {
  test("asks cli-gte for 0.10.0 once, from the project directory", async () => {
    const h = await hooks()
    await h["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: {} })
    expect(count("cli-gte")).toBe(1)
    expect(readFileSync(path.join(stubs, "cli-gte.args"), "utf8")).toBe("0.10.0")
  })

  test("spawns no launcher when the CLI is older than 0.10.0", async () => {
    behave("cli-gte", "echo no; exit 0\n")
    const h = await hooks()
    await h["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: {} })
    await h["tool.execute.after"]({ tool: "archcore_update_document", sessionID: "s", callID: "c" }, { content: [] })
    await h["experimental.chat.system.transform"]({ sessionID: "s" }, { system: [] })
    expect(count("pre-tool-use") + count("post-tool-use") + count("session-start")).toBe(0)
  })

  test("without a CLI runs session-start only, for the install instructions", async () => {
    behave("cli-gte", "echo __NO_CLI__; exit 0\n")
    const h = await hooks()
    await h["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: {} })
    await h["experimental.chat.system.transform"]({ sessionID: "s" }, { system: [] })
    expect(count("pre-tool-use")).toBe(0)
    expect(count("session-start")).toBe(1)
  })

  test("treats a missing cli-gte as an old CLI and says so in one stderr line", async () => {
    rmSync(path.join(root, "bin", "cli-gte"))
    const lines = await stderrOf(() => hooks())
    expect(lines).toEqual([`[archcore] launcher missing: ${path.join(root, "bin", "cli-gte")}\n`])
    const h = await hooks()
    await h["experimental.chat.system.transform"]({ sessionID: "s" }, { system: [] })
    expect(count("session-start")).toBe(0)
  })
})

describe("tool.execute.before", () => {
  for (const name of ["write-archcore.json", "write-regular.json", "edit-archcore.json", "apply-patch-archcore.json"]) {
    test(`hands ${name} to pre-tool-use verbatim, from the project, as host opencode`, async () => {
      const f = fixture(name)
      await (await hooks())["tool.execute.before"]({ tool: f.tool_name, sessionID: "s", callID: "c" }, { args: f.tool_input })
      expect(received("pre-tool-use")).toEqual(f)
      expect(readFileSync(path.join(stubs, "pre-tool-use.cwd"), "utf8").trim()).toBe(project)
      expect(readFileSync(path.join(stubs, "pre-tool-use.host"), "utf8")).toBe("opencode")
    })
  }

  test("ignores a tool outside the pre-mutation set", async () => {
    await (await hooks())["tool.execute.before"]({ tool: "bash", sessionID: "s", callID: "c" }, { args: {} })
    expect(count("pre-tool-use")).toBe(0)
  })

  test("throws the stderr reason without its trailing newline on exit 2", async () => {
    behave("pre-tool-use", "printf 'Use the MCP tools.\\nLine two.\\n' >&2; exit 2\n")
    const f = fixture("write-archcore.json")
    const h = await hooks()
    await expect(h["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: f.tool_input })).rejects.toThrow(
      /^Use the MCP tools\.\nLine two\.$/,
    )
  })

  test("allows the call on an exit code other than 0 or 2", async () => {
    behave("pre-tool-use", "echo noise; exit 1\n")
    const h = await hooks()
    await h["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: {} })
    const out = { output: "done" }
    await h["tool.execute.after"]({ tool: "write", sessionID: "s", callID: "c" }, out)
    expect(out.output).toBe("done")
  })

  // The sleep is the launcher's own child: only a kill of the whole process
  // group reaches it, which is what Failure Behavior 4 asks of the bridge.
  test("allows the call and kills the launcher's process group when it exceeds its budget", async () => {
    const saved = BUDGET_MS.pre
    // The budget leaves a loaded machine time to start the stub and record the
    // child's pid; the sleep still outlasts it by seconds.
    BUDGET_MS.pre = 1500
    try {
      behave("pre-tool-use", 'sleep 10 & echo $! > "$STUB_DIR/child.pid"; wait; exit 2\n')
      const started = Date.now()
      await (await hooks())["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: {} })
      expect(Date.now() - started).toBeLessThan(5000)
      const pid = Number(readFileSync(path.join(stubs, "child.pid"), "utf8"))
      const deadline = Date.now() + 1000
      while (alive(pid) && Date.now() < deadline) await Bun.sleep(20)
      expect(alive(pid)).toBe(false)
    } finally {
      BUDGET_MS.pre = saved
    }
  })

  test("allows the call when the launcher file is missing", async () => {
    rmSync(path.join(root, "bin", "pre-tool-use"))
    await (await hooks())["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: {} })
  })
})

describe("tool.execute.after", () => {
  test("appends the context pre-tool-use printed for the same callID", async () => {
    behave("pre-tool-use", "printf 'Read the spec first.\\n'; exit 0\n")
    const h = await hooks()
    await h["tool.execute.before"]({ tool: "edit", sessionID: "s", callID: "c1" }, { args: {} })
    const other = { output: "other" }
    await h["tool.execute.after"]({ tool: "edit", sessionID: "s", callID: "c2" }, other)
    const out = { output: "Edit applied." }
    await h["tool.execute.after"]({ tool: "edit", sessionID: "s", callID: "c1" }, out)
    expect(other.output).toBe("other")
    expect(out.output).toBe("Edit applied.\n\nRead the spec first.\n")
  })

  test("hands the MCP call to post-tool-use and pushes its stdout as a content item", async () => {
    behave("post-tool-use", "printf '[Archcore Validation] ok\\n'; exit 0\n")
    const f = fixture("mcp-update.json")
    const out = { content: [{ type: "text", text: "updated" }] }
    await (await hooks())["tool.execute.after"]({ tool: f.tool_name, sessionID: "s", callID: "c", args: f.tool_input }, out)
    expect(received("post-tool-use")).toEqual(f)
    expect(out.content).toEqual([
      { type: "text", text: "updated" },
      { type: "text", text: "[Archcore Validation] ok\n" },
    ])
  })

  test("leaves the output unchanged when content is not an array", async () => {
    behave("post-tool-use", "printf 'note\\n'; exit 0\n")
    const out = { output: "x" }
    await (await hooks())["tool.execute.after"]({ tool: "archcore_update_document", sessionID: "s", callID: "c", args: {} }, out)
    expect(out).toEqual({ output: "x" })
  })

  test("ignores a tool outside the post-mutation set", async () => {
    const out = { output: "x" }
    await (await hooks())["tool.execute.after"]({ tool: "archcore_get_document", sessionID: "s", callID: "c", args: {} }, out)
    expect(count("post-tool-use")).toBe(0)
    expect(out).toEqual({ output: "x" })
  })

  test("does not throw on a malformed output", async () => {
    await (await hooks())["tool.execute.after"]({ tool: "archcore_update_document", sessionID: "s", callID: "c", args: {} }, null)
  })
})

describe("experimental.chat.system.transform", () => {
  // 600 KB of a three-byte character crosses many pipe reads, so a split
  // character shows up as U+FFFD unless the stream is decoded as a whole.
  test("appends a recap that spans many pipe reads byte for byte", async () => {
    behave("session-start", "awk 'BEGIN { for (i = 0; i < 200000; i++) printf \"\\342\\200\\224\" }'; exit 0\n")
    const out = { system: [] as string[] }
    await (await hooks())["experimental.chat.system.transform"]({ sessionID: "s" }, out)
    expect(out.system).toEqual(["\u2014".repeat(200000)])
  })

  test("spawns session-start once per session and appends the recap on every request", async () => {
    behave("session-start", "sleep 0.2; printf '[Archcore] recap\\n'; exit 0\n")
    const h = await hooks()
    const a = { system: ["base"] }
    const b = { system: ["base"] }
    await Promise.all([
      h["experimental.chat.system.transform"]({ sessionID: "s1" }, a),
      h["experimental.chat.system.transform"]({ sessionID: "s1" }, b),
    ])
    const c = { system: [] as string[] }
    await h["experimental.chat.system.transform"]({ sessionID: "s1" }, c)
    expect(count("session-start")).toBe(1)
    expect(a.system).toEqual(["base", "[Archcore] recap\n"])
    expect(b.system).toEqual(["base", "[Archcore] recap\n"])
    expect(c.system).toEqual(["[Archcore] recap\n"])
    expect(received("session-start")).toEqual({ cwd: project })
  })

  test("spawns again for a new session", async () => {
    const h = await hooks()
    await h["experimental.chat.system.transform"]({ sessionID: "s1" }, { system: [] })
    await h["experimental.chat.system.transform"]({ sessionID: "s2" }, { system: [] })
    expect(count("session-start")).toBe(2)
  })

  test("leaves the system prompt unchanged without a sessionID", async () => {
    const out = { system: ["base"] }
    await (await hooks())["experimental.chat.system.transform"]({}, out)
    expect(out.system).toEqual(["base"])
    expect(count("session-start")).toBe(0)
  })

  test("adds nothing when session-start prints nothing or fails", async () => {
    const h = await hooks()
    const empty = { system: [] as string[] }
    await h["experimental.chat.system.transform"]({ sessionID: "s1" }, empty)
    behave("session-start", "printf 'partial'; exit 1\n")
    const failed = { system: [] as string[] }
    await h["experimental.chat.system.transform"]({ sessionID: "s2" }, failed)
    expect(empty.system).toEqual([])
    expect(failed.system).toEqual([])
  })
})

test("the session-start fixture carries only cwd", () => {
  expect(Object.keys(fixture("session-start.json"))).toEqual(["cwd"])
})

test("every OpenCode fixture is valid JSON in the bridge envelope", () => {
  for (const name of readdirSync(FIXTURES)) {
    const body = JSON.parse(readFileSync(path.join(FIXTURES, name), "utf8"))
    if (name === "session-start.json") continue
    expect(Object.keys(body).sort()).toEqual(["tool_input", "tool_name"])
  }
})

// OpenCode 2.x calls setup(ctx) and registers everything through ctx
// (opencode-plugin-serves-v1-and-v2-from-one-package.adr). The fake records each
// transform and hook so a test can run them the way the host would.
function fakeHost() {
  const transforms: Record<string, (editor: any) => void> = {}
  const hooks: Record<string, (event: any) => Promise<void>> = {}
  const prompts: any[] = []
  const ctx = {
    location: { directory: project },
    mcp: { transform: async (cb: any) => void (transforms.mcp = cb) },
    skill: { transform: async (cb: any) => void (transforms.skill = cb) },
    command: { transform: async (cb: any) => void (transforms.command = cb) },
    agent: { transform: async (cb: any) => void (transforms.agent = cb) },
    tool: { hook: async (name: string, cb: any) => void (hooks[name] = cb) },
    session: {
      hook: async (name: string, cb: any) => void (hooks[name] = cb),
      prompt: async (input: any) => void prompts.push(input),
    },
  }
  return { ctx: ctx as unknown as Context, transforms, hooks, prompts }
}

async function setupV2() {
  const host = fakeHost()
  await createSetup(root)(host.ctx)
  return host
}

function mapEditor(initial: Record<string, unknown> = {}) {
  const items = new Map(Object.entries(initial))
  return {
    items,
    get: (id: string) => items.get(id),
    set: (id: string, value: unknown) => void items.set(id, value),
    add: (value: any) => void items.set(value.id ?? value.name, value),
  }
}

function writeSkill(dir: string, body: string) {
  mkdirSync(path.join(root, "skills", dir), { recursive: true })
  writeFileSync(path.join(root, "skills", dir, "SKILL.md"), body)
}

describe("OpenCode 2.x setup", () => {
  test("registers the MCP server when absent and keeps an existing one", async () => {
    const { transforms } = await setupV2()
    const empty = mapEditor()
    transforms.mcp(empty)
    expect(empty.items.get("archcore")).toEqual({ type: "local", command: ["archcore", "mcp"] })
    const mine = { type: "local", command: ["my-archcore"] }
    const existing = mapEditor({ archcore: mine })
    transforms.mcp(existing)
    expect(existing.items.get("archcore")).toBe(mine)
  })

  test("adds each bundled skill under its archcore- name and keeps a skill of that id", async () => {
    writeSkill("plan", '---\nname: archcore-plan\nargument-hint: "[topic]"\ndescription: "Plan future work."\n---\n\n# Plan\n')
    const { transforms } = await setupV2()
    const editor = mapEditor()
    transforms.skill(editor)
    expect(editor.items.get("archcore-plan")).toEqual({
      id: "archcore-plan",
      name: "archcore-plan",
      description: "Plan future work.",
      path: path.join(root, "skills", "plan", "SKILL.md"),
      content: "# Plan\n",
    })
    const mine = { id: "archcore-plan" }
    const existing = mapEditor({ "archcore-plan": mine })
    transforms.skill(existing)
    expect(existing.items.get("archcore-plan")).toBe(mine)
  })

  test("adds an /archcore-* command that attaches its skill to the prompt", async () => {
    writeSkill("plan", '---\nname: archcore-plan\ndescription: "Plan future work."\n---\n\n# Plan\n')
    const { transforms, prompts } = await setupV2()
    const commands: any[] = []
    transforms.command({ add: (command: any) => commands.push(command) })
    expect(commands.map((c) => [c.name, c.description])).toEqual([["archcore-plan", "Plan future work."]])
    await commands[0].execute({ sessionID: "s", prompt: { text: " the auth redesign " }, delivery: "queue" })
    await commands[0].execute({ sessionID: "s", prompt: { text: "", skills: [{ id: "other" }] }, delivery: "steer" })
    expect(prompts).toEqual([
      { text: "the auth redesign", sessionID: "s", skills: [{ id: "archcore-plan" }], delivery: "queue" },
      { text: "/archcore-plan", sessionID: "s", skills: [{ id: "other" }, { id: "archcore-plan" }], delivery: "steer" },
    ])
  })

  test("defines both agents in the V2 shape with the read-only ruleset", async () => {
    const { transforms } = await setupV2()
    const agents: Record<string, Record<string, unknown>> = {}
    transforms.agent({ update: (id: string, fn: (a: Record<string, unknown>) => void) => fn((agents[id] ??= { id })) })
    expect(agents["archcore-auditor"]).toEqual({
      id: "archcore-auditor",
      name: "archcore-auditor",
      description: "Read-only auditor.",
      mode: "subagent",
      steps: 40,
      system: "You audit.\n",
      permissions: [
        { action: "*", resource: "*", effect: "deny" },
        { action: "archcore_get_document", resource: "*", effect: "allow" },
        { action: "read", resource: "*", effect: "allow" },
        { action: "read", resource: "*.env", effect: "ask" },
        { action: "read", resource: "*.env.*", effect: "ask" },
        { action: "read", resource: "*.env.example", effect: "allow" },
      ],
    })
  })

  for (const name of ["v2-write-archcore.json", "v2-patch-archcore.json"]) {
    test(`hands ${name} to pre-tool-use verbatim and throws the reason on exit 2`, async () => {
      behave("pre-tool-use", "printf 'Use the MCP tools.\\n' >&2; exit 2\n")
      const f = fixture(name)
      const { hooks } = await setupV2()
      await expect(hooks["execute.before"]({ tool: f.tool_name, sessionID: "s", id: "c", input: f.tool_input })).rejects.toThrow(
        /^Use the MCP tools\.$/,
      )
      expect(received("pre-tool-use")).toEqual(f)
      expect(readFileSync(path.join(stubs, "pre-tool-use.cwd"), "utf8").trim()).toBe(project)
    })
  }

  test("ignores a tool outside the pre-mutation set", async () => {
    const { hooks } = await setupV2()
    await hooks["execute.before"]({ tool: "bash", sessionID: "s", id: "c", input: {} })
    expect(count("pre-tool-use")).toBe(0)
  })

  test("appends held pre-tool context to the same call's content", async () => {
    behave("pre-tool-use", "printf 'context'; exit 0\n")
    const { hooks } = await setupV2()
    await hooks["execute.before"]({ tool: "edit", sessionID: "s", id: "c", input: { path: "a.md" } })
    const event: any = { tool: "edit", sessionID: "s", id: "c", status: "completed", result: { content: [{ type: "text", text: "done" }] } }
    await hooks["execute.after"](event)
    expect(event.result.content).toEqual([
      { type: "text", text: "done" },
      { type: "text", text: "context" },
    ])
  })

  test("delivers post-tool validation on the session's next model request, once", async () => {
    behave("post-tool-use", "printf 'validation'; exit 0\n")
    behave("session-start", "exit 0\n")
    const f = fixture("mcp-update.json")
    const { hooks } = await setupV2()
    await hooks["execute.after"]({ tool: f.tool_name, sessionID: "s", id: "c", input: f.tool_input, status: "completed", result: { output: "{}" } })
    expect(received("post-tool-use")).toEqual(f)
    const first = { sessionID: "s", system: [] as any[] }
    await hooks.context(first)
    const second = { sessionID: "s", system: [] as any[] }
    await hooks.context(second)
    expect(first.system).toEqual([{ type: "text", text: "validation" }])
    expect(second.system).toEqual([])
  })

  test("skips post-tool validation for a failed call", async () => {
    const f = fixture("mcp-update.json")
    const { hooks } = await setupV2()
    await hooks["execute.after"]({ tool: f.tool_name, sessionID: "s", id: "c", input: f.tool_input, status: "error" })
    expect(count("post-tool-use")).toBe(0)
  })

  test("does not throw from execute.after or context on a malformed event", async () => {
    const { hooks } = await setupV2()
    await hooks["execute.after"]({ tool: "archcore_update_document", sessionID: "s", id: "c", status: "completed", result: null } as any)
    await hooks.context({ sessionID: "s" } as any)
  })

  test("spawns session-start once per session and appends the recap on every request", async () => {
    behave("session-start", "sleep 0.2; printf '[Archcore] recap\\n'; exit 0\n")
    const { hooks } = await setupV2()
    const requests = [{ sessionID: "s", system: [] as any[] }, { sessionID: "s", system: [] as any[] }]
    await Promise.all(requests.map((r) => hooks.context(r)))
    expect(count("session-start")).toBe(1)
    for (const r of requests) expect(r.system).toEqual([{ type: "text", text: "[Archcore] recap\n" }])
  })

  test("spawns no launcher when the CLI is older than the gate", async () => {
    behave("cli-gte", "echo no; exit 0\n")
    const { hooks } = await setupV2()
    await hooks["execute.before"]({ tool: "write", sessionID: "s", id: "c", input: {} })
    await hooks.context({ sessionID: "s", system: [] })
    expect(count("pre-tool-use") + count("session-start")).toBe(0)
  })
})
