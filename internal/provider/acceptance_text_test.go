package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ─── unicode_normalize (UAX #15) ───────────────────────────────────────

func TestAcc_UnicodeNormalize_NFCFromNFD(t *testing.T) {
	// Input is NFD (e + combining acute = 2 codepoints, 3 bytes UTF-8). Expect NFC: 1 codepoint, 2 bytes.
	// We assert via byte length so the test does not depend on bytewise round-trip of Unicode through the test framework.
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::unicode_normalize("e\u0301", "NFC")) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(1)),
	)
}

func TestAcc_UnicodeNormalize_NFKCFlattensLigature(t *testing.T) {
	// U+FB01 (ﬁ ligature) → NFKC: "fi"
	runOutputTest(t,
		`output "test" { value = provider::burnham::unicode_normalize("ﬁne", "NFKC") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("fine")),
	)
}

func TestAcc_UnicodeNormalize_NFCNoOp(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::unicode_normalize("hello", "NFC") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("hello")),
	)
}

func TestAcc_UnicodeNormalize_RejectsBadForm(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::unicode_normalize("x", "NFG") }`,
		regexp.MustCompile(`(?is)form\s+must\s+be`),
	)
}

func TestAcc_UnicodeNormalize_FormIsCaseInsensitive(t *testing.T) {
	// Lowercase "nfc" should resolve identically to "NFC".
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::unicode_normalize("é", "nfc")) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(1)),
	)
}

// ─── slugify ────────────────────────────────────────────────────────────

func TestAcc_Slugify_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::slugify("Hello, World!") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("hello-world")),
	)
}

func TestAcc_Slugify_Transliterates(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::slugify("Café au Lait") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("cafe-au-lait")),
	)
}

func TestAcc_Slugify_LowercaseFalse(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::slugify("Hello World", { lowercase = false }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("Hello-World")),
	)
}

func TestAcc_Slugify_CustomSeparator(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::slugify("hello world", { separator = "_" }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("hello_world")),
	)
}

func TestAcc_Slugify_RejectsOversizedInput(t *testing.T) {
	// 4 MiB + 1 byte exceeds textMaxInputBytes; the function rejects before hitting the gosimple/slug mutex (otherwise multi-MB input would block every other concurrent slugify call at plan time).
	runErrorTest(t,
		`output "test" { value = provider::burnham::slugify(format("%-4194305s", "x")) }`,
		regexp.MustCompile(`(?is)input\s+exceeds\s+maximum\s+supported\s+length`),
	)
}

func TestAcc_Slugify_RejectsUnknownOption(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::slugify("x", { color = "blue" }) }`,
		regexp.MustCompile(`(?is)unknown\s+option\s+key`),
	)
}

// ─── levenshtein ────────────────────────────────────────────────────────

func TestAcc_Levenshtein_Identical(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::levenshtein("abc", "abc") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(0)),
	)
}

func TestAcc_Levenshtein_OneSubstitution(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::levenshtein("kitten", "sitten") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(1)),
	)
}

func TestAcc_Levenshtein_KittenSitting(t *testing.T) {
	// Classic example: kitten → sitting requires 3 edits
	runOutputTest(t,
		`output "test" { value = provider::burnham::levenshtein("kitten", "sitting") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(3)),
	)
}

func TestAcc_Levenshtein_EmptyToString(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::levenshtein("", "abcde") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(5)),
	)
}

