package color

import (
	"math"
	"math/big"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var hexRE = regexp.MustCompile(`^#[0-9a-f]{6}([0-9a-f]{2})?$`)

func TestConvertColor(t *testing.T) {
	cases := []struct {
		name, in, target string
		opts             convertOpts
		want             string
	}{
		{"named to hex", "rebeccapurple", "hex", defaultConvertOpts(), "#663399"},
		{"hex to hex no hash", "#663399", "hex", convertOpts{hash: false}, "663399"},
		{"hex to hex uppercase", "#663399", "hex", convertOpts{hash: true, upper: true}, "#663399"},
		{"hex to rgb", "#663399", "rgb", defaultConvertOpts(), "rgb(102, 51, 153)"},
		{"shorthand hex", "#f06", "hex", defaultConvertOpts(), "#ff0066"},
		{"rgb string to hex", "rgb(255, 0, 102)", "hex", defaultConvertOpts(), "#ff0066"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := convertColor(tc.in, tc.target, tc.opts)
			if err != nil {
				t.Fatalf("convertColor(%q,%q) error: %v", tc.in, tc.target, err)
			}
			if got != tc.want {
				t.Fatalf("convertColor(%q,%q) = %q, want %q", tc.in, tc.target, got, tc.want)
			}
		})
	}
}

func TestConvertColorPreservesAlpha(t *testing.T) {
	got, err := convertColor("#11223344", "hex", defaultConvertOpts())
	if err != nil {
		t.Fatal(err)
	}
	if got != "#11223344" {
		t.Fatalf("alpha hex = %q, want #11223344", got)
	}
}

func TestConvertColorRejectsBadInputAndTarget(t *testing.T) {
	if _, err := convertColor("not-a-color", "hex", defaultConvertOpts()); err == nil {
		t.Fatal("expected error for bad color")
	}
	if _, err := convertColor("#000", "cmyk", defaultConvertOpts()); err == nil {
		t.Fatal("expected error for unknown target")
	}
}

func TestContrastRatio(t *testing.T) {
	r, err := contrastRatio("#000000", "#ffffff")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(r-21.0) > 1e-9 {
		t.Fatalf("black/white ratio = %v, want 21", r)
	}
	if same, _ := contrastRatio("#336699", "#336699"); math.Abs(same-1.0) > 1e-9 {
		t.Fatalf("identical ratio = %v, want 1", same)
	}
	// WebAIM's canonical "just passes AA" grey on white is ~4.54.
	mid, _ := contrastRatio("#767676", "#ffffff")
	if mid < 4.4 || mid > 4.7 {
		t.Fatalf("#767676 on white = %v, want ~4.54", mid)
	}
	// Order must not matter.
	a, _ := contrastRatio("#000", "#fff")
	b, _ := contrastRatio("#fff", "#000")
	if a != b {
		t.Fatalf("ratio not symmetric: %v vs %v", a, b)
	}
}

func TestReadableText(t *testing.T) {
	cases := []struct {
		bg, want string
	}{
		{"#000000", "#ffffff"},
		{"#ffffff", "#000000"},
		{"#ffff00", "#000000"}, // bright yellow -> dark text
		{"#1e3a8a", "#ffffff"}, // deep blue -> light text
	}
	for _, tc := range cases {
		got, err := readableText(tc.bg, []string{"#000000", "#ffffff"})
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("readableText(%q) = %q, want %q", tc.bg, got, tc.want)
		}
	}
}

func TestReadableTextCustomCandidatesReturnedVerbatim(t *testing.T) {
	got, err := readableText("#ffffff", []string{"navy", "#EEE"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "navy" {
		t.Fatalf("readableText custom = %q, want navy (returned as written)", got)
	}
}

func TestDistinctColors(t *testing.T) {
	got, err := distinctColors(5, defaultDistinctOpts())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	seen := map[string]bool{}
	for _, c := range got {
		if !hexRE.MatchString(c) {
			t.Fatalf("not a hex color: %q", c)
		}
		if seen[c] {
			t.Fatalf("duplicate color %q", c)
		}
		seen[c] = true
	}
}

func TestDistinctColorsDeterministic(t *testing.T) {
	a, _ := distinctColors(8, defaultDistinctOpts())
	b, _ := distinctColors(8, defaultDistinctOpts())
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("non-deterministic at %d: %q vs %q", i, a[i], b[i])
		}
	}
}

