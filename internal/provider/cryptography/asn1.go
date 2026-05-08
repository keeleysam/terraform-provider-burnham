/*
Generic ASN.1 BER/DER decoder.

Pure-Go ASN.1 libraries assume schema-driven decoding — you tell them the structure ahead of time and they validate against it. That's the right design when you control both ends of a protocol, but a Terraform `asn1_decode(...)` needs to walk arbitrary structures (an extension OID payload, a field inside an opaque blob) and emit a structural tree the caller can pick through with `try()` and string-keyed access.

So we walk it ourselves with `encoding/asn1.RawValue`, recursively unwrapping constructed types and decoding leaves by tag. Output is a self-similar tree:

  {
    tag        = 16,           // BER/DER tag number
    class      = "universal",  // "universal" / "application" / "context" / "private"
    compound   = true,         // constructed (has children) vs primitive (has value)
    type       = "SEQUENCE",   // human label for universal tags; "" for non-universal
    value      = null,         // primitive value, null when compound = true
    children   = [...nested]   // list of decoded child elements, null when compound = false
  }

Input is base64-encoded DER bytes (the same shape `pem_decode` returns in `base64_body`). This keeps inputs ASCII-safe inside HCL.
*/

package cryptography

import (
	"context"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = (*ASN1DecodeFunction)(nil)

type ASN1DecodeFunction struct{}

func NewASN1DecodeFunction() function.Function { return &ASN1DecodeFunction{} }

func (f *ASN1DecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "asn1_decode"
}

func (f *ASN1DecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Walk an ASN.1 DER byte string into a structural tree",
		MarkdownDescription: "Decodes ASN.1 DER (or BER) bytes — supplied base64-encoded — into a recursive object tree. Each node has the same shape:\n\n- `tag` — the BER tag number (`2` for INTEGER, `6` for OBJECT IDENTIFIER, `16` for SEQUENCE, …).\n- `class` — `\"universal\"`, `\"application\"`, `\"context\"`, or `\"private\"`.\n- `compound` — `true` for constructed values that hold child nodes; `false` for primitive values.\n- `type` — human-readable name for universal-class tags (`\"INTEGER\"`, `\"SEQUENCE\"`, `\"OBJECT IDENTIFIER\"`, …); empty string for non-universal classes.\n- `value` — primitive payload as a string. Tag-specific encoding:\n  - INTEGER → decimal string\n  - BOOLEAN → `\"true\"` / `\"false\"`\n  - OBJECT IDENTIFIER → dotted form (`\"1.3.6.1.5.5.7.3.1\"`)\n  - UTF8String / PrintableString / IA5String / NumericString / GeneralString → the string value\n  - BMPString → UTF-8 (decoded from UCS-2 big-endian)\n  - T61String → the string value when all bytes are ASCII; otherwise `\"t61_hex:<hex>\"` (full ISO 6937 transcoding is intentionally not bundled — pre-encode as UTF8String upstream if you need legible output)\n  - BIT STRING / OCTET STRING → hex\n  - UTCTime / GeneralizedTime → RFC 3339 timestamp\n  - NULL → empty string\n  - other primitives → hex of the raw value bytes\n\n  Always `\"\"` when `compound = true`.\n- `children` — a list of decoded children when `compound = true`; an empty list otherwise (because the framework forbids null lists of objects in a recursive-feeling tree).\n\nInput is base64-encoded DER bytes — the same shape `pem_decode` returns in `base64_body`. This keeps inputs ASCII-safe inside HCL strings.\n\n**`value` is always a string, regardless of tag.** Even INTEGER and BOOLEAN nodes return their value as a textual representation (`\"42\"`, `\"true\"`). Consumers that need a number or bool convert per-tag with `tonumber(node.value)` / `node.value == \"true\"`. The single-typed field keeps the recursive schema buildable in Terraform (the framework can't express a recursive object type with per-node varying value types).\n\nResource limits to bound adversarial input:\n\n- The decoded DER may be at most 8 MiB. Larger inputs are rejected before parsing.\n- Nesting may be at most 64 levels deep. RFC 5280 X.509 nesting fits comfortably under this limit.\n- A single decode may produce at most 100,000 nodes. The largest realistic certs sit around 1,000.\n\nErrors when the bytes are not well-formed BER/DER, when an INTEGER won't fit in `*big.Int`, when a date stamp can't be parsed, or when any of the above limits are exceeded.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "der_base64", Description: "Base64-encoded DER bytes."},
		},
		Return: function.DynamicReturn{},
	}
}

// asn1NodeAttrs is the schema for a single node in the recursive output tree. Because Terraform's framework can't model "a list of *this same shape*" without a recursive ObjectType (which doesn't exist), we model children as a list of dynamic values and let HCL navigate the tree positionally with attribute access plus `try()`.
var asn1NodeAttrs = map[string]attr.Type{
	"tag":      types.Int64Type,
	"class":    types.StringType,
	"compound": types.BoolType,
	"type":     types.StringType,
	"value":    types.StringType,
	"children": types.ListType{ElemType: types.DynamicType},
}