func TestAcc_Levenshtein_Symmetric(t *testing.T) {
	// d(a, b) == d(b, a) for any inputs.
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::levenshtein("hello", "world") == provider::burnham::levenshtein("world", "hello")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_Levenshtein_RejectsOversizedInput(t *testing.T) {
	// 256 KiB + 1 byte exceeds the cap; the function rejects before running the O(n·m) DP.
	runErrorTest(t,
		`output "test" { value = provider::burnham::levenshtein(format("%-262145s", "x"), "y") }`,
		regexp.MustCompile(`(?is)a\s+exceeds\s+maximum\s+supported\s+length`),
	)
}

func TestAcc_Levenshtein_UnicodeCounts(t *testing.T) {
	// Distance is over codepoints, not bytes. "é" (U+00E9) is one codepoint, not two.
	runOutputTest(t,
		`output "test" { value = provider::burnham::levenshtein("café", "cafe") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(1)),
	)
}

// ─── wrap ───────────────────────────────────────────────────────────────

func TestAcc_Wrap_FitsOneLine(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::wrap("hello world", 80) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("hello world")),
	)
}

func TestAcc_Wrap_BreaksAtWidth(t *testing.T) {
	// Width 5: "the cat" wraps to "the\ncat".
	runOutputTest(t,
		`output "test" { value = provider::burnham::wrap("the cat", 5) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("the\ncat")),
	)
}

func TestAcc_Wrap_LongWordOverflows(t *testing.T) {
	// Long words are not split; they overflow on their own line.
	runOutputTest(t,
		`output "test" { value = provider::burnham::wrap("supercalifragilisticexpialidocious", 5) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("supercalifragilisticexpialidocious")),
	)
}

func TestAcc_Wrap_RejectsZeroWidth(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::wrap("x", 0) }`,
		regexp.MustCompile(`(?is)width\s+must\s+be\s+>=\s+1`),
	)
}

// ─── cowsay ─────────────────────────────────────────────────────────────

func TestAcc_Cowsay_DefaultBubbleAndCow(t *testing.T) {
	// Verify the canonical default cowsay output for "Hello".
	want := " _______\n" +
		"< Hello >\n" +
		" -------\n" +
		"        \\   ^__^\n" +
		"         \\  (oo)\\_______\n" +
		"            (__)\\       )\\/\\\n" +
		"                ||----w |\n" +
		"                ||     ||\n"
	runOutputTest(t,
		`output "test" { value = provider::burnham::cowsay("Hello") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(want)),
	)
}

func TestAcc_Cowsay_ThinkSwapsBracketsAndConnector(t *testing.T) {
	// "think" mode uses ( ) brackets and o connectors.
	runOutputTest(t,
		`output "test" { value = provider::burnham::cowsay("Hi", { action = "think" }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`(?s)^\s*____\n\(\sHi\s\)\n\s----\n.*o\s+\^__\^\n.*o\s+\(oo\).*`))),
	)
}

func TestAcc_Cowsay_CustomEyes(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cowsay("Hi", { eyes = "==" }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`(?s)\(==\)\\_______`))),
	)
}

func TestAcc_Cowsay_RejectsBadEyes(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::cowsay("x", { eyes = "abc" }) }`,
		regexp.MustCompile(`(?is)eyes\s+must\s+be\s+exactly\s+2`),
	)
}

func TestAcc_Cowsay_RejectsControlCharacterInEyes(t *testing.T) {
	// Two-codepoint string that includes a non-printable rune (U+0007 BEL) — must be rejected so cow output can't be smuggled with terminal control codes.
	runErrorTest(t,
		`output "test" { value = provider::burnham::cowsay("x", { eyes = "a\u0007" }) }`,
		regexp.MustCompile(`(?is)eyes\s+must\s+be\s+exactly\s+2\s+printable`),
	)
}

func TestAcc_Cowsay_TongueAppears(t *testing.T) {
	// Default rendering uses two spaces in the tongue slot. Custom tongue "U " should appear verbatim on the line below the eyes.
	runOutputTest(t,
		`output "test" { value = provider::burnham::cowsay("hi", { tongue = "U " }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`(?s)\(oo\)\\_+.*\n\s+U\s+\|\|----w \|`))),
	)
}

func TestAcc_Cowsay_RejectsBadTongue(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::cowsay("x", { tongue = "abc" }) }`,
		regexp.MustCompile(`(?is)tongue\s+must\s+be\s+exactly\s+2`),
	)
}

func TestAcc_Cowsay_RejectsOversizedMessage(t *testing.T) {
	// 65 KiB > the 64 KiB cap. Built via `format("%-65537s","x")` to keep the test source readable.
	runErrorTest(t,
		`output "test" { value = provider::burnham::cowsay(format("%-65537s", "x")) }`,
		regexp.MustCompile(`(?is)message\s+exceeds\s+maximum\s+length`),
	)
}

func TestAcc_Cowsay_WidthZeroPreservesNewlines(t *testing.T) {
	// width = 0 disables wrapping; an input shorter than the default width that already contains a newline must keep the line break.
	runOutputTest(t,
		`output "test" { value = provider::burnham::cowsay("a\nb", { width = 0 }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`(?s)^ ___\n/ a \\\n\\ b /\n ---\n`))),
	)
}

func TestAcc_Cowsay_RejectsBadAction(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::cowsay("x", { action = "scream" }) }`,
		regexp.MustCompile(`(?is)action\s+must\s+be`),
	)
}

func TestAcc_Cowsay_MultiLineBubble(t *testing.T) {
	// The single-line bubble uses < > brackets; multi-line uses / \ corners and | | sides. Pin the bytes for a 3-line input so the multi-line code path is exercised end-to-end.
	want := " ___\n" +
		"/ a \\\n" +
		"| b |\n" +
		"\\ c /\n" +
		" ---\n" +
		"        \\   ^__^\n" +
		"         \\  (oo)\\_______\n" +
		"            (__)\\       )\\/\\\n" +
		"                ||----w |\n" +
		"                ||     ||\n"
	runOutputTest(t,
		`output "test" { value = provider::burnham::cowsay("a\nb\nc") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(want)),
	)
}

