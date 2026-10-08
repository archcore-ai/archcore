import type { On, PromptOrigin, ToolCallArgs } from 'claude-code'
import { expect, mock, test as base } from 'claude-code/testing'
import type { Engine, MockClock, Plugin, TestRest } from 'claude-code/testing'

const ROOT = '/repo'
const PLUGIN = 'plugin_archcore_archcore'
const SURFACES = ['terminal', 'desktop'] as const
const COMMANDS = ['archcore:plan', 'archcore:document', 'archcore:review', 'archcore:init']
type Surface = (typeof SURFACES)[number]

// A document the stub server holds; `content` is its body.
type Doc = {
  path: string
  title: string
  type: string
  status: string
  isGlobal?: boolean
  content?: string
}
type Call = { server: string; tool: string; args: Record<string, unknown> }
type Seen = { calls: Call[]; suggested: string[]; reached: string[]; skips: unknown[]; logs: string[] }
type World = {
  docs?: Doc[]
  root?: string
  hasArchcore?: boolean | 'throw'
  // The first .archcore check waits for it, as a slow file system would.
  hold?: Promise<void>
  files?: Record<string, string>
  plan?: string | null
  mcp?: 'throw' | 'error' | 'garbage' | 'array' | 'no-results' | 'extra-block'
  servers?: readonly string[]
  // The first N calls find no connected Archcore server, as right after startup.
  offlineCalls?: number
  gate?: Promise<void>
  searchAnswer?: unknown
  searchFor?: (args: Record<string, unknown>) => unknown
  deny?: readonly string[]
  fail?: readonly string[]
  suggest?: 'shown' | 'hidden' | 'hidden-once' | 'reject' | 'hide-own'
  commands?: readonly string[] | 'throw'
  surfaces?: readonly Surface[]
  // Symbolic links by path prefix, as $.fs.stat({ resolve }) follows them.
  links?: Record<string, string>
  missing?: readonly string[]
}

// The engine skips a hook of the mod that threw or overran and goes on without it.
// This plugin sits above the mod and reports each such outcome.
let current: Seen | undefined
const WATCH: Plugin = {
  name: 'watch',
  tier: 'prepend',
  register: on => {
    on('*', async ($, e, next) => {
      const result = await next(e)
      const mine = next.trace.filter(entry => entry.plugin === 'archcore')
      for (const entry of mine) {
        if (entry.outcome !== 'returned' && entry.outcome !== 'passed') {
          await $.mcp.call('watch', 'skipped', { event: entry.event, outcome: entry.outcome })
        }
      }
      return result
    })
  },
}
// A test reads the mod's state through this plugin: the command /peek <turn|session|hint>.
const PEEK: Plugin = {
  name: 'peek',
  register: on => {
    on('command.run', { command: 'peek' }, async ($, e) => {
      // The kit takes only string-literal keys.
      const { value } = await (e.args === 'hint'
        ? $.state.get({ plugin: 'archcore', key: 'hint' })
        : e.args === 'session'
          ? $.state.get({ plugin: 'archcore', key: 'session' })
          : $.state.get({ plugin: 'archcore', key: 'turn' }))
      return { text: JSON.stringify(value ?? null) }
    })
  },
}
// The kit's test(name, body) and test(name, { options }, body); the inline plugins are this file's.
const test = (name: string, ...rest: TestRest) => {
  const [options, body] = rest.length === 2 ? rest : [{}, rest[0]] as const
  base(name, { ...options, plugins: [WATCH, PEEK] }, async ($, on) => {
    current = undefined
    await body($, on)
    const seen = current as Seen | undefined
    if (seen) expect(seen.skips).toEqual([])
  })
}

// The mod's state as the host holds it: `{ shape, value }`.
async function peek($: Engine, key: 'turn' | 'session' | 'hint') {
  const presentation = { isFullscreen: false, columns: 80 }
  const { text } = await $.command.run({ command: 'peek', args: key, origin: { kind: 'composer' }, presentation })
  return JSON.parse(text ?? 'null') as { shape: string; value: Record<string, unknown> | null } | null
}

// The server's answer to a content search. It honors status and limit, and ignores `source`
// on purpose, so the mod's own source_kind filter is what keeps global rows out.
function answer(world: World, tool: string, args: Record<string, unknown>): Record<string, unknown> {
  const docs = world.docs ?? []
  if (tool === 'get_document') return { content: docs.find(doc => doc.path === args.path)?.content ?? '' }
  const results = docs.flatMap(doc => {
    if (args.status !== undefined && doc.status !== args.status) return []
    if (typeof args.content !== 'string' || !(doc.content ?? '').includes(args.content)) return []
    return [{ path: doc.path, title: doc.title, type: doc.type, status: doc.status, source_kind: doc.isGlobal ? 'global' : 'local', matches: [] }]
  })
  const limit = typeof args.limit === 'number' ? args.limit : 50
  return { coverage: { local: docs.length }, hits: { local: results.length }, truncated: results.length > limit, index: [], results: results.slice(0, limit) }
}

const EMPTY_SEARCH = { coverage: { local: 3 }, hits: { local: 0 }, truncated: false, index: [], results: [] }
const slashed = (path: string) => path.replace(/\\/g, '/').replace(/\/+$/, '').toLowerCase()

