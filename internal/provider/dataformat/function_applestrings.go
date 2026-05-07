package dataformat

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ function.Function = (*AppleStringsDecodeFunction)(nil)

type AppleStringsDecodeFunction struct{}

func NewAppleStringsDecodeFunction() function.Function { return &AppleStringsDecodeFunction{} }

func (f *AppleStringsDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "applestringsdecode"
}

func (f *AppleStringsDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Parse an Apple .strings localization file",
		MarkdownDescription: "Parses an Apple [`.strings`](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPInternational/MaintaingYourOwnStringsFiles/MaintaingYourOwnStringsFiles.html) localization file body into a flat string-to-string object. " +
			"Both UTF-8 and UTF-16 (with BOM) inputs are auto-detected. `//` and `/* */` comments are tolerated and skipped.\n\n" +
			"Each entry follows `\"key\" = \"value\";` with C-style escapes inside the quoted strings: `\\\\`, `\\\"`, `\\n`, `\\r`, `\\t`, and `\\uXXXX`.\n\n" +
			"**Common uses:** ingesting `Localizable.strings` files for iOS/macOS workflows, building configuration profiles, or running diff/merge logic across translation files at plan time.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "An Apple .strings file body.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *AppleStringsDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	decoded, err := decodeAppleStringsBody(input)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to parse .strings: "+err.Error()))
		return
	}

	attrTypes := make(map[string]attr.Type, len(decoded))
	attrValues := make(map[string]attr.Value, len(decoded))
	for k, v := range decoded {
		attrTypes[k] = types.StringType
		attrValues[k] = types.StringValue(v)
	}
	obj, diags := types.ObjectValue(attrTypes, attrValues)
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to build object: "+diags.Errors()[0].Detail()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(obj)))
}

var _ function.Function = (*AppleStringsEncodeFunction)(nil)

type AppleStringsEncodeFunction struct{}

func NewAppleStringsEncodeFunction() function.Function { return &AppleStringsEncodeFunction{} }

func (f *AppleStringsEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "applestringsencode"
}

func (f *AppleStringsEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode an object as an Apple .strings localization file",
		MarkdownDescription: "Encodes a flat string-keyed object as an Apple `.strings` localization file body. " +
			"Output is UTF-8 with `\"key\" = \"value\";` lines in alphabetical key order. Backslashes (`\\\\`), double quotes (`\\\"`), newline (`\\n`), carriage return (`\\r`), and tab (`\\t`) are escaped; other control characters pass through unchanged. " +
			"Nested objects and lists are not allowed.\n\n" +
			"Output is UTF-8. Modern Xcode toolchains (Xcode 13+) accept UTF-8 `.strings` files; older tooling may require UTF-16 conversion, which can be done with `iconv` after writing the file via `local_file`.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "A flat object whose attributes are localization keys; values must be strings.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *AppleStringsEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value))
	if resp.Error != nil {
		return
	}

	entries, err := appleStringsFromValue(value.UnderlyingValue())
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, renderAppleStrings(entries)))
}

func appleStringsFromValue(v attr.Value) (map[string]string, error) {
	switch tv := v.(type) {
	case basetypes.ObjectValue:
		out := make(map[string]string, len(tv.Attributes()))
		for k, av := range tv.Attributes() {
			sv, ok := av.(basetypes.StringValue)
			if !ok {
				return nil, fmt.Errorf("key %q: .strings values must be strings, got %T", k, av)
			}
			out[k] = sv.ValueString()
		}
		return out, nil
	case basetypes.MapValue:
		out := make(map[string]string, len(tv.Elements()))
		for k, av := range tv.Elements() {
			sv, ok := av.(basetypes.StringValue)
			if !ok {
				return nil, fmt.Errorf("key %q: .strings values must be strings, got %T", k, av)
			}
			out[k] = sv.ValueString()
		}
		return out, nil
	default:
		return nil, fmt.Errorf("stringsencode requires an object or map of strings, got %T", v)
	}
}

func renderAppleStrings(entries map[string]string) string {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		buf.WriteByte('"')
		buf.WriteString(escapeAppleStringsLiteral(k))
		buf.WriteString(`" = "`)
		buf.WriteString(escapeAppleStringsLiteral(entries[k]))
		buf.WriteString("\";\n")
	}
	return buf.String()
}

