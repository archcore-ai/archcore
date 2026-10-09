import { atom, read, update } from 'claude-code'
import type { EngineInterface, PromptOrigin, Register } from 'claude-code'

import type { Hint, Session, Turn } from '../types'

type Step = Pick<Hint, 'reason' | 'command'>
type Row = Record<string, unknown>

const EMPTY: Turn = {
  requestId: '', turnId: '', isDecision: false, edits: [], editCount: 0, pushed: false, drafts: 0,
  emptySearch: null, archcoreCommand: false, plan: null, editsAfterPlan: 0, planFile: null, found: [], read: [],
}
const FRESH: Session = { reads: 0, superpowers: false }
// Bump SHAPE when Turn, Session or Hint change: a reload then drops what the old code wrote.
const SHAPE = { shape: 'v4' }
const turn = atom({ plugin: 'archcore', key: 'turn' } as const, EMPTY, SHAPE)
const session = atom({ plugin: 'archcore', key: 'session' } as const, FRESH, SHAPE)
const hint = atom({ plugin: 'archcore', key: 'hint' } as const, null as Hint | null, SHAPE)

// The Archcore MCP server as a plugin install and as a project .mcp.json name it.
const SERVERS = ['plugin_archcore_archcore', 'archcore']
const ARCHCORE_TOOL = /^mcp__(.*archcore.*)__(get_document|search_documents|create_document)$/
// An /archcore:* command owns its own follow-up: the mod stays quiet in its request.
// A Superpowers run keeps to its own steps (AGENTS.md Integrations rule 4): the
// session's `superpowers` flag keeps the mod quiet through the whole flow.
const OWN_SKILL = /^archcore:/
// A typed slash command: the prompt's first word, or the engine's <command-name> envelope.
const COMMAND = /^\s*\/([\w-]+(?::[\w-]+)?)(?=\s|$)|<command-name>\/?([\w-]+(?::[\w-]+)?)<\/command-name>/
// Plan files written outside .archcore/. Superpowers keeps its design and plan files
// under docs/superpowers/ (AGENTS.md Integrations, the rules on docs/superpowers/specs/
// and docs/superpowers/plans/ files), so nothing there counts.
const PLAN_FILE = /(^|\/)plan\.md$|(^|\/)plans\/[^/]+\.md$/i
const SUPERPOWERS = /(^|\/)docs\/superpowers\//
// search_documents filters that make an empty answer say nothing about the topic.
const FILTERS = ['types', 'status', 'source', 'mtime_after', 'path_ref']
// Prompt origins that are no new request from the person: they keep the request and its hint.
const NOT_USER = new Set([
  'task-notification', 'scheduled-trigger', 'peer', 'peer-send-message', 'projects-relay',
  'channel', 'coordinator', 'observer', 'observer-activity', 'auto-continuation',
])
// git options that take the next word as their value.
const GIT_VALUE_OPTIONS = new Set(['-C', '-c', '--git-dir', '--work-tree', '--namespace', '--config-env'])
const MAX_TURN_EDITS = 20
const MAX_PENDING = 20
const MAX_DOCS = 200
const LOOKUP_MS = 1500
// The Archcore server connects after the session starts: an unknown track lookup
// is tried again after each of these waits, until the first request starts.
const START_RETRY_MS = [2000, 5000, 10000, 20000]