// emptyDynamicList is the canonical empty `list(dynamic)` reused for every primitive ASN.1 node. Sharing one value across all such nodes saves a per-node allocation on deeply-nested structures (cert extensions can nest 4-5 levels with dozens of leaves).
var emptyDynamicList = types.ListValueMust(types.DynamicType, nil)

func (f *ASN1DecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}
	// Cap input size at 8 MiB; the decoded DER will be at most ~6 MiB. Checked on the *raw* string before TrimSpace so a megabyte of leading whitespace can't slip past the bound.
	if len(input) > asn1MaxBase64Bytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("der_base64 input exceeds maximum length of %d bytes", asn1MaxBase64Bytes))
		return
	}
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(input))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "der_base64 must be valid base64: "+err.Error())
		return
	}

	node, err := decodeASN1(der)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "decoding ASN.1: "+err.Error())
		return
	}

	out := types.DynamicValue(node)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

const (
	// asn1MaxDepth bounds recursion. A `SEQUENCE { SEQUENCE { … } }` thousands deep would otherwise grow Go's stack until the goroutine OOMs the Terraform process. RFC 5280 X.509 nesting is well under 16; 64 is generous and still bounded.
	asn1MaxDepth = 64
	// asn1MaxNodes caps total decoded nodes to bound memory regardless of shape — a flat `SEQUENCE { 100k × NULL }` would slip past the depth check otherwise. Realistic certs sit around 1,000.
	asn1MaxNodes = 100_000
	// asn1MaxBase64Bytes is the upper bound on the *encoded* input length we'll accept before even base64-decoding. 8 MiB is multiple orders of magnitude above any real cert / PKCS#7 bundle.
	asn1MaxBase64Bytes = 8 * 1024 * 1024
)

// decodeASN1 parses a single TLV at the start of `data` and returns its decoded representation. Trailing bytes after the first complete TLV are ignored; the caller is responsible for splitting at higher levels.
func decodeASN1(data []byte) (attr.Value, error) {
	var raw asn1.RawValue
	if _, err := asn1.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	nodes := 0
	return rawValueToNode(raw, 0, &nodes)
}

// rawValueToNode turns a single asn1.RawValue into a Terraform object value matching asn1NodeAttrs. `depth` is the current recursion depth (root = 0); `nodes` is a shared counter incremented once per node and rejected past `asn1MaxNodes` to bound memory regardless of tree shape.
func rawValueToNode(raw asn1.RawValue, depth int, nodes *int) (attr.Value, error) {
	if depth >= asn1MaxDepth {
		return nil, fmt.Errorf("ASN.1 nesting exceeds maximum supported depth of %d", asn1MaxDepth)
	}
	*nodes++
	if *nodes > asn1MaxNodes {
		return nil, fmt.Errorf("ASN.1 contains more than %d nodes", asn1MaxNodes)
	}
	className := classToString(raw.Class)
	tagName := universalTagName(raw)
	value := ""
	var children attr.Value = emptyDynamicList

	if raw.IsCompound {
		kids, err := decodeChildren(raw.Bytes, depth+1, nodes)
		if err != nil {
			return nil, err
		}
		children = kids
	} else {
		v, err := decodePrimitive(raw)
		if err != nil {
			return nil, err
		}
		value = v
	}

	obj, diags := types.ObjectValue(asn1NodeAttrs, map[string]attr.Value{
		"tag":      types.Int64Value(int64(raw.Tag)),
		"class":    types.StringValue(className),
		"compound": types.BoolValue(raw.IsCompound),
		"type":     types.StringValue(tagName),
		"value":    types.StringValue(value),
		"children": children,
	})
	if diags.HasError() {
		return nil, fmt.Errorf("building ASN.1 node: %s", diagsToString(diags))
	}
	return obj, nil
}

// decodeChildren walks a constructed value's contents, peeling off one TLV at a time. `depth` is propagated to bound recursion; `nodes` is the shared count threaded through all sibling subtrees.
func decodeChildren(body []byte, depth int, nodes *int) (attr.Value, error) {
	var kids []attr.Value
	rest := body
	for len(rest) > 0 {
		var raw asn1.RawValue
		next, err := asn1.Unmarshal(rest, &raw)
		if err != nil {
			return nil, err
		}
		node, err := rawValueToNode(raw, depth, nodes)
		if err != nil {
			return nil, err
		}
		kids = append(kids, types.DynamicValue(node))
		rest = next
	}
	v, diags := types.ListValue(types.DynamicType, kids)
	if diags.HasError() {
		return nil, fmt.Errorf("building children list: %s", diagsToString(diags))
	}
	return v, nil
}

// classToString translates the numeric ASN.1 class into a human label.
func classToString(c int) string {
	switch c {
	case asn1.ClassUniversal:
		return "universal"
	case asn1.ClassApplication:
		return "application"
	case asn1.ClassContextSpecific:
		return "context"
	case asn1.ClassPrivate:
		return "private"
	}
	return fmt.Sprintf("unknown(%d)", c)
}