func TestMixColors(t *testing.T) {
	// In RGB, the midpoint of black and white is mid-grey.
	got, err := mixColors("#000000", "#ffffff", 0.5, "rgb")
	if err != nil {
		t.Fatal(err)
	}
	if got != "#808080" {
		t.Fatalf("mix black/white 0.5 rgb = %q, want #808080", got)
	}
	// Endpoints return the endpoints.
	if a, _ := mixColors("#ff0000", "#0000ff", 0, "oklch"); a != "#ff0000" {
		t.Fatalf("amount 0 = %q, want #ff0000", a)
	}
	if b, _ := mixColors("#ff0000", "#0000ff", 1, "oklch"); b != "#0000ff" {
		t.Fatalf("amount 1 = %q, want #0000ff", b)
	}
	// Out-of-range amount is clamped.
	if c, _ := mixColors("#ff0000", "#0000ff", 5, "oklch"); c != "#0000ff" {
		t.Fatalf("amount 5 clamped = %q, want #0000ff", c)
	}
}

func TestMixColorsUnknownSpace(t *testing.T) {
	if _, err := mixColors("#000", "#fff", 0.5, "cmyk"); err == nil {
		t.Fatal("expected error for unknown space")
	}
}

func TestRampColors(t *testing.T) {
	got, err := rampColors([]string{"#000000", "#ffffff"}, 3, "rgb")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"#000000", "#808080", "#ffffff"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ramp[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	// Endpoints are exact regardless of space.
	ok, _ := rampColors([]string{"#123456", "#abcdef"}, 4, "oklch")
	if ok[0] != "#123456" || ok[3] != "#abcdef" {
		t.Fatalf("ramp endpoints = %q..%q, want #123456..#abcdef", ok[0], ok[3])
	}
	// A single stop yields copies.
	one, _ := rampColors([]string{"#abcabc"}, 3, "oklch")
	if len(one) != 3 || one[0] != "#abcabc" || one[2] != "#abcabc" {
		t.Fatalf("single-stop ramp = %v, want three #abcabc", one)
	}
}

func TestRampColorsErrors(t *testing.T) {
	if _, err := rampColors([]string{"#000"}, 0, "oklch"); err == nil {
		t.Fatal("expected error for count 0")
	}
	if _, err := rampColors(nil, 3, "oklch"); err == nil {
		t.Fatal("expected error for empty stops")
	}
}

// adj builds an adjustments map from alternating key/value pairs; values that
// look numeric are set as Number, otherwise as String.
func strAdj(pairs ...string) map[string]attr.Value {
	m := map[string]attr.Value{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = basetypes.NewStringValue(pairs[i+1])
	}
	return m
}

func TestAdjustColor(t *testing.T) {
	// Darkening lowers luminance; lightening raises it.
	base, _, _ := parseColor("#3366cc")
	darker, _ := adjustColor("#3366cc", strAdj("lightness", "*0.7"))
	lighter, _ := adjustColor("#3366cc", strAdj("lightness", "+0.15"))
	dc, _, _ := parseColor(darker)
	lc, _, _ := parseColor(lighter)
	if relLuminance(dc) >= relLuminance(base) {
		t.Fatalf("lightness *0.7 did not darken: %q", darker)
	}
	if relLuminance(lc) <= relLuminance(base) {
		t.Fatalf("lightness +0.15 did not lighten: %q", lighter)
	}
	// Chroma 0 (set) grayscales: r == g == b.
	gray, err := adjustColor("#3366cc", strAdj("chroma", "0"))
	if err != nil {
		t.Fatal(err)
	}
	c, _, _ := parseColor(gray)
	r, g, b := c.RGB255()
	if r != g || g != b {
		t.Fatalf("chroma 0 not grey: %q (%d,%d,%d)", gray, r, g, b)
	}
	// Alpha set to 0.5 -> eight-digit hex ending in 80.
	fade, _ := adjustColor("#3366cc", map[string]attr.Value{"alpha": basetypes.NewNumberValue(big.NewFloat(0.5))})
	if len(fade) != 9 || fade[7:] != "80" {
		t.Fatalf("alpha 0.5 = %q, want #......80", fade)
	}
	// Hue set is absolute; +360 wraps back to the same color.
	noop, _ := adjustColor("#3366cc", strAdj())
	wrapped, _ := adjustColor("#3366cc", strAdj("hue", "+360"))
	if noop != wrapped {
		t.Fatalf("hue +360 = %q, want unchanged %q", wrapped, noop)
	}
}

func TestAdjustColorErrors(t *testing.T) {
	if _, err := adjustColor("#000", strAdj("bogus", "+1")); err == nil {
		t.Fatal("expected error for unknown channel")
	}
	if _, err := adjustColor("#000", strAdj("lightness", "*abc")); err == nil {
		t.Fatal("expected error for bad operation")
	}
	if _, err := adjustColor("#000", strAdj("chroma", "/0")); err == nil {
		t.Fatal("expected error for divide by zero")
	}
}
