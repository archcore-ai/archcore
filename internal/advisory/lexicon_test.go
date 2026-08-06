package advisory

import "testing"

// TestVaguenessLexicon_IsRegexSafe: the lexicon is an exported var compiled
// into an alternation at package init. One entry carrying a metacharacter used
// to panic there, taking down every archcore command rather than just the hook.
func TestVaguenessLexicon_IsRegexSafe(t *testing.T) {
	t.Parallel()
	re := buildLexiconRe([]string{"appropriate", "n/a", "(various)", "a+b", "back\\slash"})
	if !re.MatchString("this is n/a here") {
		t.Error("a metacharacter entry must match literally")
	}
	if re.MatchString("aab") {
		t.Error("a+b must not be treated as a quantifier")
	}
}