// Stands in for the engine and every mod beneath this one: the turn and prompt
// events, the project root, the files, the Archcore server, the tools and the band.
function engine(on: On, world: World = {}): Seen {
  const seen: Seen = { calls: [], suggested: [], reached: [], skips: [], logs: [] }
  current = seen
  const root = world.root ?? ROOT
  let declined = 0
  let offline = world.offlineCalls ?? 0
  on('turn.start', ($, e) => {
    seen.reached.push(`turn.start:${e.turnId}`)
    return { turnId: e.turnId }
  })
  on('turn.complete', () => {
    seen.reached.push('turn.complete')
    return { text: 'ok' }
  })
  on('prompt.submit', ($, e) => {
    seen.reached.push(`prompt.submit:${e.text}`)
    return { text: e.text, origin: e.origin }
  })
  on('skill.prompt', ($, e) => {
    seen.reached.push(`skill.prompt:${e.skill}`)
    return { text: e.text }
  })
  on('session.start', ($, e) => {
    seen.reached.push('session.start')
    return { cwd: e.cwd }
  })
  on('session.end', ($, e) => {
    seen.reached.push(`session.end:${e.reason}`)
    return { sessionId: e.sessionId }
  })
  on('session.root', () => ({ value: root }))
  on('session.surfaces', () => ({ value: world.surfaces ?? ['terminal'] }))
  on('command.list', () =>
    world.commands === 'throw'
      ? { deny: 'no command table' }
      : { value: (world.commands ?? COMMANDS).map(name => ({ name, description: name, source: 'plugin' as const, plugin: 'archcore' })) },
  )
  on('ui.log', ($, e) => {
    seen.logs.push(e.text)
    return { value: undefined }
  })
  // The kit resolves a path that is not absolute here (a Windows one on macOS) against its cwd.
  const isAt = (path: string, rel: string) => slashed(path).endsWith(`${slashed(root)}/${rel.toLowerCase()}`)
  let hold = world.hold
  on('fs.exists', async ($, e) => {
    const wait = hold
    hold = undefined
    await wait
    return world.hasArchcore === 'throw' ? { deny: 'no file system' } : { value: (world.hasArchcore ?? true) && isAt(e.path, '.archcore') }
  })
  on('fs.read', ($, e) => {
    const rel = Object.keys(world.files ?? {}).find(key => isAt(e.path, key))
    return rel === undefined ? { deny: 'missing' } : { value: world.files![rel]! }
  })
  on('fs.stat', ($, e) => {
    if (world.missing?.includes(e.path)) return { deny: `ENOENT: ${e.path}` }
    const link = Object.entries(world.links ?? {}).find(([from]) => e.path === from || e.path.startsWith(`${from}/`))
    const landed = link ? `${link[1]}${e.path.slice(link[0].length)}` : e.path
    // A Windows realPath comes back with backslashes.
    const realPath = link?.[1].includes('\\') ? landed.replace(/\//g, '\\') : landed
    return { value: { kind: 'file' as const, size: 0, mtimeMs: 0, isLink: link !== undefined, realPath } }
  })
  on('prompt.suggest', ($, e) => {
    seen.suggested.push(e.text)
    // A failing core makes the plugin's $.prompt.suggest reject.
    if (world.suggest === 'reject') throw new Error('no prompt box')
    const isOwn = e.origin.kind === 'plugin' && e.origin.name === 'archcore'
    const isHidden =
      world.suggest === 'hidden' || (world.suggest === 'hide-own' && isOwn) || (world.suggest === 'hidden-once' && declined++ === 0)
    return { isShown: !isHidden }
  })
  on('ui.render', { component: 'AbovePrompt' }, ($, e) => $.ui.resolve(e).Text({ children: ['engine band'] }))
  on('mcp.call', async ($, e) => {
    if (e.server === 'watch') {
      seen.skips.push(e.args)
      return { value: { isError: false, content: [{ type: 'text', text: '{}' }] } }
    }
    seen.calls.push({ server: e.server, tool: e.tool, args: { ...e.args } })
    if (offline > 0) {
      offline--
      return { deny: `no connected MCP tool "${e.tool}" on a server named "${e.server}"` }
    }
    if (!(world.servers ?? [PLUGIN]).includes(e.server)) return { deny: `no server ${e.server}` }
    if (world.gate) await world.gate
    if (world.mcp === 'throw') return { deny: 'boom' }
    const body = answer(world, e.tool, e.args)
    if (world.mcp === 'no-results') delete body.results
    const text = world.mcp === 'garbage' ? 'not json' : world.mcp === 'array' ? '[]' : JSON.stringify(body)
    const blocks = [{ type: 'text', text }]
    if (world.mcp === 'extra-block') blocks.unshift({ type: 'resource', text: 'not the answer' })
    return { value: { isError: world.mcp === 'error', content: blocks } } as never
  })
  on('tool.call', ($, e) => {
    const tool = String(e.tool)
    if (world.deny?.includes(tool)) return { deny: 'blocked' }
    // A failed ExitPlanMode still carries the plan the person rejected.
    const failed = tool === 'ExitPlanMode' ? { plan: world.plan ?? null, isAgent: false } : 'failed'
    if (world.fail?.includes(tool)) return { result: failed, text: 'failed', isError: true }
    if (tool.endsWith('__search_documents')) {
      const text = world.searchFor?.(e as unknown as Record<string, unknown>) ?? world.searchAnswer ?? EMPTY_SEARCH
      return { result: 'ok', text: typeof text === 'string' ? text : JSON.stringify(text) }
    }
    if (tool === 'ExitPlanMode') return { result: { plan: world.plan ?? null, isAgent: false }, text: 'ok' }
    return { result: 'ok', text: '{}' }
  })
  return seen
}

// MCP tool names depend on the servers connected at the last reload, so they are typed loosely.
const mcpOn = (server: string, tool: string, args: Record<string, unknown> = {}) =>
  ({ tool: `mcp__${server}__${tool}`, ...args }) as unknown as ToolCallArgs
const READ = mcpOn(PLUGIN, 'get_document', { path: '.archcore/a.spec.md' })
const DRAFT = mcpOn(PLUGIN, 'create_document', { type: 'adr', filename: 'x' })
const search = (args: Record<string, unknown>) => mcpOn(PLUGIN, 'search_documents', args)
const editAt = (path: string) => ({ tool: 'Edit', file_path: path, old_string: 'a', new_string: 'b' }) as const
const edit = (rel: string) => editAt(`${ROOT}/${rel}`)
const write = (rel: string) => ({ tool: 'Write', file_path: `${ROOT}/${rel}`, content: 'x' }) as const
const bash = (command: string) => ({ tool: 'Bash', command }) as const
const PUSH = bash('git push -u origin feat')
const EXIT_PLAN = { tool: 'ExitPlanMode' } as const
const planAgent = (subagent_type: string, description: string) =>
  ({ tool: 'Agent', subagent_type, description, prompt: 'p' }) as const

const track =(title: string, content: string, extra: Partial<Doc> = {}): Doc =>
  ({ path: `.archcore/${title}.plan.md`, title, type: 'plan', status: 'draft', content, ...extra })
const block = (gate: string) => `<!-- archcore:track\ntrack: ${gate.split('.')[0]}\ngate: ${gate}\n-->`

const DONE = (turnId = 't1') => ({ answer: 'ok', durationMs: 1, isAborted: false, turnId, reason: 'answer' }) as const
const START = { cwd: ROOT, surface: 'terminal', isInteractive: true } as const
const HEADLESS = { cwd: ROOT, surface: null, isInteractive: false } as const
const NO_READ = (n: number) => `${n} file(s) edited without reading any Archcore document this session.`

async function run($: Engine, text: string, calls: readonly ToolCallArgs[] = [], turnId = 't1') {
  await $.turn.start({ text, turnId })
  for (const call of calls) await $.tool.call(call)
  await $.turn.complete(DONE(turnId))
}

// The session-start lookups run detached on the clock: start, then let them finish.
async function begin($: Engine, clock: MockClock, start: typeof START | typeof HEADLESS = START) {
  await $.session.start(start)
  await clock.settle()
}

type Props = { hasSurvey: boolean; isWorking: boolean; maxRows: number; bodyColumns: number; scroll: { offset: number; bodyRows: number }; view: { agentId?: string } }
const PROPS: Props = { hasSurvey: false, isWorking: false, maxRows: 3, bodyColumns: 160, scroll: { offset: 0, bodyRows: 3 }, view: {} }
const mount = ($: Engine, surface: Surface, props: Partial<Props> = {}) =>
  $.ui.mount({
    plugin: 'archcore', component: 'AbovePrompt', surface,
    viewport: { columns: 160, rows: 40 }, props: { ...PROPS, ...props },
  })
const exact = (text: string) => new RegExp(`^${text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`)

// The band on every surface: the hint with its command, and the band content beneath it.
async function expectBand($: Engine, command: string, reason?: string, prefix = 'Tab →') {
  for (const surface of SURFACES) {
    const ui = await mount($, surface)
    expect(await ui.find({ type: 'Text', text: exact(`${prefix} ${command.trim()}`) })).toBeDefined()
    if (reason !== undefined) expect(await ui.find({ type: 'Text', text: exact(reason) })).toBeDefined()
    expect(await ui.find({ type: 'Button', key: 'hide' })).toBeDefined()
    expect(await ui.find({ type: 'Text', text: 'engine band' })).toBeDefined()
    await ui.unmount()
  }
}

async function expectHint($: Engine, seen: Seen, command: string, reason?: string) {
  await expectBand($, command, reason)
  expect(seen.suggested.at(-1)).toBe(command)
}

async function expectNoBand($: Engine, props: Partial<Props> = {}) {
  for (const surface of SURFACES) {
    const ui = await mount($, surface, props)
    expect(await ui.find({ type: 'Button', key: 'hide' })).toBeUndefined()
    expect(await ui.find({ type: 'Text', text: /^(Tab →|run:) / })).toBeUndefined()
    expect(await ui.find({ type: 'Text', text: 'engine band' })).toBeDefined()
    await ui.unmount()
  }
}

async function expectNoHint($: Engine, seen: Seen) {
  await expectNoBand($)
  expect(seen.suggested).toEqual([])
}

// ---------------- 0: an /archcore:* or superpowers:* run stays silent ----------------

test('0: a typed /archcore:* command keeps the turn silent', async ($, on) => {
  const seen = engine(on)
  await run($, '/archcore:document decision решили: используем cobra', [DRAFT, PUSH])
  await expectNoHint($, seen)
})

test('0: a <command-name> envelope anywhere in the prompt is a command', async ($, on) => {
  const seen = engine(on)
  await run($, '<command-message>archcore:review</command-message>\n<command-name>/archcore:review</command-name>', [DRAFT])
  await expectNoHint($, seen)
})

test('0: a <command-name> envelope of /superpowers:* is a command', async ($, on) => {
  const seen = engine(on)
  await run($, '<command-name>/superpowers:brainstorming</command-name>', [DRAFT])
  await expectNoHint($, seen)
})

test('0: a <command-name> envelope without a slash is a command', async ($, on) => {
  const seen = engine(on)
  await run($, '<command-name>archcore:review</command-name>', [DRAFT])
  await expectNoHint($, seen)
})

test('0: a typed /superpowers:* command keeps the turn silent', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [DRAFT])
  await expectNoHint($, seen)
})

for (const skill of ['archcore:document', 'superpowers:brainstorming']) {
  test(`0: the ${skill} skill running in the turn keeps it silent`, async ($, on) => {
    const seen = engine(on)
    await $.turn.start({ text: 'record this', turnId: 't1' })
    await $.skill.prompt({ skill, text: 'x' })
    await $.tool.call(DRAFT)
    await $.turn.complete(DONE())
    await expectNoHint($, seen)
  })
}

test('0: a skill expanded before its turn starts marks that turn', async ($, on) => {
  const seen = engine(on)
  await $.skill.prompt({ skill: 'archcore:review', text: 'x' })
  await run($, 'review it', [DRAFT])
  await expectNoHint($, seen)
})

test('0: another skill and a mid-sentence mention do not silence the turn', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'what does /archcore:review do', turnId: 't1' })
  await $.skill.prompt({ skill: 'commit', text: 'x' })
  await $.tool.call(DRAFT)
  await $.turn.complete(DONE())
  await expectHint($, seen, '/archcore:review', '1 draft document(s) created; they are not accepted yet.')
})

test('0: a skill that only ends in archcore: does not silence the turn', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'record this', turnId: 't1' })
  await $.skill.prompt({ skill: 'notarchcore:x', text: 'x' })
  await $.tool.call(DRAFT)
  await $.turn.complete(DONE())
  await expectHint($, seen, '/archcore:review')
})

test('0: an archcore skill in one request does not silence the next', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'review it', turnId: 't1' })
  await $.skill.prompt({ skill: 'archcore:review', text: 'x' })
  await $.turn.complete(DONE())
  await run($, 'write it down', [DRAFT], 't2')
  await expectHint($, seen, '/archcore:review', '1 draft document(s) created; they are not accepted yet.')
})

test('0: a continuation of an /archcore:* turn stays silent', async ($, on) => {
  const seen = engine(on)
  await run($, '/archcore:review', [])
  await $.turn.start({ text: '', turnId: 't2' })
  await $.tool.call(DRAFT)
  await $.turn.complete(DONE('t2'))
  await expectNoHint($, seen)
})

// ---------------- 0: a Superpowers flow stays silent until it is left ----------------

test('superpowers: a brainstorming flow stays silent across its turns', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [DRAFT])
  await run($, 'Option B, the band above the prompt', [DRAFT, PUSH], 't2')
  await expectNoHint($, seen)
})

test('superpowers: a superpowers skill the model calls starts the quiet flow', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'design the band', turnId: 't1' })
  await $.skill.prompt({ skill: 'superpowers:brainstorming', text: 'x' })
  await $.turn.complete(DONE())
  await run($, 'Option B', [DRAFT], 't2')
  await expectNoHint($, seen)
})

test('superpowers: another skill the model calls mid-flow keeps it quiet', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [])
  await $.turn.start({ text: 'Option B', turnId: 't2' })
  await $.skill.prompt({ skill: 'commit', text: 'x' })
  await $.tool.call(DRAFT)
  await $.turn.complete(DONE('t2'))
  await expectNoHint($, seen)
})

test('superpowers: an /archcore:* command ends the quiet flow', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [DRAFT])
  await run($, '/archcore:review', [], 't2')
  await run($, 'ship it', [PUSH], 't3')
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

test('superpowers: an archcore skill the model calls ends the quiet flow', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [DRAFT])
  await $.turn.start({ text: 'record it', turnId: 't2' })
  await $.skill.prompt({ skill: 'archcore:document', text: 'x' })
  await $.turn.complete(DONE('t2'))
  await run($, 'ship it', [PUSH], 't3')
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

test('superpowers: another typed slash command ends the quiet flow', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [DRAFT])
  await run($, '/commit', [], 't2')
  await run($, 'ship it', [PUSH], 't3')
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

test('superpowers: a skill the person typed ends the quiet flow before its turn starts', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [DRAFT])
  await $.skill.prompt({ skill: 'commit', text: 'x' })
  await run($, 'Commit the staged files.', [], 't2')
  await run($, 'ship it', [PUSH], 't3')
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

test('superpowers: a prompt that starts with an absolute path is no slash command', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [DRAFT])
  await run($, '/Users/me/x.go fails to build', [DRAFT], 't2')
  await expectNoHint($, seen)
})

