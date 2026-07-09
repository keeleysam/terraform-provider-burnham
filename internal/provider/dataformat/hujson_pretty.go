package dataformat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// prettyEncodeHuJSON emits HuJSON in always-expanded form: every object member
// and every array element gets its own line, with trailing commas. This
// bypasses hujson.Format()'s "fit on one line if it can" packing.
//
// `comments` mirrors the data shape — string leaves become // (or /* */ for
// multi-line) comments placed before the matching key/element. Nested objects
// recurse. Missing keys are silently skipped, matching the behavior of the
// default (compact) path.
func prettyEncodeHuJSON(value any, comments attr.Value, indent string, escapeHTML bool) string {
	var b strings.Builder
	encodePrettyValue(&b, value, comments, indent, 0, escapeHTML)
	return b.String()
}

func encodePrettyValue(b *strings.Builder, v any, comments attr.Value, indent string, depth int, escapeHTML bool) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case json.Number:
		b.WriteString(string(x))
	case string:
		b.WriteString(jsonQuote(x, escapeHTML))
	case []any:
		encodePrettyArray(b, x, comments, indent, depth, escapeHTML)
	case map[string]any:
		encodePrettyObject(b, x, comments, indent, depth, escapeHTML)
	default:
		// Fallback: anything else (e.g. numbers from a non-UseNumber decoder)
		// goes through json.Marshal so we still emit valid JSON.
		raw, err := json.Marshal(x)
		if err != nil {
			b.WriteString(fmt.Sprintf("%q", err.Error()))
			return
		}
		b.Write(raw)
	}
}

func encodePrettyObject(b *strings.Builder, m map[string]any, comments attr.Value, indent string, depth int, escapeHTML bool) {
	if len(m) == 0 {
		b.WriteString("{}")
		return
	}
	pad := strings.Repeat(indent, depth)
	childPad := strings.Repeat(indent, depth+1)

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b.WriteString("{\n")
	for _, k := range keys {
		childComments, leafComment := lookupCommentChild(comments, k)
		if leafComment != "" {
			writePrettyComment(b, leafComment, childPad)
		}
		b.WriteString(childPad)
		b.WriteString(jsonQuote(k, escapeHTML))
		b.WriteString(": ")
		encodePrettyValue(b, m[k], childComments, indent, depth+1, escapeHTML)
		b.WriteString(",\n")
	}
	b.WriteString(pad)
	b.WriteString("}")
}

func encodePrettyArray(b *strings.Builder, arr []any, comments attr.Value, indent string, depth int, escapeHTML bool) {
	if len(arr) == 0 {
		b.WriteString("[]")
		return
	}
	pad := strings.Repeat(indent, depth)
	childPad := strings.Repeat(indent, depth+1)

	b.WriteString("[\n")
	for i, el := range arr {
		childComments, leafComment := lookupCommentChild(comments, fmt.Sprintf("%d", i))
		if leafComment != "" {
			writePrettyComment(b, leafComment, childPad)
		}
		b.WriteString(childPad)
		encodePrettyValue(b, el, childComments, indent, depth+1, escapeHTML)
		b.WriteString(",\n")
	}
	b.WriteString(pad)
	b.WriteString("]")
}

// lookupCommentChild returns either a nested comments value to recurse with,
// or a leaf comment string to attach before this key/element. If neither
// applies, both returns are zero values.
func lookupCommentChild(comments attr.Value, key string) (attr.Value, string) {
	if comments == nil {
		return nil, ""
	}
	obj, ok := comments.(basetypes.ObjectValue)
	if !ok {
		return nil, ""
	}
	val, ok := obj.Attributes()[key]
	if !ok {
		return nil, ""
	}
	switch v := val.(type) {
	case basetypes.StringValue:
		return nil, v.ValueString()
	case basetypes.ObjectValue:
		return v, ""
	}
	return nil, ""
}

func writePrettyComment(b *strings.Builder, comment, pad string) {
	if strings.Contains(comment, "\n") {
		escaped := strings.ReplaceAll(comment, "*/", "*\\/")
		b.WriteString(pad)
		b.WriteString("/* ")
		b.WriteString(escaped)
		b.WriteString(" */\n")
		return
	}
	b.WriteString(pad)
	b.WriteString("// ")
	b.WriteString(comment)
	b.WriteString("\n")
}

// escapeHTMLInStrings rewrites `<`, `>` and `&` to their `\uXXXX` escapes, but
// only inside JSON string literals — never inside `//` or `/* */` comments or
// structural syntax. It exists for the compact path: hujson.Format/Pack
// normalizes `\uXXXX` escapes back to literal characters, so honoring
// escape_html=true there means re-escaping the packed output.
func escapeHTMLInStrings(s string) string {
	const (
		normal = iota
		inString
		inLineComment
		inBlockComment
	)
	var b strings.Builder
	b.Grow(len(s))
	state := normal
	escaped := false // previous byte was a backslash inside a string
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch state {
		case normal:
			switch {
			case c == '"':
				state = inString
			case c == '/' && i+1 < len(s) && s[i+1] == '/':
				state = inLineComment
			case c == '/' && i+1 < len(s) && s[i+1] == '*':
				state = inBlockComment
			}
			b.WriteByte(c)
		case inString:
			switch {
			case escaped:
				escaped = false
				b.WriteByte(c)
			case c == '\\':
				escaped = true
				b.WriteByte(c)
			case c == '"':
				state = normal
				b.WriteByte(c)
			case c == '<' || c == '>' || c == '&':
				fmt.Fprintf(&b, "\\u%04x", c)
			default:
				b.WriteByte(c)
			}
		case inLineComment:
			if c == '\n' {
				state = normal
			}
			b.WriteByte(c)
		case inBlockComment:
			b.WriteByte(c)
			if c == '*' && i+1 < len(s) && s[i+1] == '/' {
				i++
				b.WriteByte(s[i])
				state = normal
			}
		}
	}
	return b.String()
}

// jsonQuote renders s as a JSON string literal. When escapeHTML is false it
// leaves `<`, `>` and `&` literal (the default, matching jsonencode); when true
// it escapes them to their `\uXXXX` forms as Go's encoder does.
func jsonQuote(s string, escapeHTML bool) string {
	if escapeHTML {
		out, _ := json.Marshal(s)
		return string(out)
	}
	out, _ := marshalNoEscapeHTML(s)
	return string(out)
}