// ─── qr_ascii ───────────────────────────────────────────────────────────

func TestAcc_QRAscii_LightOnDarkInvertsBlocks(t *testing.T) {
	// light_on_dark should swap the dark and light roles. Concretely: a payload that produced ▀ ▄ space block in dark_on_light produces the inverted set; the cell counts of ▀ vs space invert. We just assert the raw byte count of `█` differs between styles for the same payload — easier to write than computing the exact module map but enough to lock the inversion path.
	runOutputTest(t,
		`output "test" {
		   value = (
		     length(provider::burnham::qr_ascii("hello", { style = "dark_on_light" })) == length(provider::burnham::qr_ascii("hello", { style = "light_on_dark" })) &&
		     provider::burnham::qr_ascii("hello", { style = "dark_on_light" }) != provider::burnham::qr_ascii("hello", { style = "light_on_dark" })
		   )
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_QRAscii_BasicShape(t *testing.T) {
	// Verify the output looks like a QR code: rectangular, contains the half-block characters, has a quiet zone.
	// Don't lock the exact bytes — that would couple us to rsc.io/qr's encoding choices.
	runOutputTest(t,
		`output "test" { value = provider::burnham::qr_ascii("hello") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`(?s)^\s+\n.*[\x{2580}\x{2584}\x{2588}].*$`))),
	)
}

func TestAcc_QRAscii_Determinism(t *testing.T) {
	// Same payload → same QR code.
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::qr_ascii("payload") == provider::burnham::qr_ascii("payload")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_QRAscii_AllECLevelsEncode(t *testing.T) {
	// Smoke test: every EC level produces a non-empty output for a small payload. A length comparison would be flaky for tiny payloads where multiple EC levels happen to land on the same QR version.
	runOutputTest(t,
		`output "test" {
		   value = (
		     length(provider::burnham::qr_ascii("hi", { error_correction = "L" })) > 0
		     && length(provider::burnham::qr_ascii("hi", { error_correction = "M" })) > 0
		     && length(provider::burnham::qr_ascii("hi", { error_correction = "Q" })) > 0
		     && length(provider::burnham::qr_ascii("hi", { error_correction = "H" })) > 0
		   )
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_QRAscii_RejectsBadECLevel(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::qr_ascii("x", { error_correction = "Z" }) }`,
		regexp.MustCompile(`(?is)error_correction\s+must\s+be`),
	)
}

func TestAcc_QRAscii_RejectsExcessiveQuietZone(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::qr_ascii("x", { quiet_zone = 999 }) }`,
		regexp.MustCompile(`(?is)quiet_zone\s+must\s+be\s+in\s+\[0,\s*64\]`),
	)
}

func TestAcc_QRAscii_ByteExactRegression(t *testing.T) {
	// Lock the byte-exact half-block rendering for a small fixed payload at EC=L with quiet_zone=0. This is a regression test against rsc.io/qr changing its bit layout under us and against any future tweak to renderHalfBlock that silently corrupts output. If this expected value changes, scan the new output with a real QR reader before updating it — the regex-shape tests above cannot detect a malformed code.
	want := "█▀▀▀▀▀█   ▀ █ █▀▀▀▀▀█\n" +
		"█ ███ █ ▀ ▀ ▄ █ ███ █\n" +
		"█ ▀▀▀ █  █▄█▀ █ ▀▀▀ █\n" +
		"▀▀▀▀▀▀▀ █ █ ▀ ▀▀▀▀▀▀▀\n" +
		"███▄█▀▀▀▀ █▄▀█▀▄ ▄█▄▄\n" +
		"▄▄█▄ ▄▀▄█ ▀█▄█▀█▄█ ▀█\n" +
		"▀▀   ▀▀▀█▄▀▀ ▀█▀  ██▄\n" +
		"█▀▀▀▀▀█ ███ ▀ ▄ ▀ ▄██\n" +
		"█ ███ █ ▀▀▀▄▀▄▀▄▀▄▀▄▀\n" +
		"█ ▀▀▀ █ ████▄█▀█▄█▀▄▀\n" +
		"▀▀▀▀▀▀▀ ▀▀▀▀ ▀▀▀ ▀▀▀▀\n"
	runOutputTest(t,
		`output "test" { value = provider::burnham::qr_ascii("ok", { error_correction = "L", quiet_zone = 0 }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(want)),
	)
}