test('superpowers: /clear ends the quiet flow', async ($, on) => {
  const seen = engine(on)
  await run($, '/superpowers:brainstorming the band', [DRAFT])
  await $.session.end({ reason: 'clear', sessionId: 's1', resume: { id: 's1' } })
  await run($, 'ship it', [PUSH], 't2')
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

// ---------------- 1: an approved plan ----------------

test('1: an approved plan with no edits after it suggests /archcore:plan <title>', async ($, on) => {
  const seen = engine(on, { plan: '# Add hint band\n1. do x' })
  await run($, 'plan the band', [EXIT_PLAN])
  await expectHint($, seen, '/archcore:plan Add hint band', 'The approved plan is outside .archcore/. Record it?')
})

test('1: the Plan subagent counts as a proposed plan', async ($, on) => {
  const seen = engine(on)
  await run($, 'think it through', [planAgent('Plan', 'Design hint band')])
  await expectHint($, seen, '/archcore:plan Design hint band', 'The proposed plan is outside .archcore/. Record it?')
})

test('1: a Plan subagent with no description falls back to a generic title', async ($, on) => {
  const seen = engine(on)
  await run($, 'think', [planAgent('Plan', '')])
  await expectHint($, seen, '/archcore:plan the plan')
})

test('1: a plan file written after the plan is not a code edit', async ($, on) => {
  const seen = engine(on, { plan: '# Add hint band', files: { 'plan.md': '# Other' } })
  await run($, 'plan the band', [EXIT_PLAN, write('plan.md')])
  await expectHint($, seen, '/archcore:plan Add hint band')
})

test('1: code edits after the plan fall through to the edit hints', async ($, on) => {
  const seen = engine(on, { plan: '# Add hint band' })
  await run($, 'plan the band', [EXIT_PLAN, edit('cli/new/x.go')])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

test('1: a second approved plan after code edits asks to record it again', async ($, on) => {
  const seen = engine(on, { plan: '# One' })
  await run($, 'plan', [EXIT_PLAN, READ, edit('cli/new/x.go'), EXIT_PLAN])
  await expectHint($, seen, '/archcore:plan One')
})

test('1: a second Plan subagent after code edits asks to record it again', async ($, on) => {
  const seen = engine(on)
  await run($, 'plan', [planAgent('Plan', 'One'), READ, edit('cli/new/x.go'), planAgent('Plan', 'Two')])
  await expectHint($, seen, '/archcore:plan Two')
})

test('1: an ExitPlanMode with no plan is no plan', async ($, on) => {
  const seen = engine(on, { plan: null })
  await run($, 'plan the band', [EXIT_PLAN])
  await expectNoHint($, seen)
})

test('1: a rejected ExitPlanMode is no plan', async ($, on) => {
  const seen = engine(on, { plan: '# Add hint band', fail: ['ExitPlanMode'] })
  await run($, 'plan the band', [EXIT_PLAN])
  await expectNoHint($, seen)
})

test('1: a plan with no text line falls back to a generic title', async ($, on) => {
  const seen = engine(on, { plan: '  \n\t\n' })
  await run($, 'plan the band', [EXIT_PLAN])
  await expectHint($, seen, '/archcore:plan the approved plan')
})

test('1: a failed Plan subagent is no plan', async ($, on) => {
  const seen = engine(on, { fail: ['Agent'] })
  await run($, 'think it through', [planAgent('Plan', 'Design hint band')])
  await expectNoHint($, seen)
})

test('1: a non-Plan subagent is no plan', async ($, on) => {
  const seen = engine(on)
  await run($, 'look around', [planAgent('Explore', 'Find x')])
  await expectNoHint($, seen)
})

// ---------------- 3: decision words ----------------

test('3: decision words suggest /archcore:document decision', async ($, on) => {
  const seen = engine(on)
  await run($, 'решили: используем FalkorDB для графа')
  await expectHint($, seen, '/archcore:document decision ', 'This sounds like a decision. Record it?')
})

const DECISIONS = [
  'решили: в CLI используем cobra для всех команд',
  'Мы решили: используем FalkorDB',
  'We decided to use Postgres for sessions',
  'From now on we use cobra for every command',
  'отныне все команды на cobra',
  'We decided to use cobra.',
  "We've decided: cobra for the CLI",
  'We’ve decided on cobra',
  "Ok. We're going with cobra",
  'We’re going with cobra',
  'Fix the build. From now on we pin Go 1.23',
  'From now on: hooks stay small',
  'Договорились — используем cobra',
  'Решили - берём cobra',
  'Мы решили, что берём cobra',
  'Договорились, будем использовать cobra',
  'Решили, берём cobra',
  'Решили, переходим на Go',
  'Договорились, остаёмся на Go',
  'Итак, мы решили: cobra',
  'Ну вот, решили: cobra',
  'Ну вот, мы решили: cobra',
  'отныне используем cobra',
  'С этого момента пишем тесты первыми',
  'теперь всегда используем cobra',
]
for (const prompt of DECISIONS) {
  test(`3: "${prompt}" is a decision`, async ($, on) => {
    const seen = engine(on)
    await run($, prompt)
    await expectHint($, seen, '/archcore:document decision ')
  })
}

const NOT_DECISIONS = [
  // At most two words may open the sentence before "решили"; a longer lead-in is narration.
  'Ну вот и решили: cobra',
  'From now on, answer in English',
  'From now on reply in Russian',
  'from now on you write the tests first',
  'From now on always use English',
  'Отныне отвечай по-русски',
  'С этого момента всегда пиши тесты',
  'Подскажи, какую библиотеку будем использовать.',
  'Не знаю, будем использовать cobra или нет',
  'Покажи, что будем использовать для логов, и объясни',
  'Будем использовать cobra для CLI',
  'Договорились, продолжай',
  'Договорились: продолжай',
  'Мы решили-то давно',
  'Ну вот и всё, решили: cobra',
  'Explain what we do from now on with the cache',
  'Это теперь всегдашняя проблема с кэшем',
  'мы решили проблему с кэшем',
  'Мы не решили, что делать с кэшем',
  'have we decided on X?',
  'Should we use cobra from now on?',
  'Решили: берём cobra?',
  "let's use cobra",
  "Let's switch to cobra",
  'переходим на cobra',
  'what we decided yesterday is unclear',
  'tell me how we decided',
]
// Every word that makes a from-now-on sentence an instruction to the assistant.
for (const word of [
  'you', 'answer', 'reply', 'respond', 'write', 'speak', 'talk', 'use', 'call', 'please', "don't", 'do', 'be',
  'keep', 'ask', 'tell', 'show', 'explain', 'translate', 'format',
]) {
  NOT_DECISIONS.push(`From now on ${word} in short sentences`)
}
NOT_DECISIONS.push(
  'Отныне ты пишешь тесты', 'Отныне вы отвечаете кратко', 'Отныне говори по-русски', 'Отныне говорите по-русски',
  'Отныне ответь кратко', 'Отныне ответьте кратко', 'Отныне пишите тесты', 'Отныне переведи всё', 'Отныне переведите всё',
  'Отныне проверяй тесты', 'Отныне имей в виду лимиты', 'Отныне используй cobra', 'Отныне открой файл', 'Отныне делайте тесты',
)
for (const prompt of NOT_DECISIONS) {
  test(`3: "${prompt}" is no decision`, async ($, on) => {
    const seen = engine(on)
    await run($, prompt)
    await expectNoHint($, seen)
  })
}

// ---------------- 6: push ----------------

test('6: a push suggests /archcore:review', async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

const HEREDOC = "git commit -m \"$(cat <<'EOF'\nfix: x\n\ngit push is the next step\nEOF\n)\""
const PUSHES = [
  'git push',
  'git -C repo push',
  'git -c user.name=x push origin main',
  'git -C repo -c core.hooksPath=/dev/null push',
  'git --no-pager push',
  'git --git-dir .git push',
  'GIT_SSH_COMMAND=x git push',
  'A=1 B=2 git push',
  'cd repo && git push',
  'make test; git push --force-with-lease',
  'git fetch || git push',
  'git status\ngit push',
  'gh pr create --fill',
  'gh pr create --title "Support -n flag"',
  'glab mr create',
  'git push --dry-run && git push',
  'git push && git push --dry-run',
  'git push origin x && ls -n',
  'git push --no-verify',
  'git push origin feat-n',
  `${HEREDOC} && git push`,
  "cat <<'EOF' > notes.md && git push\nbody\nEOF",
  'git --work-tree . push',
  'git --namespace ns push',
  'git --config-env x=Y push',
  '(git push)',
  'echo `git push`',
  "cat > notes.md <<'EOF'\nsome text\nEOF\ngit push",
]
for (const command of PUSHES) {
  test(`6: ${JSON.stringify(command)} is a push`, async ($, on) => {
    const seen = engine(on)
    await run($, 'ship it', [bash(command)])
    await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
  })
}

const NOT_PUSHES = [
  // Another tool's push verb is not a git push, also when git appears elsewhere on the line
  // (only such a line passes the hook's git|gh|glab matcher).
  'docker push registry/img:1',
  'echo git && docker push registry/img:1',
  'git status; helm push chart.tgz oci://registry',
  'git push --dry-run',
  'git push -n origin main',
  'git push --dry-run && git push -n',
  'gh pr create --dry-run',
  'gh pr view',
  'gh issue create',
  'make push',
  'glab issue create',
  'glab mr list',
  'git commit -m "explain git push"',
  'git commit -m "wip; git push later"',
  "git commit -m 'a && git push'",
  HEREDOC,
  "cat > RELEASE.md <<'EOF'\n# Release\ngit push origin main\nEOF",
  'echo git push',
  'git status',
  'git pushx',
]
for (const command of NOT_PUSHES) {
  test(`6: ${JSON.stringify(command)} is no push`, async ($, on) => {
    const seen = engine(on)
    await run($, 'ship it', [bash(command)])
    await expectNoHint($, seen)
  })
}

for (const world of [{ deny: ['Bash'] }, { fail: ['Bash'] }] as const) {
  test(`6: a push the engine ${'deny' in world ? 'denied' : 'failed'} is no push`, async ($, on) => {
    const seen = engine(on, world)
    await run($, 'ship it', [PUSH])
    await expectNoHint($, seen)
  })
}

// Drift, coverage and closeout belong to the engine (engine-runtime-boundary.adr): an edit
// makes no path_ref lookup, whatever the documents cite.
test('6: edits in a cited or uncited directory and a push after them make no lookup', async ($, on) => {
  const seen = engine(on, {
    docs: [
      { path: '.archcore/hook.spec.md', title: 'Hook Runtime', type: 'spec', status: 'accepted', content: 'See @cli/cmd/hook.go.' },
      { path: '.archcore/rollout.plan.md', title: 'Hint band rollout', type: 'plan', status: 'draft', content: 'Covers @cli/band/.' },
    ],
  })
  await run($, 'fix it', [edit('cli/cmd/hook.go')])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
  await run($, 'add x', [READ, edit('cli/new/x.go'), edit('cli/band/x.go')], 't2')
  await expectNoBand($)
  await run($, 'build and push', [edit('cli/band/y.go'), PUSH], 't3')
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
  // No search_documents path_ref call, nor any other.
  expect(seen.calls).toEqual([])
})

// ---------------- 7: created drafts ----------------

test('7: created documents suggest /archcore:review', async ($, on) => {
  const seen = engine(on)
  await run($, 'write it down', [DRAFT, DRAFT])
  await expectHint($, seen, '/archcore:review', '2 draft document(s) created; they are not accepted yet.')
})

test('7: a failed create_document is no draft', async ($, on) => {
  const seen = engine(on, { fail: [`mcp__${PLUGIN}__create_document`] })
  await run($, 'write it down', [DRAFT])
  await expectNoHint($, seen)
})

// ---------------- 8: plan file outside Archcore ----------------

test('8: a plan file outside .archcore suggests /archcore:plan <heading>', async ($, on) => {
  const seen = engine(on, { files: { 'docs/plans/band.md': 'intro\n# Band rollout\n- step' } })
  await run($, 'write a plan file', [write('docs/plans/band.md')])
  await expectHint($, seen, '/archcore:plan Band rollout', 'A plan was written to docs/plans/band.md, outside Archcore.')
})

test('8: a root plan.md counts and a long heading is cut to 80 characters', async ($, on) => {
  const seen = engine(on, { files: { 'plan.md': `# ${'x'.repeat(100)}` } })
  await run($, 'write a plan', [write('plan.md')])
  await expectHint($, seen, `/archcore:plan ${'x'.repeat(80)}`)
})

test('8: a nested plan.md is a plan file', async ($, on) => {
  const seen = engine(on, { files: { 'docs/plan.md': '# Nested' } })
  await run($, 'write a plan', [write('docs/plan.md')])
  await expectHint($, seen, '/archcore:plan Nested')
})

test('8: an unreadable plan file in PLANS/ falls back to its path', async ($, on) => {
  const seen = engine(on)
  await run($, 'write a plan', [write('PLANS/q3.md')])
  await expectHint($, seen, '/archcore:plan PLANS/q3.md')
})

test('8: control characters in a plan file path are cleaned', async ($, on) => {
  const seen = engine(on)
  await run($, 'write a plan', [write('PLANS/q\u00073.md')])
  await expectHint($, seen, '/archcore:plan PLANS/q 3.md', 'A plan was written to PLANS/q 3.md, outside Archcore.')
})

test('8: the first plan file written names the hint', async ($, on) => {
  const seen = engine(on, { files: { 'plan.md': '# First', 'plans/b.md': '# Second' } })
  await run($, 'write plans', [write('plan.md'), write('plans/b.md')])
  await expectHint($, seen, '/archcore:plan First')
})

test('8: a plan file next to code edits leaves the hint to the code', async ($, on) => {
  const seen = engine(on, { files: { 'plan.md': '# Root plan' } })
  await run($, 'plan and code', [write('plan.md'), edit('README.md')])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

test('8: guide.md is no plan file', async ($, on) => {
  const seen = engine(on, { files: { 'docs/guide.md': '# Guide' } })
  await run($, 'write', [write('docs/guide.md')])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

// ---------------- 9: edits with no Archcore read ----------------

test('9: code edits with no Archcore read suggest /archcore:review', async ($, on) => {
  const seen = engine(on)
  await run($, 'fix it', [edit('README.md'), edit('Makefile')])
  await expectHint($, seen, '/archcore:review', NO_READ(2))
})

test('9: a read earlier in the session counts for later edits', async ($, on) => {
  const seen = engine(on)
  await run($, 'read the spec', [READ])
  await run($, 'fix it', [edit('README.md')], 't2')
  await expectNoHint($, seen)
})

test('9: a full-mode search that returns documents counts as a read', async ($, on) => {
  const seen = engine(on, { searchAnswer: { ...EMPTY_SEARCH, results: [{ path: '.archcore/b.spec.md', body: 'x' }] } })
  await run($, 'fix it', [search({ content: 'billing', mode: 'full' }), edit('README.md')])
  await expectNoHint($, seen)
})

for (const [name, args, searchAnswer] of [
  ['a full-mode search that returns nothing', { content: 'billing', mode: 'full', types: ['adr'] }, EMPTY_SEARCH],
  ['a snippets search that returns documents', { content: 'billing' }, { ...EMPTY_SEARCH, results: [{ path: '.archcore/b.spec.md' }] }],
] as const) {
  test(`9: ${name} is no read`, async ($, on) => {
    const seen = engine(on, { searchAnswer })
    await run($, 'fix it', [search(args), edit('README.md')])
    await expectHint($, seen, '/archcore:review', NO_READ(1))
  })
}

test('9: /clear forgets the reads', async ($, on) => {
  const seen = engine(on)
  await run($, 'read the spec', [READ])
  await $.session.end({ reason: 'clear', sessionId: 's1', resume: { id: 's1' } })
  await run($, 'fix it', [edit('README.md')], 't2')
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

test('9: the reason counts every distinct file the turn edited past the 20 it keeps', async ($, on) => {
  const seen = engine(on)
  await run($, 'fix it', Array.from({ length: 25 }, (_, i) => edit(`f${i}.go`)))
  await expectHint($, seen, '/archcore:review', NO_READ(25))
})

test('9: a tool whose name only starts like get_document is no read', async ($, on) => {
  const seen = engine(on)
  await run($, 'fix it', [mcpOn(PLUGIN, 'get_documents_index'), edit('README.md')])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

test('9: a document tool of another MCP server is no Archcore read', async ($, on) => {
  const seen = engine(on)
  await run($, 'fix it', [mcpOn('other', 'get_document', { path: 'x' }), mcpOn('other', 'create_document', { type: 'adr' }), edit('README.md')])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

// ---------------- 10: empty search ----------------

test('10: a filter-free search with no results suggests /archcore:document code <topic>', async ($, on) => {
  const seen = engine(on)
  await run($, 'how does billing work', [search({ content: 'billing' })])
  await expectHint($, seen, '/archcore:document code billing', 'Nothing in Archcore matches "billing".')
})

test('10: the topic is one line of at most 80 characters', async ($, on) => {
  const seen = engine(on)
  await run($, 'look', [search({ content: `billing ${'y'.repeat(90)}\nsecond line` })])
  await expectHint($, seen, `/archcore:document code ${`billing ${'y'.repeat(90)}`.slice(0, 80)}`)
})

test('10: an empty status is no filter', async ($, on) => {
  const seen = engine(on)
  await run($, 'look', [search({ content: 'billing', status: '' })])
  await expectHint($, seen, '/archcore:document code billing')
})

test('10: an empty types list is no filter', async ($, on) => {
  const seen = engine(on)
  await run($, 'look', [search({ content: 'billing', types: [] })])
  await expectHint($, seen, '/archcore:document code billing')
})

test('10: a search with results and then an empty one still gives the empty topic', async ($, on) => {
  const seen = engine(on, { searchFor: args => (args.content === 'billing' ? { ...EMPTY_SEARCH, results: [{ path: '.archcore/b.spec.md' }] } : undefined) })
  await run($, 'look', [search({ content: 'billing' }), search({ content: 'refunds' })])
  await expectHint($, seen, '/archcore:document code refunds')
})

test('10: two empty searches name the first topic', async ($, on) => {
  const seen = engine(on)
  await run($, 'look', [search({ content: 'billing' }), search({ content: 'refunds' })])
  await expectHint($, seen, '/archcore:document code billing')
})

test('10: a multi-word empty search with near_misses [] still gives the topic', async ($, on) => {
  const seen = engine(on, { searchAnswer: { ...EMPTY_SEARCH, near_misses: [] } })
  await run($, 'look', [search({ content: 'billing refunds' })])
  await expectHint($, seen, '/archcore:document code billing refunds')
})

test('10: an empty search of another MCP server is no Archcore gap', async ($, on) => {
  const seen = engine(on)
  await run($, 'look', [mcpOn('other', 'search_documents', { content: 'billing' })])
  await expectNoHint($, seen)
})

const NOT_EMPTY: [string, Record<string, unknown>, unknown?][] = [
  ['a search with results', { content: 'billing' }, { ...EMPTY_SEARCH, results: [{ path: '.archcore/b.spec.md' }] }],
  ['a search with near misses', { content: 'billing' }, { ...EMPTY_SEARCH, near_misses: [{ path: '.archcore/b.spec.md', missing: ['x'] }] }],
  ['an unparsable answer', { content: 'billing' }, 'oops'],
  ['an answer with no results array', { content: 'billing' }, { coverage: { local: 1 } }],
  ['a search with no topic', { match: 'all' }],
  ['a types filter', { content: 'billing', types: ['adr'] }],
  ['a status filter', { content: 'billing', status: 'accepted' }],
  ['a source filter', { content: 'billing', source: 'local' }],
  ['an mtime_after filter', { content: 'billing', mtime_after: '7d' }],
  ['a path_ref filter', { content: 'billing', path_ref: 'cli/' }],
]
for (const [name, args, searchAnswer] of NOT_EMPTY) {
  test(`10: ${name} is no empty search`, async ($, on) => {
    const seen = engine(on, { searchAnswer })
    await run($, 'look', [search(args)])
    await expectNoHint($, seen)
  })
}

// ---------------- priority order ----------------

test('order: plan > decision, drafts and push in one turn', async ($, on) => {
  const seen = engine(on, { plan: '# Add hint band' })
  await run($, 'решили: делаем', [EXIT_PLAN, DRAFT, PUSH])
  await expectHint($, seen, '/archcore:plan Add hint band')
})

test('order: decision > push, drafts and edits with no read', async ($, on) => {
  const seen = engine(on)
  await run($, 'решили: пушим', [edit('README.md'), DRAFT, PUSH])
  await expectHint($, seen, '/archcore:document decision ')
})

test('order: push > drafts and edits with no read', async ($, on) => {
  const seen = engine(on)
  await run($, 'record and ship', [edit('README.md'), DRAFT, PUSH])
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

test('order: drafts > plan file', async ($, on) => {
  const seen = engine(on, { files: { 'plan.md': '# Root plan' } })
  await run($, 'plan', [write('plan.md'), DRAFT])
  await expectHint($, seen, '/archcore:review', '1 draft document(s) created; they are not accepted yet.')
})

test('order: drafts > edits with no read', async ($, on) => {
  const seen = engine(on)
  await run($, 'record and fix', [edit('README.md'), DRAFT])
  await expectHint($, seen, '/archcore:review', '1 draft document(s) created; they are not accepted yet.')
})

test('order: plan file > empty search', async ($, on) => {
  const seen = engine(on, { files: { 'plan.md': '# Root plan' } })
  await run($, 'plan', [search({ content: 'billing' }), write('plan.md')])
  await expectHint($, seen, '/archcore:plan Root plan')
})

test('order: no read > empty search', async ($, on) => {
  const seen = engine(on)
  await run($, 'fix', [search({ content: 'billing' }), edit('README.md')])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

// ---------------- no .archcore/ ----------------

test('no .archcore: a turn gives no hint and makes no lookups', async ($, on) => {
  const seen = engine(on, { hasArchcore: false, plan: '# Add hint band' })
  await run($, 'decided?', [EXIT_PLAN, edit('cli/new/x.go'), DRAFT, PUSH])
  await run($, 'From now on we use cobra', [edit('cli/new/y.go')], 't2')
  await expectNoHint($, seen)
  expect(seen.calls).toEqual([])
})

test('no .archcore: a refused check counts as no .archcore', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { hasArchcore: 'throw' })
  await begin($, clock)
  await run($, 'ship it', [PUSH])
  await expectNoHint($, seen)
  expect(seen.logs.filter(line => line.startsWith('fs.exists failed'))).toHaveLength(1)
})

test('no .archcore: a .archcore created during the session turns the hints on', async ($, on) => {
  const world: { hasArchcore: boolean } = { hasArchcore: false }
  const seen = engine(on, world)
  await run($, 'ship it', [PUSH])
  await expectNoHint($, seen)
  world.hasArchcore = true
  await run($, 'ship it again', [PUSH], 't2')
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

// ---------------- the Archcore server ----------------

// Only the session-start track lookup asks the server.
const STOPPED = track('Hint band', block('sdd.design'))

for (const mode of ['throw', 'error', 'garbage', 'array', 'no-results'] as const) {
  test(`MCP ${mode}: an unknown track lookup gives no resume hint and gives up after the retries`, async ($, on) => {
    const clock = mock.clock(on)
    const seen = engine(on, { mcp: mode, docs: [STOPPED] })
    await begin($, clock)
    for (const wait of [2000, 5000, 10000, 20000]) await clock.advance(wait)
    await expectNoHint($, seen)
    expect(seen.logs).toContain('session start: the Archcore server did not answer; no resume hint')
  })
}

test('the answer is the text block, wherever it stands', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { mcp: 'extra-block', docs: [STOPPED] })
  await begin($, clock)
  await expectHint($, seen, '/archcore:plan Hint band')
})

test('an answer that is JSON but no object is unknown and logged', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { mcp: 'array', servers: [PLUGIN, 'archcore'], docs: [STOPPED] })
  await begin($, clock)
  await expectNoBand($)
  expect(seen.calls.map(call => call.server)).toEqual([PLUGIN, 'archcore'])
  expect(seen.logs).toContain('search_documents on plugin_archcore_archcore answered no JSON object')
})

test('a failing server writes one debug line per distinct reason', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { mcp: 'throw', docs: [STOPPED] })
  await begin($, clock)
  for (const wait of [2000, 5000]) await clock.advance(wait)
  expect(seen.calls).toHaveLength(6)
  expect(seen.logs.filter(line => line.startsWith('search_documents on plugin_archcore_archcore failed'))).toHaveLength(1)
  expect(seen.logs.filter(line => line.startsWith('search_documents on archcore failed'))).toHaveLength(1)
})

test('a missing plugin server falls back to archcore and remembers it', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { servers: ['archcore'], docs: [STOPPED] })
  await begin($, clock)
  await expectHint($, seen, '/archcore:plan Hint band')
  expect(seen.calls.map(call => `${call.server} ${call.tool}`)).toEqual([
    `${PLUGIN} search_documents`, 'archcore search_documents', 'archcore get_document',
  ])
})

test('the server is learned from the tool names the agent calls', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { servers: ['archcore'], docs: [STOPPED] })
  await $.session.start(START)
  await $.tool.call(mcpOn('archcore', 'get_document', { path: '.archcore/a.md' }))
  await clock.settle()
  await expectHint($, seen, '/archcore:plan Hint band')
  expect(seen.calls.map(call => call.server)).toEqual(['archcore', 'archcore'])
})

// ---------------- what counts as an edit ----------------

test('an errored or denied edit is not counted', async ($, on) => {
  const seen = engine(on, { fail: ['Edit'], deny: ['Write'] })
  await run($, 'fix it', [edit('README.md'), write('src/x.go')])
  await expectNoHint($, seen)
})

test('edits outside the root, in .archcore and under docs/superpowers are ignored', async ($, on) => {
  const seen = engine(on)
  await run($, 'tweak', [
    editAt('/elsewhere/x.go'),
    editAt('/repo-other/x.go'),
    edit('.archcore/a.adr.md'),
    write('docs/superpowers/plans/band.md'),
    write('packages/web/docs/superpowers/specs/band.md'),
    edit('packages/web/docs/superpowers/notes.go'),
  ])
  await expectNoHint($, seen)
})

test('a NotebookEdit counts as a code edit', async ($, on) => {
  const seen = engine(on)
  await run($, 'tweak', [{ tool: 'NotebookEdit', notebook_path: `${ROOT}/nb/a.ipynb`, new_source: 'x' }])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

// The plan-file hint prints the project-relative path the mod made of the edit.
const expectPlanFile = ($: Engine, seen: Seen, rel: string) =>
  expectHint($, seen, `/archcore:plan ${rel}`, `A plan was written to ${rel}, outside Archcore.`)

test('Windows paths are normalized against the root', async ($, on) => {
  const seen = engine(on, { root: 'C:\\proj\\' })
  await run($, 'plan', [editAt('C:\\proj\\docs\\plans\\x.md')])
  await expectPlanFile($, seen, 'docs/plans/x.md')
})

test('the drive letter compares in any case', async ($, on) => {
  const seen = engine(on, { root: 'C:\\proj' })
  await run($, 'plan', [editAt('c:/proj/plans/x.md')])
  await expectPlanFile($, seen, 'plans/x.md')
})

test('a .. that lands in .archcore/ is no code edit', async ($, on) => {
  const seen = engine(on)
  await run($, 'tweak', [editAt('/repo/src/../.archcore/a.md')])
  await expectNoHint($, seen)
})

test('dot segments fold before the path is used', async ($, on) => {
  const seen = engine(on)
  await run($, 'plan', [editAt('/repo/cli/../docs/./plans/x.md')])
  await expectPlanFile($, seen, 'docs/plans/x.md')
})

test('a relative path is taken from the session root', async ($, on) => {
  const seen = engine(on)
  await run($, 'plan', [editAt('plans/x.md')])
  await expectPlanFile($, seen, 'plans/x.md')
})

test('a .. that leaves the root is outside the project', async ($, on) => {
  const seen = engine(on)
  await run($, 'tweak', [editAt('/repo/../elsewhere/x.go')])
  await expectNoHint($, seen)
})

test('a file under a symlinked root is matched by where it lands', async ($, on) => {
  const seen = engine(on, { root: '/var/repo', links: { '/var': '/private/var' } })
  await run($, 'plan', [editAt('/private/var/repo/plans/x.md')])
  await expectPlanFile($, seen, 'plans/x.md')
})

test('a realPath with backslashes is normalized before the comparison', async ($, on) => {
  const seen = engine(on, { root: '/var/repo', links: { '/var': 'D:\\vol', '/private/var': 'D:\\vol' } })
  await run($, 'plan', [editAt('/private/var/repo/plans/x.md')])
  await expectPlanFile($, seen, 'plans/x.md')
})

test('a path outside the root that cannot be resolved stays outside', async ($, on) => {
  const seen = engine(on, { missing: ['/elsewhere/x.go'] })
  await run($, 'tweak', [editAt('/elsewhere/x.go')])
  await expectNoHint($, seen)
  expect(seen.logs.filter(line => line.startsWith('fs.stat failed'))).toHaveLength(1)
})

test('a root of / holds every absolute path', async ($, on) => {
  const seen = engine(on, { root: '/' })
  await run($, 'plan', [editAt('/etc/plans/x.md')])
  await expectPlanFile($, seen, 'etc/plans/x.md')
})

test('a re-edited file counts once', async ($, on) => {
  const seen = engine(on)
  await run($, 'fix it', [edit('README.md'), edit('README.md')])
  await expectHint($, seen, '/archcore:review', NO_READ(1))
})

// ---------------- what the mod hooks ----------------

// Matched registrations: a tool call the mod does not read never runs its code, so
// input it cannot read (a Bash call with no command) cannot fail its hook.
test('tool calls the mod does not read never reach its code', async ($, on) => {
  const seen = engine(on)
  await run($, 'look', [
    { tool: 'Bash' } as unknown as ToolCallArgs,
    { tool: 'Bash', command: 42 } as unknown as ToolCallArgs,
    { tool: 'Agent', subagent_type: 'Explore' } as unknown as ToolCallArgs,
  ])
  await expectNoHint($, seen)
})

test('an edit with no path is ignored', async ($, on) => {
  const seen = engine(on)
  await run($, 'tweak', [{ tool: 'Edit' } as unknown as ToolCallArgs, { tool: 'NotebookEdit' } as unknown as ToolCallArgs])
  await expectNoHint($, seen)
})

test('the turn keeps no prompt text and at most 20 edits; the session keeps no edits; shape v4', async ($, on) => {
  engine(on)
  await run($, 'From now on we use cobra', [READ, ...Array.from({ length: 25 }, (_, i) => edit(`a/f${i}.go`))])
  const state = await peek($, 'turn')
  expect(state?.shape).toBe('v4')
  expect(Object.keys(state?.value ?? {})).not.toContain('prompt')
  expect(state?.value?.isDecision).toBe(true)
  expect(state?.value?.edits).toHaveLength(20)
  expect((state?.value?.edits as string[]).at(-1)).toBe('a/f24.go')
  expect(state?.value?.editCount).toBe(25)
  expect(await peek($, 'session')).toEqual({ shape: 'v4', value: { reads: 1, superpowers: false } })
})

// ---------------- lifecycle ----------------

test('every event reaches the engine beneath the mod', async ($, on) => {
  const seen = engine(on)
  await $.session.start(START)
  await $.prompt.submit({ text: 'hi', wait: false, origin: { kind: 'composer' } })
  await $.skill.prompt({ skill: 'commit', text: 'x' })
  await $.turn.start({ text: 'hi', turnId: 't1' })
  await $.turn.complete(DONE())
  await $.session.end({ reason: 'other', sessionId: 's1', resume: { id: 's1' } })
  expect(seen.reached).toEqual(['session.start', 'prompt.submit:hi', 'skill.prompt:commit', 'turn.start:t1', 'turn.complete', 'session.end:other'])
})

test('a subagent turn.complete is not the end of the request', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'ship it', turnId: 't1' })
  await $.tool.call(PUSH)
  await $.turn.complete({ ...DONE(), agentId: 'agent-1' })
  await expectNoHint($, seen)
  await $.turn.complete(DONE())
  await expectHint($, seen, '/archcore:review')
})

test('a turn.complete with another turnId is stale', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'ship it', turnId: 't1' })
  await $.tool.call(PUSH)
  await $.turn.complete(DONE('t0'))
  await expectNoHint($, seen)
})

test('a turn.complete that finishes after the next request starts writes nothing', async ($, on) => {
  const clock = mock.clock(on)
  let release = () => {}
  const seen = engine(on, { hold: new Promise<void>(resolve => { release = resolve }) })
  await $.turn.start({ text: 'add x', turnId: 't1' })
  await $.tool.call(edit('cli/new/x.go'))
  const done = $.turn.complete(DONE())
  await clock.settle()
  await $.turn.start({ text: 'next question', turnId: 't2' })
  release()
  await done
  await expectNoHint($, seen)
  expect(await peek($, 'hint')).toBeNull()
})

test("a slow .archcore check of one request does not replace the next request's hint", async ($, on) => {
  const clock = mock.clock(on)
  let release = () => {}
  const seen = engine(on, { hold: new Promise<void>(resolve => { release = resolve }) })
  await $.turn.start({ text: 'add x', turnId: 't1' })
  await $.tool.call(edit('cli/new/x.go'))
  const done = $.turn.complete(DONE())
  await clock.settle()
  await run($, 'write it down', [DRAFT], 't2')
  release()
  await done
  await expectHint($, seen, '/archcore:review', '1 draft document(s) created; they are not accepted yet.')
})

test('a hint written after the next request started is neither drawn nor suggested', async ($, on) => {
  let release = () => {}
  let reached = () => {}
  const gate = new Promise<void>(resolve => { release = resolve })
  const isReached = new Promise<void>(resolve => { reached = resolve })
  const seen = engine(on)
  on('state.set', async ($, e, next) => {
    if (e.key === 'hint' && (e.value as { value: unknown } | undefined)?.value !== null) {
      reached()
      await gate
    }
    return next(e)
  })
  await $.turn.start({ text: 'ship it', turnId: 't1' })
  await $.tool.call(PUSH)
  const done = $.turn.complete(DONE())
  await isReached
  await $.turn.start({ text: 'next question', turnId: 't2' })
  release()
  await done
  await expectNoHint($, seen)
  expect((await peek($, 'hint'))?.value?.requestId).toBe('t1')
  await $.prompt.suggest({ text: 'engine guess', origin: { kind: 'suggestion' } })
  expect(seen.suggested).toEqual(['engine guess'])
})

test('a /clear while the turn-end checks run drops the late hint', async ($, on) => {
  const clock = mock.clock(on)
  let release = () => {}
  const seen = engine(on, { hold: new Promise<void>(resolve => { release = resolve }) })
  await $.turn.start({ text: 'add x', turnId: 't1' })
  await $.tool.call(edit('cli/new/x.go'))
  const done = $.turn.complete(DONE())
  await clock.settle()
  await $.session.end({ reason: 'clear', sessionId: 's1', resume: { id: 's1' } })
  release()
  await done
  await expectNoHint($, seen)
  expect((await peek($, 'hint'))?.value).toBeNull()
})

for (const reason of ['aborted', 'error'] as const) {
  test(`a turn that ended as ${reason} gives no hint`, async ($, on) => {
    const seen = engine(on)
    await $.turn.start({ text: 'ship it', turnId: 't1' })
    await $.tool.call(PUSH)
    await $.turn.complete({ ...DONE(), reason, isAborted: reason === 'aborted' })
    await expectNoHint($, seen)
  })
}

test('a refused turn gives no hint', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'ship it', turnId: 't1' })
  await $.tool.call(PUSH)
  await $.turn.complete({ answer: '', durationMs: 1, isAborted: false, turnId: 't1', reason: 'refusal', refusal: 'dialog' } as never)
  await expectNoHint($, seen)
})

test('a continuation keeps the request and the hint', async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await $.turn.start({ text: '  ', turnId: 't2' })
  await expectBand($, '/archcore:review')
  await $.tool.call(DRAFT)
  await $.turn.complete(DONE('t2'))
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

const NOT_THE_USER: PromptOrigin[] = [
  { kind: 'task-notification' },
  { kind: 'scheduled-trigger' },
  { kind: 'peer' },
  { kind: 'peer-send-message' },
  { kind: 'projects-relay' },
  { kind: 'channel', server: 'slack' },
  { kind: 'coordinator' },
  { kind: 'observer' },
  { kind: 'observer-activity' },
  { kind: 'auto-continuation' },
  { kind: 'plugin', name: 'relay' },
]
for (const origin of NOT_THE_USER) {
  test(`a ${origin.kind} turn keeps the hint and the request`, async ($, on) => {
    const seen = engine(on)
    await run($, 'ship it', [PUSH])
    await $.prompt.submit({ text: 'news', wait: false, origin })
    await $.turn.start({ text: 'news', turnId: 't2' })
    await expectBand($, '/archcore:review', 'The branch went out for review.')
    await $.turn.complete(DONE('t2'))
    await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
  })
}

const THE_USER: PromptOrigin[] = [
  { kind: 'composer' },
  { kind: 'bridge' },
  { kind: 'sdk' },
  { kind: 'slack-ping' },
  { kind: 'unclassified' },
  { kind: 'plugin', name: 'relay', asUser: true },
]
for (const origin of THE_USER) {
  test(`a ${origin.kind}${'asUser' in origin ? ' asUser' : ''} prompt after a notification starts a new request`, async ($, on) => {
    const seen = engine(on)
    await $.prompt.submit({ text: 'task done', wait: false, origin: { kind: 'task-notification' } })
    await run($, 'task done', [PUSH])
    await $.prompt.submit({ text: 'next', wait: false, origin })
    await $.turn.start({ text: 'next', turnId: 't2' })
    await expectNoBand($)
    await $.turn.complete(DONE('t2'))
    expect(seen.suggested).toEqual(['/archcore:review'])
  })
}

test('a user prompt with the text of an earlier notification is a new request', async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await $.prompt.submit({ text: 'done', wait: false, origin: { kind: 'task-notification' } })
  await $.turn.start({ text: 'done', turnId: 't2' })
  await $.turn.complete(DONE('t2'))
  await $.prompt.submit({ text: 'done', wait: false, origin: { kind: 'composer' } })
  await $.turn.start({ text: 'done', turnId: 't3' })
  await expectNoBand($)
  expect(seen.suggested).toEqual(['/archcore:review', '/archcore:review'])
})

test('a notification is matched to its turn by the trimmed text', async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await $.prompt.submit({ text: '  news \n', wait: false, origin: { kind: 'task-notification' } })
  await $.turn.start({ text: 'news', turnId: 't2' })
  await expectBand($, '/archcore:review', 'The branch went out for review.')
  expect(seen.suggested).toEqual(['/archcore:review'])
})

test('a user prompt queued behind a plugin prompt starts a new request', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'ship it', turnId: 't1' })
  await $.tool.call(PUSH)
  await $.prompt.submit({ text: 'background news', wait: false, origin: { kind: 'plugin', name: 'other' } })
  await $.turn.complete(DONE())
  await $.prompt.submit({ text: 'what is 2+2', wait: true, origin: { kind: 'composer' }, turnId: 't1' })
  await $.turn.start({ text: 'what is 2+2', turnId: 't2' })
  await expectNoBand($)
  await $.tool.call(DRAFT)
  await $.turn.complete(DONE('t2'))
  await $.turn.start({ text: 'background news', turnId: 't3' })
  await expectBand($, '/archcore:review', '1 draft document(s) created; they are not accepted yet.')
  await $.turn.complete(DONE('t3'))
  expect(seen.suggested).toEqual(['/archcore:review', '/archcore:review', '/archcore:review'])
})

