package regex

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func run(t *testing.T, op uint32, pat, inp, rep string) json.RawMessage {
	t.Helper()
	v, err := runOp(context.Background(), op, pat, inp, rep)
	if err != nil {
		t.Fatalf("runOp op=%d pat=%q: %v", op, pat, err)
	}
	return v
}

func decodeBool(t *testing.T, v json.RawMessage) bool {
	t.Helper()
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		t.Fatalf("decode bool: %v", err)
	}
	return b
}

func decodeMap(t *testing.T, v json.RawMessage) map[string]string {
	t.Helper()
	m := map[string]string{}
	if string(v) != "null" {
		if err := json.Unmarshal(v, &m); err != nil {
			t.Fatalf("decode map: %v", err)
		}
	}
	return m
}

func decodeList(t *testing.T, v json.RawMessage) []string {
	t.Helper()
	var l []string
	if err := json.Unmarshal(v, &l); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	return l
}

func decodeString(t *testing.T, v json.RawMessage) string {
	t.Helper()
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		t.Fatalf("decode string: %v", err)
	}
	return s
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Backreferences: the marquee PCRE feature Go's RE2 rejects outright.
func TestPCREMatchBackreference(t *testing.T) {
	if !decodeBool(t, run(t, opMatch, `(\w+) \1`, "hello hello", "")) {
		t.Fatal("expected a backreference match on repeated word")
	}
	if decodeBool(t, run(t, opMatch, `(\w+) \1`, "hello world", "")) {
		t.Fatal("expected no match on distinct words")
	}
}

// Lookahead/lookbehind: also unsupported by RE2.
func TestPCREMatchLookaround(t *testing.T) {
	if !decodeBool(t, run(t, opMatch, `foo(?=bar)`, "foobar", "")) {
		t.Fatal("lookahead should match")
	}
	if decodeBool(t, run(t, opMatch, `foo(?=bar)`, "foobaz", "")) {
		t.Fatal("lookahead should not match")
	}
	if !decodeBool(t, run(t, opMatch, `(?<=\$)\d+`, "price $42", "")) {
		t.Fatal("lookbehind should match")
	}
}

func TestPCRECaptures(t *testing.T) {
	m := decodeMap(t, run(t, opCaptures, `(?<year>\d{4})-(?<month>\d{2})`, "2026-07", ""))
	for k, want := range map[string]string{"0": "2026-07", "1": "2026", "2": "07", "year": "2026", "month": "07"} {
		if m[k] != want {
			t.Fatalf("captures[%q] = %q, want %q (full=%v)", k, m[k], want, m)
		}
	}
	// No match -> empty map.
	if m := decodeMap(t, run(t, opCaptures, `\d+`, "abc", "")); len(m) != 0 {
		t.Fatalf("no-match captures = %v, want empty", m)
	}
}

func TestPCREFindAll(t *testing.T) {
	if l := decodeList(t, run(t, opFindAll, `\d+`, "a1b22c333", "")); !eq(l, []string{"1", "22", "333"}) {
		t.Fatalf("find_all = %v", l)
	}
}

func TestPCREReplaceBackref(t *testing.T) {
	if s := decodeString(t, run(t, opReplace, `(\w+)@(\w+)`, "user@host", `${2}.${1}`)); s != "host.user" {
		t.Fatalf("replace = %q, want host.user", s)
	}
}

func TestPCRESplit(t *testing.T) {
	if l := decodeList(t, run(t, opSplit, `\s*,\s*`, "a, b ,c", "")); !eq(l, []string{"a", "b", "c"}) {
		t.Fatalf("split = %v", l)
	}
}

func TestPCREInvalidPattern(t *testing.T) {
	if _, err := runOp(context.Background(), opMatch, `(unclosed`, "x", ""); err == nil {
		t.Fatal("expected an error for an invalid pattern")
	}
}

// Invalid UTF-8 in an argument is a clean EngineError, not a silent "" (which would make an
// invalid pattern match everywhere or invalid input never match).
func TestPCREInvalidUTF8(t *testing.T) {
	_, err := runOp(context.Background(), opMatch, "\xff\xfe", "x", "")
	if err == nil {
		t.Fatal("expected an error for an invalid-UTF-8 pattern")
	}
	var ee *EngineError
	if !errors.As(err, &ee) || !strings.Contains(ee.Msg, "UTF-8") {
		t.Fatalf("expected an EngineError mentioning UTF-8, got %T: %v", err, err)
	}
}

