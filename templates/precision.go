// Package templates owns the document model's vocabulary: the type constants,
// their virtual categories, the body template each type is created from, and the
// section contract the post-write precision check measures against.
package templates

// Precision canon.
//
// These are the rules a document is measured against: the vague words that hide
// a missing fact, the sections a type owes its reader, and the notation a
// normative document must use. They live here, beside the templates, because
// both answer the same question — what a document of this type should look like.
//
// Data, not code, and deliberately so. The rules were previously a shell script
// in the plugin and a paragraph in a tool description, and the two drifted. One
// table, read by one engine, cannot.
//
// The governing document is the shared rule concepts/document-prose-canon: it
// assigns every type a prose profile, a line format, and a metric. This file is
// where that assignment reaches a project that installs no plugin, so a change
// there is unlanded until it lands here.

// VaguenessLexiconEN are English words that promise a quality without naming a
// fact, a version, a threshold, or a measurement.
var VaguenessLexiconEN = []string{
	"appropriate", "robust", "scalable", "modern", "various",
	"optimal", "efficient", "flexible", "convenient", "seamless", "streamlined",
}

// VaguenessLexiconRU is the Russian half of the same list. Matched as
// substrings because the words inflect; "неоптимальный" contains "оптимальный"
// and is exactly as unfalsifiable.
var VaguenessLexiconRU = []string{
	"оптимальн", "удобн", "правильн", "надёжн", "надежн",
	"гибк", "современн", "передов", "эффективн", "масштабируем",
}

// VaguenessPhrases are multi-word claims with the same defect.
var VaguenessPhrases = []string{
	"best practices", "as needed", "world class", "world-class",
	"cutting edge", "cutting-edge",
}

// OpenListMarkers end a requirement with "and more of the same" (canon S8). A
// reader cannot satisfy a list whose end is not stated, and an agent cannot
// decide whether its own case is in it.
//
// Both languages in one list: a Russian document grades its requirements with
// the same BCP 14 keywords, so its clauses fail the same way.
var OpenListMarkers = []string{
	"etc.", "and so on", "и т.д.", "и т. д.", "и т.п.", "и т. п.",
}

// AmbiguousAlternatives leave the reader to pick which branch binds (canon S9).
var AmbiguousAlternatives = []string{
	"and/or", "и/или",
}

// SectionRule is one required heading. Aliases carry the historical spellings a
// document may still use, so a rename does not turn every older document into a
// finding.
type SectionRule struct {
	// Name is the canonical heading, used in the message.
	Name string
	// Aliases are additional accepted spellings.
	Aliases []string
}

// The section rules that more than one table or check names. The engine reads a
// section through a rule and never through a literal, so a rename here reaches
// the required-section table, the foreign-section table, and every check at
// once — which is what "data, not code" is worth.
var (
	// SectionContext holds an adr's trigger.
	SectionContext = SectionRule{Name: "Context"}
	// SectionAlternatives holds what an adr compared its decision against.
	SectionAlternatives = SectionRule{Name: "Alternatives Considered", Aliases: []string{"Alternatives"}}
	// SectionEnforcement holds the check a rule owes its reader.
	SectionEnforcement = SectionRule{Name: "Enforcement"}
	// SectionBefore and SectionAfter hold a cpat's two code forms.
	SectionBefore = SectionRule{Name: "Before"}
	SectionAfter  = SectionRule{Name: "After"}
)

