package update

// What this file answers is "did this project build this binary?" — the first
// condition of the unattended-update policy, and the only one that cannot be
// recovered from at runtime.

// officialBuild is injected only by this repository's release workflow, as
// -X archcore-cli/internal/update.officialBuild=<value>. Its absence is what
// stops a fork or a repackaged build from replacing itself with this project's
// release — a takeover of a binary this project does not own —
// unattended-update.spec §3. It is separate from the telemetry key on purpose: a
// fork may legitimately inject its own key, and telemetry variables govern
// telemetry only.
//
// It lives in this package rather than in main so no boolean has to be threaded
// through the root command and every test that constructs one. It is a string
// because -X can set nothing else; the value itself carries no meaning, only its
// presence does.
//
// A `go build`, a `go install`, a fork and a CI build are all inert by
// construction, which is the same property telemetry gets from its own ldflag.
var officialBuild string

// isOfficialBuild reports whether this binary carries the release marker.
//
// Any non-empty value counts. The release workflow decides what to put there,
// and a policy that also parsed the value would turn a formatting change in the
// workflow into a silent loss of unattended updates for a whole release.
func isOfficialBuild() bool {
	return officialBuild != ""
}
