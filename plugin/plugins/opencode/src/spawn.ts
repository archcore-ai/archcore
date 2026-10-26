import { spawn } from "node:child_process"
import { existsSync } from "node:fs"

export type Result = { code: number | null; stdout: string; stderr: string }

const NONE: Result = { code: null, stdout: "", stderr: "" }

// The answer for a launcher file that is not there. run has already named the
// path on stderr, so a caller compares against it instead of warning again.
export const MISSING: Result = { ...NONE }

// A null code means "no answer" (missing file, spawn error, or budget exceeded);
// every caller treats it as allow without context (opencode-adapter.spec, Failure Behavior).
export function run(file: string, args: string[], cwd: string, input: string, budgetMs: number): Promise<Result> {
  if (!existsSync(file)) {
    process.stderr.write(`[archcore] launcher missing: ${file}\n`)
    return Promise.resolve(MISSING)
  }
  return new Promise((resolve) => {
    let settled = false
    let timer: ReturnType<typeof setTimeout> | undefined
    const finish = (result: Result) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      resolve(result)
    }
    const child = spawn(file, args, {
      cwd,
      env: { ...process.env, ARCHCORE_HOST: "opencode" },
      stdio: ["pipe", "pipe", "pipe"],
      detached: true,
    })
    let stdout = ""
    let stderr = ""
    // Decoding per stream, not per chunk, keeps a character split across two
    // pipe reads whole.
    child.stdout.setEncoding("utf8").on("data", (chunk: string) => (stdout += chunk))
    child.stderr.setEncoding("utf8").on("data", (chunk: string) => (stderr += chunk))
    // The launcher's own children inherit the pipes, so a kill of the shell alone
    // would leave "close" waiting on them; the whole process group goes.
    timer = setTimeout(() => {
      try {
        process.kill(-child.pid!, "SIGKILL")
      } catch {}
      finish(NONE)
    }, budgetMs)
    child.on("error", () => finish(NONE))
    child.on("close", (code) => finish({ code, stdout, stderr }))
    child.stdin.on("error", () => {})
    child.stdin.end(input)
  })
}
