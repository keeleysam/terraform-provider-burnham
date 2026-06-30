package text

import "testing"

func TestDedentString(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"common spaces", "    a\n      b", "a\n  b"},
		{"tabs", "\t\tx\n\t\ty", "x\ny"},
		{"relative indent preserved", "    if x:\n        y", "if x:\n    y"},
		{"no common indent leaves text untouched", "a\n  b", "a\n  b"},
		{"mixed tab/space prefixes have no common margin", "  a\n\tb", "  a\n\tb"},
		{"whitespace-only line normalized and ignored for margin", "    a\n   \n    b", "a\n\nb"},
		{"leading blank line preserved", "\n    a\n    b", "\na\nb"},
		{"empty string", "", ""},
		{"single line", "    only", "only"},
		{"trailing whitespace on content lines kept", "    a  \n    b", "a  \nb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dedentString(c.in); got != c.want {
				t.Errorf("dedentString(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