test('a turn with no prompt.submit after a notification turn is a new request', async ($, on) => {
  engine(on)
  await run($, 'ship it', [PUSH])
  await $.prompt.submit({ text: 'task done', wait: false, origin: { kind: 'task-notification' } })
  await run($, 'task done', [], 't2')
  await $.turn.start({ text: '<command-name>/compact</command-name>', turnId: 't3' })
  await expectNoBand($)
})

test('a prompt delivered into a running turn does not mark the next one', async ($, on) => {
  engine(on)
  await run($, 'ship it', [PUSH])
  await $.prompt.submit({ text: 'fyi', wait: false, origin: { kind: 'peer' }, turnId: 't1' })
  await $.turn.start({ text: 'next', turnId: 't2' })
  await expectNoBand($)
})

test('a new request clears the band and the previous request', async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await $.turn.start({ text: 'what is this', turnId: 't2' })
  await expectNoBand($)
  await $.turn.complete(DONE('t2'))
  await expectNoBand($)
  expect(seen.suggested).toEqual(['/archcore:review'])
})

test('/clear resets the request and the hint', async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
  await $.session.end({ reason: 'clear', sessionId: 's1', resume: { id: 's1' } })
  await expectNoBand($)
  expect((await peek($, 'turn'))?.value?.requestId).toBe('')
  expect((await peek($, 'hint'))?.value).toBeNull()
  await run($, 'push again', [PUSH], 't2')
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

test('/clear drops the session-start hint', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { hasArchcore: false })
  await begin($, clock)
  await expectHint($, seen, '/archcore:init')
  await $.session.end({ reason: 'clear', sessionId: 's1', resume: { id: 's1' } })
  await expectNoBand($)
})