// universalTagName returns a stable name for the well-known universal tags. Empty for non-universal classes (where the tag number is opaque without a schema).
func universalTagName(raw asn1.RawValue) string {
	if raw.Class != asn1.ClassUniversal {
		return ""
	}
	switch raw.Tag {
	case asn1.TagBoolean:
		return "BOOLEAN"
	case asn1.TagInteger:
		return "INTEGER"
	case asn1.TagBitString:
		return "BIT STRING"
	case asn1.TagOctetString:
		return "OCTET STRING"
	case asn1.TagNull:
		return "NULL"
	case asn1.TagOID:
		return "OBJECT IDENTIFIER"
	case asn1.TagEnum:
		return "ENUMERATED"
	case asn1.TagUTF8String:
		return "UTF8String"
	case asn1.TagSequence:
		return "SEQUENCE"
	case asn1.TagSet:
		return "SET"
	case asn1.TagNumericString:
		return "NumericString"
	case asn1.TagPrintableString:
		return "PrintableString"
	case asn1.TagT61String:
		return "T61String"
	case asn1.TagIA5String:
		return "IA5String"
	case asn1.TagUTCTime:
		return "UTCTime"
	case asn1.TagGeneralizedTime:
		return "GeneralizedTime"
	case asn1.TagGeneralString:
		return "GeneralString"
	case asn1.TagBMPString:
		return "BMPString"
	}
	return fmt.Sprintf("[universal %d]", raw.Tag)
}

// decodePrimitive renders a primitive value into a string per the per-tag rules documented on asn1_decode.
func decodePrimitive(raw asn1.RawValue) (string, error) {
	if raw.Class != asn1.ClassUniversal {
		return hex.EncodeToString(raw.Bytes), nil
	}
	switch raw.Tag {
	case asn1.TagBoolean:
		var b bool
		if _, err := asn1.Unmarshal(raw.FullBytes, &b); err != nil {
			return "", err
		}
		if b {
			return "true", nil
		}
		return "false", nil
	case asn1.TagInteger, asn1.TagEnum:
		var n *big.Int
		if _, err := asn1.Unmarshal(raw.FullBytes, &n); err != nil {
			return "", err
		}
		return n.String(), nil
	case asn1.TagOID:
		var oid asn1.ObjectIdentifier
		if _, err := asn1.Unmarshal(raw.FullBytes, &oid); err != nil {
			return "", err
		}
		return oid.String(), nil
	case asn1.TagNull:
		return "", nil
	case asn1.TagUTF8String, asn1.TagPrintableString, asn1.TagIA5String, asn1.TagNumericString, asn1.TagGeneralString:
		// These encodings are all ASCII-compatible (PrintableString / IA5String / NumericString are strict ASCII subsets; UTF8String is UTF-8; GeneralString is registered-charset-tagged but in practice ASCII for X.509). Pass the bytes through as a Go string verbatim.
		return string(raw.Bytes), nil
	case asn1.TagBMPString:
		// BMPString is UCS-2 big-endian — two bytes per Unicode codepoint, BMP only. Decode to a UTF-8 Go string so consumers get a real string instead of mojibake.
		if len(raw.Bytes)%2 != 0 {
			return "", fmt.Errorf("BMPString length %d is not a multiple of 2", len(raw.Bytes))
		}
		runes := make([]uint16, len(raw.Bytes)/2)
		for i := range runes {
			runes[i] = binary.BigEndian.Uint16(raw.Bytes[2*i:])
		}
		return string(utf16.Decode(runes)), nil
	case asn1.TagT61String:
		// T61String (Teletex) is the ISO 6937 character set with shifts; mapping it to UTF-8 requires a full charset table that golang.org/x/text doesn't ship. The ASCII range round-trips cleanly, so try a UTF-8 decode and fall back to hex for anything outside ASCII.
		for _, b := range raw.Bytes {
			if b >= 0x80 {
				return "t61_hex:" + hex.EncodeToString(raw.Bytes), nil
			}
		}
		return string(raw.Bytes), nil
	case asn1.TagBitString:
		var bs asn1.BitString
		if _, err := asn1.Unmarshal(raw.FullBytes, &bs); err != nil {
			return "", err
		}
		return hex.EncodeToString(bs.Bytes), nil
	case asn1.TagOctetString:
		return hex.EncodeToString(raw.Bytes), nil
	case asn1.TagUTCTime:
		t, err := time.Parse("0601021504Z0700", string(raw.Bytes))
		if err != nil {
			t, err = time.Parse("060102150405Z0700", string(raw.Bytes))
			if err != nil {
				return "", fmt.Errorf("UTCTime parse: %w", err)
			}
		}
		return t.UTC().Format(time.RFC3339), nil
	case asn1.TagGeneralizedTime:
		// Try a few common GeneralizedTime forms.
		for _, layout := range []string{
			"20060102150405Z0700",
			"20060102150405.000Z0700",
			"20060102150405.999999999Z0700",
		} {
			if t, err := time.Parse(layout, string(raw.Bytes)); err == nil {
				return t.UTC().Format(time.RFC3339Nano), nil
			}
		}
		return "", fmt.Errorf("GeneralizedTime parse: unrecognised format %q", string(raw.Bytes))
	}
	return hex.EncodeToString(raw.Bytes), nil
}
