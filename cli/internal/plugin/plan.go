package plugin

import "slices"

// Plan is the whole decision layer: evidence in, actions out. It runs no
// command, touches no file and reads no environment. That purity is what lets
// one test prove the update step and `archcore plugin update` produce the same
// plan from the same evidence. A host that no tier addresses yields no action.
//
// The result is ordered by the canonical host order, not by the order evidence
// arrived in, so two callers that collect evidence differently still produce
// comparable plans. Evidence naming a host that ships no plugin is ignored. If
// evidence names one host twice, the first entry wins — a second observation of
// one host is a collector defect, and picking the first keeps the plan
// deterministic instead of letting the later entry silently override.
func Plan(verb Verb, evidence []Evidence) []Action {
	actions := make([]Action, 0, len(evidence))
	for _, host := range hostOrder {
		ev, found := firstEvidence(evidence, host)
		if !found {
			continue
		}
		spec, ok := SpecFor(host)
		if !ok {
			continue
		}
		if action, ok := planHost(verb, spec, ev); ok {
			actions = append(actions, action)
		}
	}
	return actions
}

// firstEvidence returns the first observation of a host.
func firstEvidence(evidence []Evidence, host Host) (Evidence, bool) {
	for _, ev := range evidence {
		if ev.Host == host {
			return ev, true
		}
	}
	return Evidence{}, false
}

// planHost applies one verb's tier table to one host. An unrecognized verb
// addresses nothing, which keeps a future verb silent until its tiers exist.
func planHost(verb Verb, spec HostSpec, ev Evidence) (Action, bool) {
	switch verb {
	case VerbInstall:
		return planInstall(spec, ev)
	case VerbUpdate:
		return planUpdate(spec, ev)
	case VerbRemove:
		return planRemove(spec, ev)
	case VerbStatus:
		return planStatus(spec, ev)
	}
	return Action{}, false
}

// planUpdate implements requirements 5 to 10 of updating-the-plugin.spec.
// The tiers in order: a host with no CLI mechanism prints its UI note on
// registry evidence; a present CLI whose own listing names the plugin runs the
// update; a present CLI whose listing failed, did not parse, or does not name
// the plugin is skipped in silence; an absent CLI over a registry that names the
// plugin prints the exact command; anything else is skipped in silence.
//
// The middle tier is the load-bearing one. It is why a machine that never
// installed the plugin pays no mutating command and sees no output, and it
// reaches that silence from the shape of the evidence rather than from a host's
// error text.
func planUpdate(spec HostSpec, ev Evidence) (Action, bool) {
	if !spec.hasCLI() {
		if ev.RegistryListed {
			return uiNoteAction(spec, VerbUpdate, ev), true
		}
		return Action{}, false
	}
	if ev.CLIPresent {
		if !ev.ListingOK || !ev.Listed {
			return Action{}, false
		}
		action := runAction(spec, VerbUpdate, ev)
		if action.Kind == ActionRun && len(action.Commands) == 0 {
			return Action{}, false
		}
		return action, true
	}
	if ev.RegistryListed {
		return printCommandAction(spec, VerbUpdate, ev), true
	}
	return Action{}, false
}

