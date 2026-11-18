package advisory

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"archcore-cli/templates"
)

// maxScannedFileBytes skips generated and vendored blobs: a source file a
// document cites is far smaller, and one huge bundle would dominate the pass.
const maxScannedFileBytes = 1 << 20

// prefixBytes keys the substring index; four bytes are two Cyrillic letters.
const prefixBytes = 4

var (
	codeTokenRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
	// codeCiteTextRe finds "<slug>.<type>" with no file-name character but "/"
	// before it; the caller drops a match that continues as a file name ("Foo.spec.ts").
	codeCiteTextRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9_.-])(` + codeCiteRe + `)`)
	// looseCiteRe finds the prose form "<slug> <type>" ("feed spec §18"); only a
	// slug the corpus holds makes it a citation, so "the plan" stays a phrase.
	looseCiteRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9_./-])([a-z0-9]+(?:-[a-z0-9]+)*) (` + strings.Join(docTypeNames(), "|") + `)\b`)
	// scannedExtensions are the code and interface-text files a claim is checked
	// against: the code-reference list plus the localization formats.
	scannedExtensions = func() map[string]bool {
		set := map[string]bool{".strings": true, ".xcstrings": true, ".plist": true}
		for _, ext := range templates.CodeReferenceExtensions {
			set["."+ext] = true
		}
		delete(set, ".md")
		return set
	}()
)

type codeScan struct {
	identifiers map[string]bool
	substrings  map[string]bool
	// citations maps a "<slug>.<type>" citation to the files that hold it.
	citations map[string][]string
	// looseCitations maps the "<slug> <type>" form, keyed "<slug>.<type>".
	looseCitations map[string][]string
}

// scanCode reads every tracked source and text file outside .archcore/ once and
// reports which wanted identifiers, as whole tokens, and which substrings it
// holds, and which document citations appear where. A Markdown file outside
// .archcore/ is read for citations only: a README naming its contract cites it,
// but its prose proves no identifier exists. A file that cannot be read is
// skipped: the check is advisory and reports what it could read.
func scanCode(baseDir string, tracked []string, wanted map[string][]citedToken, substrings []string) codeScan {
	out := codeScan{identifiers: map[string]bool{}, substrings: map[string]bool{}, citations: map[string][]string{}, looseCitations: map[string][]string{}}
	index := map[string][]string{}
	for _, s := range substrings {
		if len(s) >= prefixBytes {
			index[s[:prefixBytes]] = append(index[s[:prefixBytes]], s)
		}
	}
	for _, rel := range tracked {
		ext := strings.ToLower(path.Ext(rel))
		prose := ext == ".md"
		if strings.HasPrefix(rel, ".archcore/") || !scannedExtensions[ext] && !prose {
			continue
		}
		full := filepath.Join(baseDir, filepath.FromSlash(rel))
		if info, err := os.Stat(full); err != nil || info.Size() > maxScannedFileBytes {
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		content := string(data)
		for _, m := range codeCiteTextRe.FindAllStringSubmatchIndex(content, -1) {
			// A "." that continues as a file name ("Foo.spec.ts") ends no citation;
			// a sentence-final "." or ".md" does.
			if end := m[3]; end+1 < len(content) && content[end] == '.' && isFileNameByte(content[end+1]) && !strings.HasPrefix(content[end:], ".md") {
				continue
			}
			out.citations[content[m[2]:m[3]]] = append(out.citations[content[m[2]:m[3]]], rel)
		}
		for _, m := range looseCiteRe.FindAllStringSubmatch(content, -1) {
			key := m[1] + "." + m[2]
			out.looseCitations[key] = append(out.looseCitations[key], rel)
		}
		if prose {
			continue
		}
		for _, tok := range codeTokenRe.FindAllString(content, -1) {
			if _, ok := wanted[tok]; ok {
				out.identifiers[tok] = true
			}
		}
		for i := 0; i+prefixBytes <= len(content); i++ {
			for _, s := range index[content[i:i+prefixBytes]] {
				if !out.substrings[s] && strings.HasPrefix(content[i:], s) {
					out.substrings[s] = true
				}
			}
		}
	}
	return out
}
