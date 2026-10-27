// The values the next-step hints mod (hooks/next-step.tsx) keeps in $.state.
// What the current request has done so far; reset on each new user request.
export type Turn = {
  // The turnId of the turn.start that began this request; '' before any. A hint names it.
  requestId: string
  // The id of the last turn.start; a turn.complete with another id is stale.
  turnId: string
  // The request's prompt reads as a decision, judged at turn.start.
  isDecision: boolean
  // Project-relative paths of edited code files, most recent last, the last 20 only:
  // not .archcore/, plan files, or docs/superpowers/.
  edits: string[]
  // Distinct files edited, for the "N file(s)" reason; it counts past the 20 kept.
  editCount: number
  pushed: boolean
  drafts: number
  // The first search_documents topic that matched nothing.
  emptySearch: string | null
  // The request ran an /archcore:* or superpowers:* command or skill.
  archcoreCommand: boolean
  // A plan approved in plan mode, or proposed by the Plan subagent.
  plan: { title: string; isProposed: boolean } | null
  editsAfterPlan: number
  // The first plan file written outside .archcore/ and docs/superpowers/.
  planFile: string | null
  // Distinct document paths the request's searches returned, the first 200 only.
  found: string[]
  // Distinct document paths the request read: get_document and full-mode search results.
  read: string[]
}

// What the conversation has done since it began or since /clear.
export type Session = {
  // Archcore documents read: get_document calls and full-mode searches that returned documents.
  reads: number
  // A superpowers:* skill ran and no other slash command or /archcore:* skill followed.
  superpowers: boolean
}

// A next step proposed after a turn: why, the command, whether the prompt box
// took the command as its Tab suggestion, and the request it belongs to.
export type Hint = { reason: string; command: string; shown: boolean; requestId: string }

declare module 'claude-code' {
  interface PluginState {
    archcore: { turn: Shaped<Turn>; session: Shaped<Session>; hint: Shaped<Hint | null> }
  }
}