// planInstall implements requirements 3, 5, and 9 of plugin-delivery.spec
// with failure behavior 1.
//
// Install is not update. It runs only after an explicit host selection, a
// named --agent, or a typed verb, so consent is already given and a missing
// listing is not a reason to refuse: a present CLI whose listing failed still
// installs, and the host's own already-registered tolerance absorbs a repeat.
// The one refusal is the opposite case — a listing that names the plugin makes
// the install a reported no-op, which is what keeps a rerun of `archcore init`
// from nagging or reinstalling.
//
// Two cells the spec leaves open, decided toward a quiet idempotent surface —
// both resolve the same way, because the deciding question is the same one:
//   - An absent CLI over a registry that names the plugin reports it installed
//     rather than printing an install command. Requirement 16 already treats the
//     registry as the presence evidence when the CLI is absent, and printing an
//     install line for an installed plugin would nag.
//   - Cursor over a registry that names the plugin reports it installed too.
//     Requirement 5 states the UI note without a condition, but its point is that
//     Cursor never runs a command, and a report runs none. The Conformance
//     sentence — "a rerun over an installed plugin is a reported no-op" — and the
//     invariant that repeated inits never nag both bind Cursor as much as the
//     hosts with a CLI. Cursor with no registry evidence still prints the note.
func planInstall(spec HostSpec, ev Evidence) (Action, bool) {
	if !spec.hasCLI() {
		if ev.RegistryListed {
			return reportInstalledAction(spec, ev), true
		}
		return uiNoteAction(spec, VerbInstall, ev), true
	}
	if ev.CLIPresent {
		if ev.ListingOK && ev.Listed {
			return reportInstalledAction(spec, ev), true
		}
		return runAction(spec, VerbInstall, ev), true
	}
	if ev.RegistryListed {
		return reportInstalledAction(spec, ev), true
	}
	return printCommandAction(spec, VerbInstall, ev), true
}

// planRemove implements requirement 17 of plugin-delivery.spec with failure
// behavior 5.
//
// Removal is reachable only from a typed `archcore plugin remove`, so consent
// is explicit and a failed listing does not refuse the verb — the same
// asymmetry install has. The evidence that does stop it is proof of absence: a
// listing that ran, parsed, and does not name the plugin means there is nothing
// to remove, and so does an absent CLI over a registry that does not name it.
// Both are silent, because removing nothing is not news.
//
// [decision] The spec does not name this tier split; it follows the package's
// silence bias and requirement 19, which lets a host be skipped for missing
// evidence without failing a direct invocation.
func planRemove(spec HostSpec, ev Evidence) (Action, bool) {
	if !spec.hasCLI() {
		if ev.RegistryListed {
			return uiNoteAction(spec, VerbRemove, ev), true
		}
		return Action{}, false
	}
	if ev.CLIPresent {
		if ev.ListingOK && !ev.Listed {
			return Action{}, false
		}
		return runAction(spec, VerbRemove, ev), true
	}
	if ev.RegistryListed {
		return printCommandAction(spec, VerbRemove, ev), true
	}
	return Action{}, false
}

// planStatus implements requirement 16 of plugin-delivery.spec. Status
// reports every host it was given evidence for, including a host whose evidence
// found nothing: "nothing here" is the report, not a reason to skip. A host the
// evidence slice never names is not reported at all — Plan invents no evidence,
// so the caller decides which hosts a status run covers. The executor words the
// report from the echoed evidence, which is what carries the presence answer and
// the version when the host names one.
func planStatus(spec HostSpec, ev Evidence) (Action, bool) {
	return Action{Host: spec.Host, Kind: ActionReportStatus, Evidence: ev}, true
}

// runAction plans the host's mutating sequence for a verb. A verb with no
// commands on this host cannot run, so it falls back to the UI note.
func runAction(spec HostSpec, verb Verb, ev Evidence) Action {
	cmds := spec.commandsFor(verb)
	if len(cmds) == 0 {
		return uiNoteAction(spec, verb, ev)
	}
	if verb == VerbUpdate {
		cmds = addressInstalls(spec, cmds, ev.Installs)
	}
	return Action{
		Host:          spec.Host,
		Kind:          ActionRun,
		Commands:      cmds,
		MigrateSource: verb == VerbUpdate,
		// Only a Claude Code install merges the autoUpdate marketplace entry, and
		// only after the commands succeed.
		MergeAutoUpdate: verb == VerbInstall && spec.MergeAutoUpdate,
		// The same host's removal takes the entry back. spec.MergeAutoUpdate is
		// the marker for "this host carries the entry"; the verb decides which
		// direction it moves.
		RemoveAutoUpdate: verb == VerbRemove && spec.MergeAutoUpdate,
		Evidence:         ev,
	}
}