test('/clear forgets a notification that never started a turn', async ($, on) => {
  engine(on)
  await $.prompt.submit({ text: 'done', wait: false, origin: { kind: 'task-notification' } })
  await $.session.end({ reason: 'clear', sessionId: 's1', resume: { id: 's1' } })
  await run($, 'ship it', [PUSH])
  await $.prompt.submit({ text: 'done', wait: false, origin: { kind: 'composer' } })
  await $.turn.start({ text: 'done', turnId: 't2' })
  await expectNoBand($)
})

test('/clear forgets an /archcore:* skill whose turn never started', async ($, on) => {
  const seen = engine(on)
  await $.skill.prompt({ skill: 'archcore:review', text: 'x' })
  await $.session.end({ reason: 'clear', sessionId: 's1', resume: { id: 's1' } })
  await run($, 'ship it', [PUSH])
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

test('the completion of a turn from before /clear is ignored', async ($, on) => {
  const seen = engine(on)
  await $.turn.start({ text: 'ship it', turnId: 't1' })
  await $.tool.call(PUSH)
  await $.session.end({ reason: 'clear', sessionId: 's1', resume: { id: 's1' } })
  await $.turn.complete(DONE())
  await expectNoHint($, seen)
})

test('a session end other than /clear keeps the hint', async ($, on) => {
  engine(on)
  await run($, 'ship it', [PUSH])
  await $.session.end({ reason: 'other', sessionId: 's1', resume: { id: 's1' } })
  await expectBand($, '/archcore:review')
})

test('a turn with nothing drawing gives no hint', async ($, on) => {
  const seen = engine(on, { surfaces: [] })
  await run($, 'fix it', [edit('cli/new/x.go'), PUSH])
  await expectNoHint($, seen)
})

// ---------------- the commands and the debug log ----------------

test('no hint when the Archcore commands are absent', async ($, on) => {
  const seen = engine(on, { commands: ['commit'] })
  await run($, 'ship it', [PUSH])
  await expectNoHint($, seen)
  expect(seen.logs).toEqual(['/archcore:review is not available; no hint'])
})

test('a hint needs its own command, not just any Archcore command', async ($, on) => {
  const seen = engine(on, { commands: ['archcore:review'], plan: '# Add hint band' })
  await run($, 'plan the band', [EXIT_PLAN])
  await expectNoHint($, seen)
})

test('no hint when the command list cannot be read', async ($, on) => {
  const seen = engine(on, { commands: 'throw' })
  await run($, 'ship it', [PUSH])
  await run($, 'ship it again', [PUSH], 't2')
  await expectNoHint($, seen)
  expect(seen.logs.filter(line => line.startsWith('command.list failed'))).toHaveLength(1)
})

// ---------------- the band ----------------

test('Hide removes the hint on every surface', async ($, on) => {
  engine(on)
  for (const [i, surface] of SURFACES.entries()) {
    await run($, 'ship it', [PUSH], `t${i}`)
    const ui = await mount($, surface)
    await ui.press({ key: 'hide' })
    expect(await ui.find({ type: 'Button', key: 'hide' })).toBeUndefined()
    expect(await ui.find({ type: 'Text', text: 'engine band' })).toBeDefined()
    await ui.unmount()
  }
  await expectNoBand($)
})

for (const props of [{ hasSurvey: true }, { isWorking: true }, { view: { agentId: 'agent-1' } }]) {
  test(`${JSON.stringify(props)} hides the band`, async ($, on) => {
    engine(on)
    await run($, 'ship it', [PUSH])
    await expectNoBand($, props)
    await expectBand($, '/archcore:review')
  })
}

test('the band puts Hide first and cuts its lines at the end in 60 columns', async ($, on) => {
  engine(on)
  await run($, 'ship it', [PUSH])
  for (const surface of SURFACES) {
    const ui = await mount($, surface, { bodyColumns: 60 })
    const all = await ui.findAll({})
    const at = (type: string, text?: string) => all.findIndex(one => one.type === type && (text === undefined || one.text === text))
    const [hide, reason, command] = [at('Button'), at('Text', 'The branch went out for review.'), at('Text', 'Tab → /archcore:review')]
    expect(hide).toBeGreaterThan(-1)
    expect(reason).toBeGreaterThan(hide)
    expect(command).toBeGreaterThan(reason)
    expect(all[reason]?.props.wrap).toBe('truncate-end')
    expect(all[command]?.props.wrap).toBe('truncate-end')
    await ui.unmount()
  }
})

test('control characters never reach the band or the box', async ($, on) => {
  const seen = engine(on, { plan: '# Add\u0007hint\u001b[31m band\u009b\t now' })
  await run($, 'plan the band', [EXIT_PLAN])
  await expectHint($, seen, '/archcore:plan Add hint [31m band now')
  await run($, 'look', [search({ content: 'bill\u0000ing\u0085x' })], 't2')
  await expectHint($, seen, '/archcore:document code bill ing x', 'Nothing in Archcore matches "bill ing x".')
})

test('a suggestion the box declined at turn end is tried once more when idle', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { suggest: 'hidden-once' })
  await run($, 'ship it', [PUSH])
  await expectBand($, '/archcore:review', 'The branch went out for review.', 'run:')
  await clock.settle()
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
  expect(seen.suggested).toEqual(['/archcore:review', '/archcore:review'])
})