// Negative lookahead/lookbehind, the other half of the lookaround family RE2 rejects.
func TestPCREMatchNegativeLookaround(t *testing.T) {
	if !decodeBool(t, run(t, opMatch, `foo(?!bar)`, "foobaz", "")) {
		t.Fatal("negative lookahead should match when 'bar' does not follow")
	}
	if decodeBool(t, run(t, opMatch, `foo(?!bar)`, "foobar", "")) {
		t.Fatal("negative lookahead should not match when 'bar' follows")
	}
	if !decodeBool(t, run(t, opMatch, `(?<!\$)\d+`, "id 42", "")) {
		t.Fatal("negative lookbehind should match a number not preceded by $")
	}
	if decodeBool(t, run(t, opMatch, `(?<!\$)\d`, "$4", "")) {
		t.Fatal("negative lookbehind should not match a lone digit right after $")
	}
}

// A group that did not participate in the match is omitted, so numbered keys can be
// non-contiguous. This is the documented captures contract; guard it against regressions.
func TestPCRECapturesNonParticipating(t *testing.T) {
	m := decodeMap(t, run(t, opCaptures, `(a)?(b)`, "b", ""))
	if m["0"] != "b" || m["2"] != "b" {
		t.Fatalf("captures = %v, want 0=b and 2=b", m)
	}
	if _, ok := m["1"]; ok {
		t.Fatalf("group 1 did not participate; it should be omitted, got %v", m)
	}
}

func TestPCREFindAllNoMatch(t *testing.T) {
	if l := decodeList(t, run(t, opFindAll, `\d+`, "abc", "")); len(l) != 0 {
		t.Fatalf("no-match find_all = %v, want empty list", l)
	}
}

// The marquee replacement forms: named ${name}, bare $1/$2, and the $$ literal-dollar escape.
func TestPCREReplaceForms(t *testing.T) {
	if s := decodeString(t, run(t, opReplace, `(?<user>\w+)@(?<host>\w+)`, "user@host", `${host}.${user}`)); s != "host.user" {
		t.Fatalf("named backref replace = %q, want host.user", s)
	}
	if s := decodeString(t, run(t, opReplace, `(\w+)@(\w+)`, "user@host", `$2.$1`)); s != "host.user" {
		t.Fatalf("bare numbered replace = %q, want host.user", s)
	}
	if s := decodeString(t, run(t, opReplace, `\d+`, "cost 5", `$$$0`)); s != "cost $5" {
		t.Fatalf("dollar-escape replace = %q, want 'cost $5'", s)
	}
}

// Split keeps empty fields where separators are adjacent or at the ends, matching the doc.
func TestPCRESplitEmpties(t *testing.T) {
	if l := decodeList(t, run(t, opSplit, `,`, ",a,,b,", "")); !eq(l, []string{"", "a", "", "b", ""}) {
		t.Fatalf("split = %v, want [\"\" a \"\" b \"\"]", l)
	}
}

// Regression for the op-3 abort: a catastrophic backreference pattern trips fancy-regex's
// backtrack limit. Every op, including replace, must surface that as a clean user-facing
// EngineError (not a wasm trap), and replace must behave like match rather than aborting.
func TestPCREReplaceBacktrackLimitIsCleanError(t *testing.T) {
	pat := `^(a*)*\1b$`
	inp := strings.Repeat("a", 40)

	_, matchErr := runOp(context.Background(), opMatch, pat, inp, "")
	_, replaceErr := runOp(context.Background(), opReplace, pat, inp, "X")

	for name, err := range map[string]error{"match": matchErr, "replace": replaceErr} {
		if err == nil {
			t.Fatalf("%s: expected a backtrack-limit error, got nil", name)
		}
		var ee *EngineError
		if !errors.As(err, &ee) {
			t.Fatalf("%s: expected *EngineError (user regex fault), got %T: %v", name, err, err)
		}
		if !strings.Contains(strings.ToLower(ee.Msg), "backtrack") {
			t.Fatalf("%s: error message %q does not mention the backtrack limit", name, ee.Msg)
		}
	}
}
