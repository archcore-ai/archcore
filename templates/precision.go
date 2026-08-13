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

// SectionRule is one required heading. Aliases carry the historical spellings a
// document may still use, so a rename does not turn every older document into a
// finding.
type SectionRule struct {
	// Name is the canonical heading, used in the message.
	Name string
	// Aliases are additional accepted spellings.
	Aliases []string
}

// RequiredSections lists the headings each type owes its reader. A type absent
// from this map has no structural requirement.
var RequiredSections = map[DocumentType][]SectionRule{
	TypeADR: {
		{Name: "Context"},
		{Name: "Decision"},
		{Name: "Alternatives Considered", Aliases: []string{"Alternatives"}},
		{Name: "Consequences"},
	},
	TypeRule: {
		{Name: "Rule"},
		{Name: "Rationale"},
		{Name: "Enforcement"},
	},
	TypeSpec: {
		{Name: "Purpose"},
		{Name: "Surface", Aliases: []string{"Contract Surface"}},
		{Name: "Normative Behavior"},
		{Name: "Conformance"},
	},
	TypeGuide: {
		{Name: "Steps"},
		{Name: "Verification"},
	},
	TypeRFC: {
		{Name: "Summary"},
		{Name: "Motivation"},
		{Name: "Detailed Design"},
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
		{Section: SectionRule{Name: "Alternatives Considered", Aliases: []string{"Alternatives"}}, Owner: TypeADR},
	},
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

// Precision thresholds.
const (
	// MinBodyChars is the floor below which a body is a placeholder, not a
	// document.
	MinBodyChars = 200
	// MaxSpecBodyLines caps a spec. Past it the document is describing rather
	// than specifying.
	MaxSpecBodyLines = 80
	// MaxCodeBlockLines is the longest code block an architect-voice document
	// carries before it should reference the source instead.
	MaxCodeBlockLines = 5
)