test('a suggestion the box declines twice shows run: and is not tried a third time', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { suggest: 'hidden' })
  await run($, 'ship it', [PUSH])
  await clock.settle()
  await clock.advance(1000)
  await expectBand($, '/archcore:review', 'The branch went out for review.', 'run:')
  expect(seen.suggested).toEqual(['/archcore:review', '/archcore:review'])
})

test('the retry is dropped when the next request started first', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { suggest: 'hidden' })
  await run($, 'ship it', [PUSH])
  await $.turn.start({ text: 'next', turnId: 't2' })
  await clock.settle()
  expect(seen.suggested).toEqual(['/archcore:review'])
})

test('a rejected suggestion leaves run: in the band', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { suggest: 'reject' })
  await run($, 'ship it', [PUSH])
  await expectBand($, '/archcore:review', undefined, 'run:')
  // The idle retry is rejected too and is not tried a third time.
  await clock.settle()
  await clock.advance(1000)
  await expectBand($, '/archcore:review', undefined, 'run:')
  expect(seen.suggested).toEqual(['/archcore:review', '/archcore:review'])
  // note() logs a repeated reason once.
  expect(seen.logs.filter(line => line.startsWith('prompt.suggest failed'))).toHaveLength(1)
})

test("the engine's own suggestion yields to the hint and turns run: into Tab →", async ($, on) => {
  const seen = engine(on, { suggest: 'hide-own' })
  await run($, 'from now on we use cobra')
  await expectBand($, '/archcore:document decision ', undefined, 'run:')
  await $.prompt.suggest({ text: 'some engine guess', origin: { kind: 'suggestion' } })
  expect(seen.suggested).toEqual(['/archcore:document decision ', '/archcore:document decision '])
  await expectBand($, '/archcore:document decision ')
})