func escapeAppleStringsLiteral(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// decodeAppleStringsBody parses the .strings format. Detects UTF-16 BOM and converts to UTF-8 first; tolerates // and /* */ comments; reads "key" = "value"; entries.
func decodeAppleStringsBody(input string) (map[string]string, error) {
	body, err := normalizeAppleStringsToUTF8(input)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string)
	p := appleStringsParser{src: body}
	for {
		p.skipWhitespaceAndComments()
		if p.eof() {
			break
		}
		key, err := p.readQuotedString()
		if err != nil {
			return nil, err
		}
		p.skipWhitespaceAndComments()
		if !p.consume('=') {
			return nil, fmt.Errorf("expected '=' after key %q at offset %d", key, p.pos)
		}
		p.skipWhitespaceAndComments()
		value, err := p.readQuotedString()
		if err != nil {
			return nil, err
		}
		p.skipWhitespaceAndComments()
		if !p.consume(';') {
			return nil, fmt.Errorf("expected ';' after value for key %q at offset %d", key, p.pos)
		}
		out[key] = value
	}
	return out, nil
}

func normalizeAppleStringsToUTF8(input string) (string, error) {
	bs := []byte(input)
	if len(bs) >= 2 {
		if bs[0] == 0xFF && bs[1] == 0xFE {
			// UTF-16LE
			return decodeUTF16(bs[2:], false)
		}
		if bs[0] == 0xFE && bs[1] == 0xFF {
			// UTF-16BE
			return decodeUTF16(bs[2:], true)
		}
	}
	// UTF-8: strip optional BOM.
	if len(bs) >= 3 && bs[0] == 0xEF && bs[1] == 0xBB && bs[2] == 0xBF {
		return string(bs[3:]), nil
	}
	return input, nil
}

func decodeUTF16(bs []byte, bigEndian bool) (string, error) {
	if len(bs)%2 != 0 {
		return "", fmt.Errorf("UTF-16 input has odd byte length")
	}
	u16 := make([]uint16, len(bs)/2)
	for i := 0; i < len(u16); i++ {
		if bigEndian {
			u16[i] = uint16(bs[2*i])<<8 | uint16(bs[2*i+1])
		} else {
			u16[i] = uint16(bs[2*i+1])<<8 | uint16(bs[2*i])
		}
	}
	runes := utf16.Decode(u16)
	out := make([]byte, 0, len(runes))
	buf := make([]byte, 4)
	for _, r := range runes {
		n := utf8.EncodeRune(buf, r)
		out = append(out, buf[:n]...)
	}
	return string(out), nil
}

type appleStringsParser struct {
	src string
	pos int
}

func (p *appleStringsParser) eof() bool { return p.pos >= len(p.src) }

func (p *appleStringsParser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *appleStringsParser) consume(c byte) bool {
	if p.peek() == c {
		p.pos++
		return true
	}
	return false
}

func (p *appleStringsParser) skipWhitespaceAndComments() {
	for !p.eof() {
		c := p.peek()
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.pos++
		case c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/':
			for !p.eof() && p.peek() != '\n' {
				p.pos++
			}
		case c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '*':
			p.pos += 2
			for !p.eof() {
				if p.peek() == '*' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/' {
					p.pos += 2
					break
				}
				p.pos++
			}
		default:
			return
		}
	}
}

func (p *appleStringsParser) readQuotedString() (string, error) {
	if !p.consume('"') {
		return "", fmt.Errorf("expected '\"' at offset %d", p.pos)
	}
	var b strings.Builder
	for !p.eof() {
		c := p.peek()
		if c == '"' {
			p.pos++
			return b.String(), nil
		}
		if c == '\\' {
			p.pos++
			if p.eof() {
				return "", fmt.Errorf("unterminated escape at offset %d", p.pos)
			}
			esc := p.peek()
			p.pos++
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			case '0':
				b.WriteByte(0)
			case 'u', 'U':
				if p.pos+4 > len(p.src) {
					return "", fmt.Errorf("truncated \\u escape at offset %d", p.pos)
				}
				hex := p.src[p.pos : p.pos+4]
				p.pos += 4
				var cp uint32
				for i := 0; i < 4; i++ {
					cp <<= 4
					switch ch := hex[i]; {
					case ch >= '0' && ch <= '9':
						cp |= uint32(ch - '0')
					case ch >= 'a' && ch <= 'f':
						cp |= uint32(ch-'a') + 10
					case ch >= 'A' && ch <= 'F':
						cp |= uint32(ch-'A') + 10
					default:
						return "", fmt.Errorf("invalid hex digit %q in \\u escape", ch)
					}
				}
				b.WriteRune(rune(cp))
			default:
				b.WriteByte(esc)
			}
			continue
		}
		b.WriteByte(c)
		p.pos++
	}
	return "", fmt.Errorf("unterminated string starting before offset %d", p.pos)
}