// RequiredSections lists the headings each type owes its reader. A type absent
// from this map has no structural requirement.
var RequiredSections = map[DocumentType][]SectionRule{
	TypeADR: {
		SectionContext,
		{Name: "Decision"},
		SectionAlternatives,
		{Name: "Consequences"},
	},
	TypeRule: {
		{Name: "Rule"},
		{Name: "Rationale"},
		SectionEnforcement,
	},
	TypeSpec: {
		{Name: "Purpose"},
		{Name: "Surface", Aliases: []string{"Contract Surface"}},
		{Name: "Normative Behavior"},
		{Name: "Conformance"},
	},
	// "Procedure" is an alias because 4 of the corpus's 16 guides spell the
	// section that way; without it the check reported a missing Steps section on
	// a guide that has one under its other name.
	TypeGuide: {
		{Name: "Prerequisites"},
		{Name: "Steps", Aliases: []string{"Procedure"}},
		{Name: "Verification"},
	},
	TypeRFC: {
		{Name: "Summary"},
		{Name: "Motivation"},
		{Name: "Detailed Design"},
		{Name: "Drawbacks"},
		{Name: "Alternatives"},
	},
	// Prefix matching keeps the canonical headings ("Problem Statement",
	// "Goals and Success Metrics") and their shorter spellings both valid,
	// the same way "Purpose" admits a spec's "Purpose & Scope".
	TypePRD: {
		{Name: "Vision"},
		{Name: "Problem"},
		{Name: "Goals"},
		{Name: "Requirements"},
	},
	TypeDoc: {
		{Name: "Overview"},
	},
	TypeIdea: {
		{Name: "Idea"},
		{Name: "Value"},
		{Name: "Risks and Constraints", Aliases: []string{"Risks"}},
	},
	TypePlan: {
		{Name: "Goal"},
		{Name: "Tasks"},
		{Name: "Acceptance Criteria"},
	},
	// The rnd contract is its ending: an investigation that states no verdict and
	// no next step leaves the reader where it found them.
	TypeRnD: {
		{Name: "Approach"},
		{Name: "Findings"},
		{Name: "Recommendation"},
		{Name: "Next Action"},
	},
	TypeTaskType: {
		{Name: "When to Use", Aliases: []string{"When to use"}},
		{Name: "Steps"},
	},
	TypeCPAT: {
		{Name: "Why", Aliases: []string{"Rationale"}},
		SectionBefore,
		SectionAfter,
		{Name: "Scope"},
	},
	TypeMRD: {
		{Name: "Market Landscape"},
		{Name: "Competitive Analysis"},
		{Name: "Market Needs"},
	},
	TypeBRD: {
		{Name: "Business Objectives"},
		{Name: "Stakeholders"},
		{Name: "Success Metrics and ROI", Aliases: []string{"Success Metrics"}},
	},
	TypeURD: {
		{Name: "User Personas", Aliases: []string{"Personas"}},
		{Name: "User Requirements"},
		{Name: "Acceptance Criteria"},
	},
	// The four ISO 29148 sets are tailored subsets, which §4.4 of the standard
	// allows: the full clauses list 19 to 20 subsections each, and a template
	// that demands all of them is a template nobody fills in.
	TypeBRS: {
		{Name: "Business Purpose and Scope", Aliases: []string{"Purpose and Scope", "Purpose"}},
		{Name: "Mission, Goals and Objectives"},
		{Name: "Business Constraints"},
		{Name: "Success Criteria"},
	},
	TypeStRS: {
		{Name: "Purpose and Scope", Aliases: []string{"Purpose"}},
		{Name: "Stakeholder Classes"},
		{Name: "Stakeholder Requirements"},
		{Name: "Operational Concept", Aliases: []string{"Operational Concept (ConOps)", "ConOps"}},
	},
	TypeSyRS: {
		{Name: "System Purpose and Scope", Aliases: []string{"Purpose and Scope", "Purpose"}},
		{Name: "System Requirements"},
		{Name: "System Interfaces"},
		{Name: "Verification Approach", Aliases: []string{"Verification"}},
	},
	TypeSRS: {
		{Name: "Purpose and Scope", Aliases: []string{"Purpose", "Scope"}},
		{Name: "Software Requirements"},
		{Name: "External Interfaces"},
		{Name: "Verification Matrix", Aliases: []string{"Verification"}},
	},
}

// ForeignSection is a heading whose content another document type owns. Owner
// names that type, so a finding can say where the content belongs.
type ForeignSection struct {
	Section SectionRule
	Owner   DocumentType
}