// printCommandAction plans the exact command line for the user to run. It is
// the tier for a host whose CLI the CLI cannot reach.
func printCommandAction(spec HostSpec, verb Verb, ev Evidence) Action {
	cmds := spec.commandsFor(verb)
	if len(cmds) == 0 {
		return uiNoteAction(spec, verb, ev)
	}
	return Action{
		Host:     spec.Host,
		Kind:     ActionPrintCommand,
		Commands: cmds,
		Evidence: ev,
	}
}

// uiNoteAction plans the one-line instruction for a host with no CLI mechanism.
func uiNoteAction(spec HostSpec, verb Verb, ev Evidence) Action {
	return Action{
		Host:     spec.Host,
		Kind:     ActionPrintUINote,
		Note:     spec.noteFor(verb),
		Evidence: ev,
	}
}

// reportInstalledAction plans the no-op an install over a present plugin must
// be. It runs no command, which is the whole point.
//
// It still carries the settings merge. An install whose commands succeeded but
// whose settings write failed leaves a machine that reports "already installed"
// on every later run, and the entry conformance requires would then never be
// written by any path. Because the merge skips the write when the file already
// carries the entry, the ordinary rerun stays a true no-op: the flag only heals
// the file that never got one.
func reportInstalledAction(spec HostSpec, ev Evidence) Action {
	return Action{
		Host:            spec.Host,
		Kind:            ActionReportInstalled,
		MergeAutoUpdate: spec.MergeAutoUpdate,
		Evidence:        ev,
	}
}

// maxAddressedInstalls bounds how many installations one host's update
// addresses. It protects the step budget: each installation costs one host
// command of about 2 s, and a machine that installs per project in throwaway
// directories keeps collecting records — updating-the-plugin.spec, Surface.
const maxAddressedInstalls = 32

// addressInstalls repeats the last command of an update sequence once per
// listed installation — updating-the-plugin.spec §19 to §22. The commands before
// it refresh what every installation updates from, so they run once. A host
// that listed installations and can address none of them updates nothing —
// updating-the-plugin.spec §25.
func addressInstalls(spec HostSpec, cmds []Command, installs []Install) []Command {
	if spec.UpdateAddressing == AddressingNone || len(cmds) == 0 || len(installs) == 0 {
		return cmds
	}
	base := cmds[len(cmds)-1]
	seen := make(map[string]bool, len(installs))
	addressed := make([]Command, 0, min(len(installs), maxAddressedInstalls))
	for _, install := range installs {
		c, ok := addressInstall(spec.UpdateAddressing, base, install)
		if !ok || seen[c.String()] {
			continue
		}
		seen[c.String()] = true
		addressed = append(addressed, c)
		if len(addressed) == maxAddressedInstalls {
			break
		}
	}
	if len(addressed) == 0 {
		return nil
	}
	return append(cmds[:len(cmds)-1:len(cmds)-1], addressed...)
}

// addressInstall builds the update command for one installation. The second
// result is false for an installation the step must not touch.
func addressInstall(addressing InstallAddressing, base Command, install Install) (Command, bool) {
	c := base
	c.Args = slices.Clone(base.Args)
	c.ContinueOnFailure = true
	switch addressing {
	case AddressingNone:
		return base, true
	case AddressingName:
		// Only a name the host accepts is passed on: any other token a listing line
		// matched on — a header, a path — would become the argument of a mutating
		// command — updating-the-plugin.spec §26.
		if (install.Name != pluginName && install.Name != PluginID) || len(c.Args) == 0 {
			return Command{}, false
		}
		c.Args[len(c.Args)-1] = install.Name
		return c, true
	case AddressingScope:
		switch install.Scope {
		case "":
			return c, true
		case ScopeUser:
			c.Args = append(c.Args, "--scope", string(ScopeUser))
			return c, true
		case ScopeProject, ScopeLocal:
			// Claude Code updates the first installation of the scope when the
			// working directory matches none, so an unreachable project is skipped
			// rather than run from anywhere else — anthropics/claude-code#90519.
			if !install.ProjectPresent {
				return Command{}, false
			}
			c.Args = append(c.Args, "--scope", string(install.Scope))
			c.Dir = install.ProjectPath
			return c, true
		case ScopeManaged:
			return Command{}, false
		}
	}
	return Command{}, false
}
