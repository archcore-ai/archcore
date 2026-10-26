import { afterEach, beforeEach, describe, expect, test } from "bun:test"
import { chmodSync, existsSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, realpathSync, rmSync, writeFileSync } from "node:fs"
import { tmpdir } from "node:os"
import path from "node:path"
import { BUDGET_MS, createServer } from "./bridge"

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
})

afterEach(() => {
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
    BUDGET_MS.pre = 200
    try {
      behave("pre-tool-use", 'sleep 5 & echo $! > "$STUB_DIR/child.pid"; wait; exit 2\n')
      const started = Date.now()
      await (await hooks())["tool.execute.before"]({ tool: "write", sessionID: "s", callID: "c" }, { args: {} })
      expect(Date.now() - started).toBeLessThan(2000)
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