// ForeignSections lists, per type, the headings that carry another type's
// content. The assignment is the content-kind ownership table recorded in
// prd-spec-plan-content-ownership.adr and carried by the plugin's
// skills/_shared/prd-contract.md. A type absent from this map admits any
// heading.
//
// Only unambiguous headings are listed. A prd names a business constraint
// inside its Problem Statement without owing the reader a Constraints section,
// so "Constraints" is deliberately absent: the finding must mean the content
// moved, never that the author phrased a paragraph differently.
var ForeignSections = map[DocumentType][]ForeignSection{
	TypePRD: {
		{Section: SectionRule{Name: "Surface"}, Owner: TypeSpec},
		{Section: SectionRule{Name: "Normative Behavior"}, Owner: TypeSpec},
		{Section: SectionRule{Name: "Failure Behavior"}, Owner: TypeSpec},
		{Section: SectionRule{Name: "Conformance"}, Owner: TypeSpec},
		{Section: SectionRule{Name: "Solution Overview"}, Owner: TypeSpec},
		{Section: SectionRule{Name: "Technical Considerations"}, Owner: TypeSpec},
		{Section: SectionRule{Name: "Tasks"}, Owner: TypePlan},
		{Section: SectionRule{Name: "Timeline"}, Owner: TypePlan},
		{Section: SectionRule{Name: "Milestones"}, Owner: TypePlan},
		{Section: SectionRule{Name: "Phases"}, Owner: TypePlan},
		{Section: SectionRule{Name: "Acceptance Criteria"}, Owner: TypePlan},
		{Section: SectionAlternatives, Owner: TypeADR},
	},
	// A source document captures where a requirement comes from; the formal ISO
	// structure belongs to the specification layer. Formalization runs one way,
	// sources to specifications, so ISO structure inside a source means the two
	// layers were conflated — and a conflated pair traces in neither direction.
	TypeMRD: sourceForeignSections,
	TypeBRD: sourceForeignSections,
	TypeURD: sourceForeignSections,
}

// sourceForeignSections are the specification-layer headings that a source
// document must not carry. Only unambiguous ones are listed: a source states a
// business rule or a user need without owing anyone ISO notation, so a heading
// that both layers may legitimately use stays off this list.
var sourceForeignSections = []ForeignSection{
	{Section: SectionRule{Name: "Mission, Goals and Objectives"}, Owner: TypeBRS},
	{Section: SectionRule{Name: "Operational Concept", Aliases: []string{"Operational Concept (ConOps)", "ConOps"}}, Owner: TypeStRS},
	{Section: SectionRule{Name: "Stakeholder Requirements"}, Owner: TypeStRS},
	{Section: SectionRule{Name: "System Requirements"}, Owner: TypeSyRS},
	{Section: SectionRule{Name: "Software Requirements"}, Owner: TypeSRS},
	{Section: SectionRule{Name: "Verification Approach"}, Owner: TypeSyRS},
	{Section: SectionRule{Name: "Verification Matrix"}, Owner: TypeSRS},
}

// ArchitectVoiceTypes are the types that argue rather than instruct. A long code
// block in one of them is usually implementation detail that belongs behind an
// @path reference — whereas a rule, a guide, or a cpat needs the literal text,
// because the exact bytes are the artifact.
var ArchitectVoiceTypes = map[DocumentType]bool{
	TypeADR: true, TypeRFC: true, TypeDoc: true,
	TypePRD: true, TypeIdea: true, TypePlan: true,
	TypeMRD: true, TypeBRD: true, TypeURD: true,
	TypeBRS: true, TypeStRS: true, TypeSyRS: true, TypeSRS: true,
}

// ProseProfile names which half of the shared writing profile binds a type. The
// STE half constrains the sentence; the ISO 24495-1 half constrains the
// structure. Both halves apply to every document — the profile names the half a
// check may report on, so an adr is never measured by a step's word cap and a
// guide is never measured for a missing evidence marker.
type ProseProfile string

const (
	// ProfileSTE binds a document whose lines instruct or obligate.
	ProfileSTE ProseProfile = "STE"
	// ProfileISO binds a document whose lines argue or describe.
	ProfileISO ProseProfile = "ISO"
)

