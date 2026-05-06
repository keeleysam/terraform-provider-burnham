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
func prettyEncodeHuJSON(value any, comments attr.Value, indent string) string {
	var b strings.Builder
	encodePrettyValue(&b, value, comments, indent, 0)
	return b.String()
}

func encodePrettyValue(b *strings.Builder, v any, comments attr.Value, indent string, depth int) {
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
		b.WriteString(jsonQuote(x))
	case []any:
		encodePrettyArray(b, x, comments, indent, depth)
	case map[string]any:
		encodePrettyObject(b, x, comments, indent, depth)
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

func encodePrettyObject(b *strings.Builder, m map[string]any, comments attr.Value, indent string, depth int) {
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
		b.WriteString(jsonQuote(k))
		b.WriteString(": ")
		encodePrettyValue(b, m[k], childComments, indent, depth+1)
		b.WriteString(",\n")
	}
	b.WriteString(pad)
	b.WriteString("}")
}

func encodePrettyArray(b *strings.Builder, arr []any, comments attr.Value, indent string, depth int) {
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
		encodePrettyValue(b, el, childComments, indent, depth+1)
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

func jsonQuote(s string) string {
	out, _ := json.Marshal(s)
	return string(out)
}