// ponytail: keyword heuristics on the prompt and the command line; swap for a
// model classifier (decision) or a real shell parser (push) if they misfire.
// A decision is an explicit first-person statement: "we decided …", "решили: …",
// "from now on we …". A sentence that tells the assistant how to work ("from now
// on, answer in English", "договорились, продолжай") is an instruction, not a decision.
const WE_DECIDED = /(?:^|[.!?\n])\s*(?:we decided|we['’]ve decided|we['’]re going with)\b/i
// "решили" or "договорились" opens a sentence (after at most two words, none of them "не")
// and a separator follows; after a comma only "что" or a we-verb ("используем", "будем") goes on.
const RU_DECIDED = /(?:^|[.!?\n])\s*(?:(?!не\s)[\p{L}-]+,?\s+){0,2}?(?:мы\s+)?(?:решили|договорились)\s*(?:([:—–]|-(?=\s))|,)\s*(\p{L}+)/iu
const FROM_NOW_ON = /(?:^|[.!?\n])\s*from now on\b|(?:отныне|с этого момента|теперь всегда)(?!\p{L})/iu
const NEXT_WORD = /^[\s,:—–-]*(?:(?:always|never|всегда|никогда)\s+)?([\p{L}'’-]+)/iu
const ADDRESSED = /^(?:you|answer|reply|respond|write|speak|talk|use|call|please|don['’]t|do|be|keep|ask|tell|show|explain|translate|format|ты|вы|пиши(?:те)?|говори(?:те)?|ответь(?:те)?|переведи(?:те)?|[а-яё]+(?:ай|яй|ей|уй|ой)(?:те)?)$/iu
const WE_VERB = /^(?:что|[а-яё]+(?:ем|ём|им)(?:ся)?)$/iu

let server = SERVERS[0]!
// Prompts submitted and not started yet, oldest first: a turn.start takes the one with its text.
let pending: { text: string; isUser: boolean }[] = []
// A skill.prompt raised before its turn.start (a typed /name) marks the coming turn.
let isSkillPending = false
// Between a turn.start and the main loop's turn.complete; a reload forgets it.
let isRunning = false
// Moved by every turn.start and /clear. State reads within one dispatch see one
// moment, so a turn.complete compares this instead to see that it went stale.
let epoch = 0
const logged = new Set<string>()

const str = (value: unknown) => (typeof value === 'string' ? value : '')
const isRow = (value: unknown): value is Row => typeof value === 'object' && value !== null && !Array.isArray(value)
const rows = (value: unknown): Row[] => (Array.isArray(value) ? value.filter(isRow) : [])
const hasValue = (value: unknown) => (Array.isArray(value) ? value.length > 0 : value !== undefined && value !== null && value !== '')
const toEnd = (list: readonly string[], item: string) => [...list.filter(one => one !== item), item]
// ponytail: the first 200 distinct paths only; the count stops there.
const addAll = (list: readonly string[], items: readonly string[]) =>
  [...new Set([...list, ...items.filter(Boolean)])].slice(0, MAX_DOCS)
// C0 and C1 control characters out, whitespace runs to one space: what the band and the box show.
const clean = (text: string) => text.replace(/[\u0000-\u001f\u007f-\u009f]/g, ' ').replace(/\s+/g, ' ').trim()
const why = (err: unknown) => clean(err instanceof Error ? err.message : String(err)).slice(0, 120)

function parse(text: unknown): Row | undefined {
  try {
    const value: unknown = JSON.parse(str(text))
    return isRow(value) ? value : undefined
  } catch {
    return undefined
  }
}

// First Markdown heading, else the first non-empty line: one clean line, 80 characters at most.
function titleOf(text: string, fallback: string): string {
  const lines = text.split('\n').map(clean).filter(Boolean)
  const line = lines.find(one => one.startsWith('#')) ?? lines[0] ?? clean(fallback)
  return line.replace(/^#+\s*/, '').slice(0, 80)
}

// One debug-log line per distinct reason a lookup or call was given up.
function note($: EngineInterface, reason: string) {
  if (logged.has(reason)) return
  logged.add(reason)
  $.ui.log(reason, { to: 'debug' })
}

function isDecision(prompt: string): boolean {
  if (prompt.trimEnd().endsWith('?')) return false
  if (WE_DECIDED.test(prompt)) return true
  const ru = RU_DECIDED.exec(prompt)
  if (ru && (ru[1] ? !ADDRESSED.test(ru[2] ?? '') : WE_VERB.test(ru[2] ?? ''))) return true
  const later = FROM_NOW_ON.exec(prompt)
  const word = later && NEXT_WORD.exec(prompt.slice(later.index + later[0].length))?.[1]
  return typeof word === 'string' && !ADDRESSED.test(word)
}

// Heredoc bodies and quoted text are data, not commands; the rest splits into simple
// commands, each read past VAR=value prefixes and git's global options.
function isPush(command: string): boolean {
  const code = command
    .replace(/<<-?[ \t]*(['"]?)(\w+)\1([^\n]*)[\s\S]*?\n[ \t]*\2[ \t]*(?=\n|$)/g, '<<$3')
    .replace(/'[^']*'|"(?:[^"\\]|\\[\s\S])*"/g, "''")
  return code.split(/[;&|\n()`]/).some(part => {
    const words = part.trim().split(/\s+/)
    let i = 0
    while (/^[A-Za-z_]\w*=/.test(words[i] ?? '')) i++
    const [tool, verb, action] = words.slice(i)
    if ((tool === 'gh' && verb === 'pr') || (tool === 'glab' && verb === 'mr')) {
      return action === 'create' && !words.includes('--dry-run')
    }
    if (tool !== 'git') return false
    i++
    while (words[i]?.startsWith('-')) i += GIT_VALUE_OPTIONS.has(words[i]!) ? 2 : 1
    return words[i] === 'push' && !words.slice(i + 1).some(word => word === '-n' || word === '--dry-run')
  })
}

// Absolute, '/'-separated, '.' and '..' folded, the drive letter upper-cased; a relative
// path is taken from `base`.
function absolute(path: string, base: string): string {
  const slashed = path.replace(/\\/g, '/')
  const [head = '', ...parts] = (/^(?:\/|[A-Za-z]:\/)/.test(slashed) ? slashed : `${base}/${slashed}`).split('/')
  const kept: string[] = []
  for (const part of parts) {
    if (part === '..') kept.pop()
    else if (part !== '' && part !== '.') kept.push(part)
  }
  return `${head.toUpperCase()}/${kept.join('/')}`
}

function under(file: string, base: string): string | undefined {
  const prefix = base.endsWith('/') ? base : `${base}/`
  return file.startsWith(prefix) ? file.slice(prefix.length) : undefined
}

const rootOf = async ($: EngineInterface) => absolute(await $.session.root(), '/')
const inRoot = (root: string, rel: string) => `${root}/${rel}`

async function realPath($: EngineInterface, path: string): Promise<string | undefined> {
  try {
    const real = (await $.fs.stat(path, { resolve: true })).realPath
    return real === undefined ? undefined : absolute(real, '/')
  } catch (err) {
    note($, `fs.stat failed: ${why(err)}`)
    return undefined
  }
}

// Project-relative path, or undefined outside the project root. A spelling outside the
// root is checked again by where it lands, as a root under /var lands under /private/var.
async function projectPath($: EngineInterface, path: string): Promise<string | undefined> {
  const root = await rootOf($)
  const file = absolute(path, root)
  const rel = under(file, root)
  if (rel !== undefined) return rel
  const [realFile, realRoot] = await Promise.all([realPath($, file), realPath($, root)])
  return realFile && realRoot ? under(realFile, realRoot) : undefined
}

// undefined when unknown: the engine refused the check.
async function hasArchcore($: EngineInterface): Promise<boolean | undefined> {
  try {
    return await $.fs.exists(inRoot(await rootOf($), '.archcore'))
  } catch (err) {
    note($, `fs.exists failed: ${why(err)}`)
    return undefined
  }
}

const LATE: unique symbol = Symbol('late')

// The work's answer, or undefined after LOOKUP_MS. A sleep the clock refuses sets no deadline.
async function timed<T>($: EngineInterface, what: string, work: Promise<T | undefined>): Promise<T | undefined> {
  const stop = new AbortController()
  const deadline = $.clock.sleep(LOOKUP_MS, { signal: stop.signal }).then(
    (): typeof LATE => LATE,
    () => new Promise<never>(() => {}),
  )
  try {
    const value = await Promise.race([work, deadline])
    if (value !== LATE) return value
    note($, `${what} took over ${LOOKUP_MS} ms`)
    return undefined
  } finally {
    stop.abort()
  }
}

async function callServer($: EngineInterface, name: string, tool: string, args: Row): Promise<Row | undefined> {
  try {
    const res = await $.mcp.call(name, tool, args)
    const body = res.isError ? undefined : parse(res.content.find(block => block.type === 'text')?.text)
    if (!body) note($, `${tool} on ${name} answered ${res.isError ? 'an error' : 'no JSON object'}`)
    return body
  } catch (err) {
    note($, `${tool} on ${name} failed: ${why(err)}`)
    return undefined
  }
}

// The tool's JSON answer, or undefined when it is unknown: every server threw,
// reported an error or answered no JSON object, or the deadline passed.
function callArchcore($: EngineInterface, tool: string, args: Row): Promise<Row | undefined> {
  return timed($, tool, (async () => {
    for (const name of [server, ...SERVERS.filter(one => one !== server)]) {
      const body = await callServer($, name, tool, args)
      if (body) {
        server = name
        return body
      }
    }
    return undefined
  })())
}

async function nextStep($: EngineInterface, t: Turn, s: Session): Promise<Step | null> {
  if (t.archcoreCommand || s.superpowers) return null
  // No .archcore/ (or no answer): only the session-start /archcore:init hint speaks.
  if ((await hasArchcore($)) !== true) return null

  // A plan approved or proposed and not yet implemented; once edits follow,
  // /archcore:plan no longer fits ("not for documenting existing code").
  if (t.plan && t.editsAfterPlan === 0) {
    const state = t.plan.isProposed ? 'proposed' : 'approved'
    return { reason: `The ${state} plan is outside .archcore/. Record it?`, command: `/archcore:plan ${t.plan.title}` }
  }

  if (t.isDecision) {
    return { reason: 'This sounds like a decision. Record it?', command: '/archcore:document decision ' }
  }

  // Drift, coverage and closeout need the engine's code alignment and staleness
  // advisories (engine-runtime-boundary.adr): the runtime does not compute them.
  if (t.pushed) {
    return { reason: 'The branch went out for review.', command: '/archcore:review' }
  }
  if (t.drafts > 0) {
    return { reason: `${t.drafts} draft document(s) created; they are not accepted yet.`, command: '/archcore:review' }
  }
  if (t.planFile && t.edits.length === 0) {
    let topic = titleOf('', t.planFile)
    try {
      topic = titleOf(await $.fs.read(inRoot(await rootOf($), t.planFile)), t.planFile)
    } catch (err) {
      note($, `fs.read of a plan file failed: ${why(err)}`)
    }
    return { reason: `A plan was written to ${clean(t.planFile)}, outside Archcore.`, command: `/archcore:plan ${topic}` }
  }
  if (t.edits.length > 0 && s.reads === 0) {
    return { reason: `${t.editCount} file(s) edited without reading any Archcore document this session.`, command: '/archcore:review' }
  }
  if (t.emptySearch) {
    return { reason: `Nothing in Archcore matches "${t.emptySearch}".`, command: `/archcore:document code ${t.emptySearch}` }
  }
  return null
}

// The command that resumes a track, by its id (skills/_shared/tracks/<id>.md and the
// argument hints of skills/*/SKILL.md). Standalone evidence is filed through document research.
function resumeOf(track: string, title: string, type: string): string | undefined {
  switch (track) {
    case 'research':
      return type === 'evidence' ? `/archcore:document research ${title}` : `/archcore:plan ${title}`
    case 'sdd':
    case 'requirements-cascade':
      return `/archcore:plan ${title}`
    case 'decision':
      return `/archcore:document decision ${title}`
    case 'describe':
      return `/archcore:document code ${title}`
    case 'actualize':
    case 'closeout':
    case 'experience':
      return '/archcore:review'
    case 'import':
      return '/archcore:init import'
    default:
      return undefined
  }
}

// The first `gate:` inside an archcore:track block outside fenced code, unless it is
// the gate contract's `<track>.<stage>` placeholder.
function gateOf(content: string): string | undefined {
  const text = content.replace(/^ {0,3}(`{3,}|~{3,})[^\n]*\n[\s\S]*?^ {0,3}\1[^\n]*$/gm, '')
  for (const [, block = ''] of text.matchAll(/<!-- archcore:track\b([\s\S]*?)-->/g)) {
    const gate = /^gate:\s*([a-z][a-z-]*\.[a-z-]+)\s*$/m.exec(block)?.[1]
    if (gate) return gate
  }
  return undefined
}

// A local draft whose track block names a real gate is a track its command resumes; plans first.
// Undefined when the search answer is unknown, null when no draft is stopped.
async function stoppedTrack($: EngineInterface): Promise<Step | null | undefined> {
  const body = await callArchcore($, 'search_documents', {
    content: '<!-- archcore:track', match: 'exact', status: 'draft', source: 'local', limit: 10,
  })
  if (!body || !Array.isArray(body.results)) return undefined
  const drafts = rows(body?.results).filter(row => row.source_kind === 'local')
  for (const row of [...drafts.filter(one => one.type === 'plan'), ...drafts.filter(one => one.type !== 'plan')]) {
    const gate = gateOf(str((await callArchcore($, 'get_document', { path: row.path }))?.content))
    const [track = '', stage = ''] = gate?.split('.') ?? []
    const title = titleOf(str(row.title), str(row.path))
    const command = resumeOf(track, title, str(row.type))
    if (command) return { reason: `Draft "${title}" stopped at the ${stage.replace(/-/g, ' ')} step.`, command }
  }
  return null
}

// The search topic when a filter-free search_documents found nothing and no near miss.
function emptySearchTopic(args: Row, body: Row | undefined): string {
  if (!body || !Array.isArray(body.results) || body.results.length > 0) return ''
  if (hasValue(body.near_misses) || FILTERS.some(key => hasValue(args[key]))) return ''
  return titleOf(str(args.content), '')
}

async function recordEdit($: EngineInterface, path: string) {
  const rel = await projectPath($, path)
  // Edits outside the project, to documents, and to Superpowers files say nothing about code.
  if (rel === undefined || rel.startsWith('.archcore/') || SUPERPOWERS.test(rel)) return
  if (PLAN_FILE.test(rel)) {
    await update($, turn, t => (t.planFile ? t : { ...t, planFile: rel }))
    return
  }
  await update($, turn, t => ({
    ...t,
    edits: toEnd(t.edits, rel).slice(-MAX_TURN_EDITS),
    // ponytail: a file re-edited after it left the last 20 counts twice.
    editCount: t.editCount + (t.edits.includes(rel) ? 0 : 1),
    editsAfterPlan: t.plan ? t.editsAfterPlan + 1 : 0,
  }))
}

// A hint names a command the person can run now, or none: the Archcore plugin may be off.
async function isAvailable($: EngineInterface, command: string): Promise<boolean> {
  const name = command.slice(1).split(' ')[0]
  try {
    if ((await $.command.list()).some(one => one.name === name)) return true
    note($, `/${name} is not available; no hint`)
  } catch (err) {
    note($, `command.list failed: ${why(err)}`)
  }
  return false
}

const suggest = ($: EngineInterface, command: string) =>
  $.prompt.suggest({ text: command }).then(
    done => done.isShown,
    (err: unknown) => {
      note($, `prompt.suggest failed: ${why(err)}`)
      return false
    },
  )

const markShown = ($: EngineInterface) => update($, hint, h => (h ? { ...h, shown: true } : h))

// Draws the step in the band at once; "Tab →" replaces "run:" once the box takes it.
// `started` is the epoch the step was computed in: a later request owns the band.
async function show($: EngineInterface, step: Step, requestId: string, started: number) {
  if (started !== epoch || !(await isAvailable($, step.command))) return
  await update($, hint, () => ({ ...step, requestId, shown: false }))
  // The write may land after the next request began: the band skips such a hint; the box must not get it.
  if (started !== epoch) return
  if (await suggest($, step.command)) {
    await markShown($)
    return
  }
  // The box declines while the turn winds down: one more try once the session is idle.
  $.clock.after(0, () => {
    if (started !== epoch) return
    suggest($, step.command)
      .then(isShown => (isShown ? markShown($) : undefined))
      .catch((err: unknown) => note($, `retry of prompt.suggest failed: ${why(err)}`))
  })
}

// The session-start hint: /archcore:init without .archcore/, else a stopped track to resume.
async function startHint($: EngineInterface, started: number, attempt = 0): Promise<void> {
  if (started !== epoch) return
  const has = await hasArchcore($)
  if (has === false) return show($, { reason: 'This repository has no .archcore/.', command: '/archcore:init' }, '', started)
  if (!has) return
  const step = await stoppedTrack($)
  if (step) return show($, step, '', started)
  if (step === null) return
  const wait = START_RETRY_MS[attempt]
  if (wait === undefined) return note($, 'session start: the Archcore server did not answer; no resume hint')
  $.clock.after(wait, () => {
    startHint($, started, attempt + 1).catch((err: unknown) => note($, `session start lookup failed: ${why(err)}`))
  })
}

const isUserOrigin = (origin: PromptOrigin) => (origin.kind === 'plugin' ? origin.asUser === true : !NOT_USER.has(origin.kind))
const commandOf = (prompt: string) => {
  const m = COMMAND.exec(prompt)
  return m?.[1] ?? m?.[2]
}

export const register: Register = (on, options) => {
  // The /config row "Next-step hints" (userConfig next_step_hints) turns the mod off;
  // a change reloads the module with the new value.
  if (options.next_step_hints === false) return
  on('session.start', async ($, e, next) => {
    const done = await next(e)
    // A -p run or an SDK host draws nowhere yet: no lookups and no hint.
    if (!e.isInteractive || e.surface === null) return done
    // The session's first start only: a reload, an enable or a respawn keeps the live hint.
    if ((await read($, turn)).turnId !== '' || (await read($, hint)) !== null) return done
    const started = epoch
    // Detached, so the lookups do not hold the first prompt.
    $.clock.after(0, () => {
      startHint($, started).catch((err: unknown) => note($, `session start lookup failed: ${why(err)}`))
    })
    return done
  })

  // A /clear ends the conversation; no session.start follows it.
  on('session.end', async ($, e, next) => {
    if (e.reason === 'clear') {
      epoch++
      isSkillPending = false
      pending = []
      await update($, turn, () => EMPTY)
      await update($, hint, () => null)
      await update($, session, () => FRESH)
    }
    return next(e)
  })

  on('prompt.submit', ($, e, next) => {
    pending = [...pending, { text: e.text.trim(), isUser: isUserOrigin(e.origin) }].slice(-MAX_PENDING)
    return next(e)
  })

  on('skill.prompt', async ($, e, next) => {
    if (OWN_SKILL.test(e.skill)) {
      isSkillPending = true
      await update($, turn, t => ({ ...t, archcoreCommand: true }))
    }
    // A superpowers:* skill starts the quiet flow; an /archcore:* skill, or another
    // skill the person typed (expanded before its turn starts), ends it.
    const isSuperpowers = e.skill.startsWith('superpowers:')
    if (isSuperpowers || OWN_SKILL.test(e.skill) || !isRunning) {
      await update($, session, s => (s.superpowers === isSuperpowers ? s : { ...s, superpowers: isSuperpowers }))
    }
    return next(e)
  })

  on('turn.start', async ($, e, next) => {
    const prompt = e.text.trim()
    const i = pending.findIndex(one => one.text === prompt)
    const isUser = i < 0 || pending[i]!.isUser
    if (i >= 0) pending.splice(i, 1)
    const command = commandOf(prompt)
    const isOwnCommand = isSkillPending || (command !== undefined && OWN_SKILL.test(command))
    // An empty text continues the same request; a notification, a peer or a
    // schedule is no new user request: both keep the request and the hint.
    const isNewRequest = prompt !== '' && isUser
    isRunning = true
    epoch++
    if (isNewRequest) {
      await update($, turn, () => ({
        ...EMPTY, requestId: e.turnId, turnId: e.turnId, isDecision: isDecision(prompt), archcoreCommand: isOwnCommand,
      }))
      if (command !== undefined) {
        const isSuperpowers = command.startsWith('superpowers:')
        await update($, session, s => (s.superpowers === isSuperpowers ? s : { ...s, superpowers: isSuperpowers }))
      }
    } else {
      await update($, turn, t => ({ ...t, turnId: e.turnId, archcoreCommand: t.archcoreCommand || isOwnCommand }))
    }
    return next(e)
  })

  on('tool.call', { tool: ['Write', 'Edit', 'NotebookEdit'] }, async ($, e, next) => {
    const ran = await next(e)
    if (ran.deny === undefined && !ran.isError) await recordEdit($, str(e.tool === 'NotebookEdit' ? e.notebook_path : e.file_path))
    return ran
  })

  on('tool.call', { tool: 'ExitPlanMode' }, async ($, e, next) => {
    const ran = await next(e)
    const plan = ran.deny === undefined && !ran.isError && isRow(ran.result) ? str(ran.result.plan) : ''
    if (plan) {
      await update($, turn, t => ({ ...t, plan: { title: titleOf(plan, 'the approved plan'), isProposed: false }, editsAfterPlan: 0 }))
    }
    return ran
  })

  on('tool.call', { tool: 'Agent', subagent_type: 'Plan' }, async ($, e, next) => {
    const ran = await next(e)
    if (ran.deny === undefined && !ran.isError) {
      await update($, turn, t => ({ ...t, plan: { title: titleOf(e.description, 'the plan'), isProposed: true }, editsAfterPlan: 0 }))
    }
    return ran
  })

  on('tool.call', { tool: 'Bash', command: /\b(?:git|gh|glab)\b/ }, async ($, e, next) => {
    const ran = await next(e)
    if (ran.deny === undefined && !ran.isError && isPush(e.command)) await update($, turn, t => ({ ...t, pushed: true }))
    return ran
  })

  on('tool.call', { tool: ARCHCORE_TOOL }, async ($, e, next) => {
    const ran = await next(e)
    const m = ARCHCORE_TOOL.exec(e.tool)
    if (ran.deny !== undefined || ran.isError || !m) return ran
    server = m[1] ?? server
    const args = e as unknown as Row
    if (m[2] === 'create_document') {
      await update($, turn, t => ({ ...t, drafts: t.drafts + 1 }))
    } else if (m[2] === 'get_document') {
      await update($, session, s => ({ ...s, reads: s.reads + 1 }))
      await update($, turn, t => ({ ...t, read: addAll(t.read, [str(args.path)]) }))
    } else {
      const body = parse(ran.text)
      const paths = rows(body?.results).map(row => str(row.path))
      if (args.mode === 'full' && paths.length > 0) await update($, session, s => ({ ...s, reads: s.reads + 1 }))
      await update($, turn, t => ({
        ...t, found: addAll(t.found, paths), read: args.mode === 'full' ? addAll(t.read, paths) : t.read,
      }))
      const topic = emptySearchTopic(args, body)
      if (topic) await update($, turn, t => (t.emptySearch ? t : { ...t, emptySearch: topic }))
    }
    return ran
  })

  on('turn.complete', async ($, e, next) => {
    const done = await next(e)
    // A subagent's run raises no turn.start; its completion is not the request's.
    if (e.agentId !== undefined) return done
    isSkillPending = false
    isRunning = false
    const started = epoch
    const t = await read($, turn)
    if (t.turnId !== e.turnId || e.reason !== 'answer') return done
    // Nothing draws the band or the box (a -p run, an SDK host before it attaches).
    if ((await $.session.surfaces()).length === 0) return done
    const step = await nextStep($, t, await read($, session))
    if (step) await show($, step, t.requestId, started)
    return done
  })

  // The engine guesses the next prompt after the turn too and would replace ours.
  on('prompt.suggest', { origin: { kind: 'suggestion' } }, async ($, e, next) => {
    const step = await read($, hint)
    if (!step || step.requestId !== (await read($, turn)).requestId) return next(e)
    const done = await next({ ...e, text: step.command })
    if (done.isShown) await markShown($)
    return done
  })

  on('ui.render', { component: 'AbovePrompt' }, async ($, e, next) => {
    // A survey, a running turn, or a subagent's transcript.
    if (e.props.hasSurvey || e.props.isWorking || e.props.view.agentId !== undefined) return next(e)
    const t = await read($, turn)
    const stored = await read($, hint)
    // A hint of an earlier request is a late write.
    const step = stored && stored.requestId === t.requestId ? stored : null
    const docs = t.found.length + t.read.length > 0 ? `documents: ${t.found.length} found · ${t.read.length} read` : ''
    if (!step && !docs) return next(e)
    const { Box, Button, Text } = $.ui.resolve(e)
    // A band tree replaces later mods' band content; theirs stays below ours.
    const below = await next(e)
    if (!step) {
      return (
        <Box flexDirection="column">
          <Box>
            <Text color="cyan">archcore </Text>
            <Text dimColor wrap="truncate-end">{docs}</Text>
          </Box>
          {below}
        </Box>
      )
    }
    return (
      <Box flexDirection="column">
        <Box>
          <Button key="hide" label="Hide" onPress={() => update($, hint, () => null)} />
          <Text color="cyan"> archcore </Text>
          <Text wrap="truncate-end">{step.reason}</Text>
        </Box>
        <Text dimColor wrap="truncate-end">{`${step.shown ? 'Tab →' : 'run:'} ${step.command.trim()}`}</Text>
        {docs ? <Text dimColor wrap="truncate-end">{docs}</Text> : null}
        {below}
      </Box>
    )
  })
}
