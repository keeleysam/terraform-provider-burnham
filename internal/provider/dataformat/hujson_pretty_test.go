package dataformat

import "testing"

func TestEscapeHTMLInStrings(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "escapes inside string values",
			in:   `{"q": "a < b > c & d"}`,
			want: "{\"q\": \"a \\u003c b \\u003e c \\u0026 d\"}",
		},
		{
			name: "leaves structural syntax untouched",
			in:   `{"a": 1}`,
			want: `{"a": 1}`,
		},
		{
			name: "does not escape inside a line comment",
			in:   "// a < b & c\n{\"q\": \"x\"}",
			want: "// a < b & c\n{\"q\": \"x\"}",
		},
		{
			name: "does not escape inside a block comment",
			in:   "/* a < b & c */\n{\"q\": \"y < z\"}",
			want: "/* a < b & c */\n{\"q\": \"y \\u003c z\"}",
		},
		{
			name: "handles escaped quote within a string",
			in:   `{"q": "he said \"< >\" ok"}`,
			want: "{\"q\": \"he said \\\"\\u003c \\u003e\\\" ok\"}",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := escapeHTMLInStrings(c.in); got != c.want {
				t.Errorf("escapeHTMLInStrings(%q)\n = %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}