test("without a hint the engine's own suggestion passes", async ($, on) => {
  const seen = engine(on)
  await $.prompt.suggest({ text: 'some engine guess', origin: { kind: 'suggestion' } })
  expect(seen.suggested).toEqual(['some engine guess'])
})

test("the engine's suggestion passes once the next request started", async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await $.turn.start({ text: 'what now', turnId: 't2' })
  await $.prompt.suggest({ text: 'some engine guess', origin: { kind: 'suggestion' } })
  expect(seen.suggested).toEqual(['/archcore:review', 'some engine guess'])
})

test("another plugin's suggestion passes even with a hint", async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await $.prompt.suggest({ text: 'other text', origin: { kind: 'plugin', name: 'other' } })
  expect(seen.suggested).toEqual(['/archcore:review', 'other text'])
})

// ---------------- session start ----------------

test('session start without .archcore suggests /archcore:init', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { hasArchcore: false })
  await begin($, clock)
  await expectHint($, seen, '/archcore:init', 'This repository has no .archcore/.')
  expect(seen.calls).toEqual([])
})

test('session start retries the track lookup until the Archcore server connects', async ($, on) => {
  const clock = mock.clock(on)
  const doc = track('Hint band', `# x\n${block('sdd.design')}`)
  // Both server names fail on the first try and once more: two rounds find nothing connected.
  const seen = engine(on, { docs: [doc], offlineCalls: 4 })
  await begin($, clock)
  await expectNoBand($)
  await clock.advance(2000)
  await expectNoBand($)
  await clock.advance(5000)
  await expectHint($, seen, '/archcore:plan Hint band', 'Draft "Hint band" stopped at the design step.')
  expect(seen.logs.some(line => line.includes('no connected MCP tool'))).toBe(true)
})

test('session start gives up after the last retry and stays silent', async ($, on) => {
  const clock = mock.clock(on)
  const doc = track('Hint band', `# x\n${block('sdd.design')}`)
  const seen = engine(on, { docs: [doc], offlineCalls: 1000 })
  await begin($, clock)
  for (const wait of [2000, 5000, 10000, 20000, 60000]) await clock.advance(wait)
  await expectNoBand($)
  // One first try and four retries, each over both server names.
  expect(seen.calls.filter(call => call.tool === 'search_documents')).toHaveLength(10)
  expect(seen.logs).toContain('session start: the Archcore server did not answer; no resume hint')
})

test('session start with no stopped draft makes one lookup and no retry', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on)
  await begin($, clock)
  for (const wait of [2000, 5000, 10000, 20000]) await clock.advance(wait)
  await expectNoHint($, seen)
  expect(seen.calls.filter(call => call.tool === 'search_documents')).toHaveLength(1)
  expect(seen.logs).toEqual([])
})

test('a session-start retry stops once the first request starts', async ($, on) => {
  const clock = mock.clock(on)
  const doc = track('Hint band', `# x\n${block('sdd.design')}`)
  const seen = engine(on, { docs: [doc], offlineCalls: 2 })
  await begin($, clock)
  await $.turn.start({ text: 'hello', turnId: 't1' })
  await clock.advance(2000)
  expect(seen.calls.filter(call => call.tool === 'search_documents')).toHaveLength(2)
})

test('session start with a local draft stopped at a gate suggests resuming it', async ($, on) => {
  const clock = mock.clock(on)
  const doc = track('Hint band', `# x\n${block('sdd.design')}`)
  const seen = engine(on, { docs: [doc] })
  await begin($, clock)
  await expectHint($, seen, '/archcore:plan Hint band', 'Draft "Hint band" stopped at the design step.')
  expect(seen.calls).toEqual([
    { server: PLUGIN, tool: 'search_documents', args: { content: '<!-- archcore:track', match: 'exact', status: 'draft', source: 'local', limit: 10 } },
    { server: PLUGIN, tool: 'get_document', args: { path: doc.path } },
  ])
})

test('session start names a hyphenated stage in words', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { docs: [track('Hint band', block('sdd.split-tasks'))] })
  await begin($, clock)
  await expectHint($, seen, '/archcore:plan Hint band', 'Draft "Hint band" stopped at the split tasks step.')
})

const RESUMES: [string, string][] = [
  ['sdd.design', '/archcore:plan Hint band'],
  ['requirements-cascade.urd', '/archcore:plan Hint band'],
  ['research.gather', '/archcore:plan Hint band'],
  ['decision.adr', '/archcore:document decision Hint band'],
  ['describe.draft', '/archcore:document code Hint band'],
  ['actualize.verdict', '/archcore:review'],
  ['closeout.verify', '/archcore:review'],
  ['experience.offer', '/archcore:review'],
  ['import.triage', '/archcore:init import'],
]
for (const [gate, command] of RESUMES) {
  test(`session start resumes a ${gate} draft with ${command}`, async ($, on) => {
    const clock = mock.clock(on)
    const seen = engine(on, { docs: [track('Hint band', block(gate))] })
    await begin($, clock)
    await expectHint($, seen, command, `Draft "Hint band" stopped at the ${gate.split('.')[1]} step.`)
  })
}

test('session start resumes a stopped evidence draft with /archcore:document research', async ($, on) => {
  const clock = mock.clock(on)
  const evidence = track('Vendor report', block('research.gather'), { path: '.archcore/vendor-report.evidence.md', type: 'evidence' })
  const seen = engine(on, { docs: [evidence] })
  await begin($, clock)
  await expectHint($, seen, '/archcore:document research Vendor report', 'Draft "Vendor report" stopped at the gather step.')
})

test('session start prefers a stopped plan over another stopped draft', async ($, on) => {
  const clock = mock.clock(on)
  const spec = track('Spec A', block('describe.draft'), { path: '.archcore/a.spec.md', type: 'spec' })
  const plan = track('Plan B', block('sdd.design'))
  const seen = engine(on, { docs: [spec, plan] })
  await begin($, clock)
  await expectHint($, seen, '/archcore:plan Plan B')
  expect(seen.calls.filter(call => call.tool === 'get_document').map(call => call.args.path)).toEqual([plan.path])
})

