package oel

import (
	"bufio"
	"os"
	"strings"
	"testing"

	okta "github.com/keeleysam/okta-expression-parser"
)

// corpusEntry is one reference expression with the section it came from and that section's upstream-support tag.
type corpusEntry struct {
	section string
	tag     string // "SUPPORTED" or "NEEDS-UPSTREAM"
	expr    string
}

// readCorpus parses testdata/corpus.txt: `## name  [TAG]` starts a section, `#` lines are comments, blank lines are skipped, every other line is one expression.
func readCorpus(t *testing.T) []corpusEntry {
	t.Helper()
	f, err := os.Open("testdata/corpus.txt")
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer f.Close()

	var out []corpusEntry
	var section, tag string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "## "):
			section = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			tag = ""
			if i := strings.Index(section, "["); i >= 0 {
				tag = strings.Trim(section[i:], "[]")
				section = strings.TrimSpace(section[:i])
			}
		case strings.HasPrefix(line, "#"):
			continue
		default:
			out = append(out, corpusEntry{section: section, tag: tag, expr: line})
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan corpus: %v", err)
	}
	return out
}

// TestCorpusParses is the coverage guarantee: every documented expression in
// the corpus parses with the (forked) parser the provider references, whether
// it was in the original upstream (SUPPORTED) or added by the fork
// (NEEDS-UPSTREAM). This proves the parser covers all documented Okta EL.
func TestCorpusParses(t *testing.T) {
	p := okta.New()
	var n int
	for _, e := range readCorpus(t) {
		n++
		if _, err := p.Parse(e.expr); err != nil {
			t.Errorf("[%s] corpus expression does not parse: %q: %v", e.section, e.expr, err)
		}
	}
	if n == 0 {
		t.Fatal("no corpus entries found; corpus.txt may be missing or malformed")
	}
}