// ProseProfiles is the per-type assignment recorded in the shared rule
// concepts/document-prose-canon. Every type appears: a type with no profile
// would fall through every check that reads this table, and
// TestProseProfiles_Completeness pins that.
var ProseProfiles = map[DocumentType]ProseProfile{
	TypeSpec: ProfileSTE, TypeRule: ProfileSTE, TypeGuide: ProfileSTE,
	TypeTaskType: ProfileSTE,
	TypeBRS:      ProfileSTE, TypeStRS: ProfileSTE, TypeSyRS: ProfileSTE, TypeSRS: ProfileSTE,

	TypeADR: ProfileISO, TypeRFC: ProfileISO, TypeDoc: ProfileISO,
	TypePRD: ProfileISO, TypePlan: ProfileISO, TypeIdea: ProfileISO,
	TypeRnD: ProfileISO, TypeCPAT: ProfileISO,
	TypeMRD: ProfileISO, TypeBRD: ProfileISO, TypeURD: ProfileISO,
}

// ClauseSections lists, per type, the headings whose numbered items are graded
// requirements. A numbered item outside them is prose — an adr enumerates its
// alternatives without owing them a modal, and grading those would make the
// finding mean nothing.
//
// The four ISO 29148 types are deliberately absent, and this is not an
// oversight to repair. They carry their requirements as identified table rows
// (`| SyR-F-001 | ... | P0 | Test |`), which is what the standard asks for and
// what their templates emit; a numbered-clause check finds nothing there and
// would only claim a coverage the engine does not have. Adding them back means
// first moving those templates off tables — and losing the requirement IDs that
// the traceability sections point at.
var ClauseSections = map[DocumentType][]SectionRule{
	TypeSpec: {
		{Name: "Normative Behavior"},
		{Name: "Failure Behavior", Aliases: []string{"Error Handling"}},
	},
	TypeRule: {
		{Name: "Rule"},
	},
}

// StepSections lists, per type, the headings whose numbered items are procedure
// steps. A step carries one imperative action and no modal: it tells the reader
// what to do now, and an obligation is not a thing anyone can do.
//
// Membership here, not the prose profile, is what gives a type the step checks.
// A plan records claims and still holds a numbered task list, and the two facts
// stop contradicting each other once each table grades only its own items.
var StepSections = map[DocumentType][]SectionRule{
	TypeGuide:    {{Name: "Steps", Aliases: []string{"Procedure"}}},
	TypeTaskType: {{Name: "Steps"}},
	TypePlan:     {{Name: "Tasks"}},
}

// Precision thresholds.
const (
	// MinBodyChars is the floor below which a body is a placeholder, not a
	// document.
	MinBodyChars = 200
	// MaxClauseWords caps a graded requirement and MaxStepWords a procedure
	// step, both from canon S2. Past the cap a line usually carries a second
	// obligation or buries the actor behind subordinate clauses. The metric is
	// the half of the profile that prose alone never held: while it existed only
	// as "one idea per sentence", 22% of rule clauses, 29% of guide steps, and
	// 59% of plan tasks ran over. Reproduce with @scripts/prose-conformance.py,
	// which measures the same predicate the engine reports on.
	MaxClauseWords = 25
	MaxStepWords   = 20
	// MaxSpecBodyLines caps a spec. Past it the document is describing rather
	// than specifying.
	//
	// 120, raised from 80. Two measurements moved it. First, the cap counts every
	// body line, and blank lines plus headings take ~19 of them on the six-section
	// form — a quarter of the old budget went to structure before a single
	// requirement was written. Second, the old number was stricter than the median
	// of the closest comparable corpus (OpenSpec's 36 dogfooded per-capability
	// specs run ~110 lines), and no comparable tool caps by line count at all;
	// they decompose instead. 120 also collapses the flagship exemption that
	// /archcore:init carried for hotspot synthesis: one number, which the engine
	// can actually check, replaces two of which it knew only one.
	//
	// The cap is a proxy and a coarse one — line length across this repository's
	// own specs spans 64 to 187 characters, so 120 lines buys anywhere from 1.5k
	// to 5k tokens. It stays a line count because that is what an author can see
	// while writing. What it cannot see, the split path handles: past the cap the
	// answer is decomposition, not a larger document.
	MaxSpecBodyLines = 120
	// MaxCodeBlockLines is the longest code block an architect-voice document
	// carries before it should reference the source instead.
	MaxCodeBlockLines = 5
)