const NO_TRACK: [string, Doc][] = [
  ['the gate contract placeholder', track('Gate contract', '<!-- archcore:track\ngate: <track>.<stage>\n-->')],
  ['a gate line after the block', track('Loose gate', '<!-- archcore:track\ntrack: sdd\n-->\ngate: sdd.design')],
  ['a track block inside fenced code', track('Contract', `intro\n\`\`\`markdown\n${block('sdd.design')}\n\`\`\`\n`)],
  ['a track block inside a tilde fence', track('Contract', `~~~\n${block('sdd.design')}\n~~~`)],
  ['an unknown track', track('Other', block('other.step'))],
  ['a gate line between two blocks', track('Between', `<!-- archcore:track\ntrack: sdd\n-->\ngate: sdd.design\n<!-- archcore:track\ngate: <track>.<stage>\n-->`)],
  ['an accepted document', track('Done plan', block('sdd.design'), { status: 'accepted' })],
  ['a global draft', track('Org plan', block('sdd.design'), { isGlobal: true })],
]
for (const [name, doc] of NO_TRACK) {
  test(`session start ignores ${name}`, async ($, on) => {
    const clock = mock.clock(on)
    const seen = engine(on, { docs: [doc] })
    await begin($, clock)
    await expectNoHint($, seen)
  })
}

test('session start cuts a long or multi-line draft title to one line of 80 characters', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { docs: [{ ...track('x', block('sdd.design')), title: `${'D'.repeat(90)}\nmore` }] })
  await begin($, clock)
  await expectHint($, seen, `/archcore:plan ${'D'.repeat(80)}`)
})

test('session start finds the real gate after a fenced example', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { docs: [track('Hint band', `\`\`\`\n${block('sdd.design')}\n\`\`\`\n${block('sdd.tasks')}`)] })
  await begin($, clock)
  await expectHint($, seen, '/archcore:plan Hint band', 'Draft "Hint band" stopped at the tasks step.')
})

test('session start resumes the first draft with a real gate', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, {
    docs: [
      track('Gate contract', '<!-- archcore:track\ngate: <track>.<stage>\n-->'),
      track('Hint band', `<!-- archcore:track\ngate: <track>.<stage>\n-->\n${block('sdd.tasks')}`),
    ],
  })
  await begin($, clock)
  await expectHint($, seen, '/archcore:plan Hint band', 'Draft "Hint band" stopped at the tasks step.')
})

test('session start with a failing server gives no hint', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { mcp: 'throw', docs: [track('Hint band', block('sdd.design'))] })
  await begin($, clock)
  await expectNoHint($, seen)
})

test('session start does not hold the first prompt for its lookups', async ($, on) => {
  const clock = mock.clock(on)
  let release = () => {}
  const seen = engine(on, { docs: [track('Hint band', block('sdd.design'))], gate: new Promise<void>(resolve => { release = resolve }) })
  await $.session.start(START)
  await clock.settle()
  expect(seen.calls).toHaveLength(1)
  release()
  await clock.settle()
  await expectHint($, seen, '/archcore:plan Hint band')
})

test('session start gives no hint when the server outlasts 1500 ms', async ($, on) => {
  const clock = mock.clock(on)
  let release = () => {}
  const seen = engine(on, { docs: [track('Hint band', block('sdd.design'))], gate: new Promise<void>(resolve => { release = resolve }) })
  await begin($, clock)
  await clock.advance(1500)
  await expectNoHint($, seen)
  expect(seen.logs).toContain('search_documents took over 1500 ms')
  release()
})

test('session start takes an answer that lands just inside 1500 ms', async ($, on) => {
  const clock = mock.clock(on)
  let release = () => {}
  const seen = engine(on, { docs: [STOPPED], gate: new Promise<void>(resolve => { release = resolve }) })
  await begin($, clock)
  await clock.advance(1499)
  release()
  await clock.settle()
  await expectHint($, seen, '/archcore:plan Hint band')
  expect(seen.logs).toEqual([])
})

test('a session-start hint that lands after the first request started is dropped', async ($, on) => {
  const clock = mock.clock(on)
  let release = () => {}
  const seen = engine(on, { docs: [track('Hint band', block('sdd.design'))], gate: new Promise<void>(resolve => { release = resolve }) })
  await begin($, clock)
  await $.turn.start({ text: 'hello', turnId: 't1' })
  release()
  await clock.settle()
  await expectNoHint($, seen)
})

test("a slow session-start lookup does not replace the first request's hint", async ($, on) => {
  const clock = mock.clock(on)
  let release = () => {}
  const seen = engine(on, { docs: [track('Hint band', block('sdd.design'))], gate: new Promise<void>(resolve => { release = resolve }) })
  await begin($, clock)
  await run($, 'write it down', [DRAFT])
  release()
  await clock.settle()
  await expectHint($, seen, '/archcore:review', '1 draft document(s) created; they are not accepted yet.')
})

test('a headless start makes no lookups and no suggestion', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { hasArchcore: false, docs: [track('Hint band', block('sdd.design'))] })
  await begin($, clock, HEADLESS)
  await expectNoHint($, seen)
  expect(seen.calls).toEqual([])
})

for (const start of [{ cwd: ROOT, surface: 'terminal', isInteractive: false }, { cwd: ROOT, surface: null, isInteractive: true }] as const) {
  test(`a start with ${JSON.stringify(start)} makes no lookups`, async ($, on) => {
    const clock = mock.clock(on)
    const seen = engine(on, { hasArchcore: false })
    await $.session.start(start)
    await clock.settle()
    await expectNoHint($, seen)
  })
}

test('a host that attaches a surface after a headless start gets turn hints', async ($, on) => {
  const clock = mock.clock(on)
  const world: { surfaces: Surface[] } = { surfaces: [] }
  const seen = engine(on, world)
  await begin($, clock, HEADLESS)
  world.surfaces = ['desktop']
  await run($, 'ship it', [PUSH])
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
})

test('a later session.start (reload, enable, respawn) keeps the live hint', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { docs: [track('Hint band', block('sdd.design'))] })
  await run($, 'ship it', [PUSH])
  await begin($, clock)
  await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
  expect(seen.calls).toEqual([])
})

test('a later session.start after a turn with no hint stays quiet', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { hasArchcore: false })
  await run($, 'hello')
  await begin($, clock)
  await expectNoHint($, seen)
})

test('a second session.start before any turn keeps the start hint without new lookups', async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { docs: [track('Hint band', block('sdd.design'))] })
  await begin($, clock)
  const calls = seen.calls.length
  await begin($, clock)
  await expectHint($, seen, '/archcore:plan Hint band')
  expect(seen.calls).toHaveLength(calls)
})

// ---------------- userConfig next_step_hints ----------------

// The /config row "Next-step hints": false registers no hook at all.
test('next_step_hints false: no start, decision or push hint, no MCP call, no suggestion', { options: { next_step_hints: false } }, async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { docs: [STOPPED] })
  await begin($, clock)
  await expectNoBand($)
  await run($, 'решили: используем cobra')
  await expectNoBand($)
  await run($, 'ship it', [PUSH], 't2')
  await expectNoHint($, seen)
  expect(seen.calls).toEqual([])
  expect(await peek($, 'turn')).toBeNull()
})

test('next_step_hints false: no /archcore:init without .archcore', { options: { next_step_hints: false } }, async ($, on) => {
  const clock = mock.clock(on)
  const seen = engine(on, { hasArchcore: false })
  await begin($, clock)
  await expectNoHint($, seen)
})

test('next_step_hints false: the engine suggestion passes', { options: { next_step_hints: false } }, async ($, on) => {
  const seen = engine(on)
  await run($, 'ship it', [PUSH])
  await $.prompt.suggest({ text: 'some engine guess', origin: { kind: 'suggestion' } })
  expect(seen.suggested).toEqual(['some engine guess'])
})

for (const [name, options] of [['left at its default', {}], ['true', { options: { next_step_hints: true } }]] as const) {
  test(`next_step_hints ${name}: the same flow shows each hint`, options, async ($, on) => {
    const clock = mock.clock(on)
    const seen = engine(on, { docs: [STOPPED] })
    await begin($, clock)
    await expectHint($, seen, '/archcore:plan Hint band', 'Draft "Hint band" stopped at the design step.')
    await run($, 'решили: используем cobra')
    await expectHint($, seen, '/archcore:document decision ', 'This sounds like a decision. Record it?')
    await run($, 'ship it', [PUSH], 't2')
    await expectHint($, seen, '/archcore:review', 'The branch went out for review.')
    expect(seen.calls.map(call => call.tool)).toEqual(['search_documents', 'get_document'])
  })
}

// ---------------- the documents line ----------------

const docsLine = (found: number, read: number) => `documents: ${found} found · ${read} read`
const FOUND = { results: [{ path: '.archcore/a.spec.md' }, { path: '.archcore/b.adr.md' }] }

async function expectDocs($: Engine, text: string | null, props: Partial<Props> = {}) {
  for (const surface of SURFACES) {
    const ui = await mount($, surface, props)
    const line = await ui.find({ type: 'Text', text: /^documents: / })
    if (text === null) expect(line).toBeUndefined()
    else expect(line?.text).toBe(text)
    expect(await ui.find({ type: 'Text', text: 'engine band' })).toBeDefined()
    await ui.unmount()
  }
}

test('docs: a turn with searches and reads shows distinct counts without a hint', async ($, on) => {
  const seen = engine(on, { searchAnswer: FOUND })
  await run($, 'how does x work', [search({ content: 'x' }), search({ content: 'y' }), READ, READ])
  await expectDocs($, docsLine(2, 1))
  await expectNoBand($)
  expect(seen.suggested).toEqual([])
})

test('docs: a full-mode search counts its results as read', async ($, on) => {
  engine(on, { searchAnswer: FOUND })
  await run($, 'how does x work', [search({ content: 'x', mode: 'full' })])
  await expectDocs($, docsLine(2, 2))
})

test('docs: the line sits under the hint', async ($, on) => {
  engine(on, { searchAnswer: FOUND })
  await run($, 'look and ship', [search({ content: 'x' }), PUSH])
  await expectBand($, '/archcore:review')
  await expectDocs($, docsLine(2, 0))
})

test('docs: no Archcore lookup shows no line', async ($, on) => {
  engine(on)
  await run($, 'ship it', [PUSH])
  await expectDocs($, null)
})

test('docs: a new request starts the counts over', async ($, on) => {
  engine(on, { searchAnswer: FOUND })
  await run($, 'look', [search({ content: 'x' }), READ])
  await run($, 'other', [], 't2')
  await expectDocs($, null)
})

test('docs: a failed read is not counted, and a running turn hides the line', async ($, on) => {
  engine(on, { fail: [`mcp__${PLUGIN}__get_document`] })
  await run($, 'look', [READ])
  await expectDocs($, null)
  await run($, 'look again', [search({ content: 'x', mode: 'full' })], 't2')
  await expectDocs($, null, { isWorking: true })
})
