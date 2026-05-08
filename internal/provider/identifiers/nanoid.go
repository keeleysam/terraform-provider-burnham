/*
Deterministic Nano ID — short, URL-friendly identifiers derived from a seed string.

The upstream [nanoid](https://github.com/ai/nanoid) algorithm draws bytes from a CSPRNG and uses rejection sampling against the chosen alphabet. That is non-deterministic by design and so unsuitable for plan-time HCL: a Terraform plan that produces a different ID on each refresh churns state forever.

This implementation keeps the shape (configurable alphabet, configurable size, default 21-character `_-0-9A-Za-z` alphabet) but replaces the randomness with HMAC-SHA-256 in counter mode keyed by the caller-supplied `seed`. Same seed → same ID, every plan, forever. Each call mixes a context label (`burnham/nanoid`) and the alphabet length into the HMAC input so that asking for a different size or a different alphabet doesn't accidentally produce a prefix of an earlier output.

Modulo bias: bytes 0–255 mapped via `b % len(alphabet)` are exactly uniform when `len(alphabet)` divides 256 evenly (e.g. 32, 64, 128). For odd-sized alphabets there's a tiny per-character bias proportional to `256 mod len(alphabet)`. This matters in the original CSPRNG-driven nanoid (where you want true uniform distribution); here, where the goal is just a stable derivation from a seed, the bias is harmless — the outputs are not used as cryptographic keys.
*/

package identifiers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// nanoidDefaultAlphabet is the 64-character URL-safe alphabet used by upstream nanoid by default. Order matches the reference implementation.
const nanoidDefaultAlphabet = "_-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// nanoidDefaultSize is upstream nanoid's default output length.
const nanoidDefaultSize = 21

// nanoidMaxSize bounds output length; nothing in the algorithm needs a cap, but plan-time strings have no realistic use case past a few hundred chars and capping prevents pathological allocations.
const nanoidMaxSize = 1024

var _ function.Function = (*NanoidFunction)(nil)

type NanoidFunction struct{}

func NewNanoidFunction() function.Function { return &NanoidFunction{} }

func (f *NanoidFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "nanoid"
}

func (f *NanoidFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Deterministic Nano ID derived from a seed string",
		MarkdownDescription: "Returns a Nano ID string derived deterministically from `seed` via HMAC-SHA-256 in counter mode. Same `seed` always returns the same ID — perfect for stable, plan-time identifiers that don't churn on re-apply.\n\nDefault alphabet is the 64-character URL-safe set `_-0-9A-Za-z` (matching the upstream [nanoid](https://github.com/ai/nanoid) reference); default `size` is 21 characters. Both can be overridden via the optional `options` object:\n\n- `alphabet` (string) — the alphabet to draw from. Must be non-empty and contain no duplicate runes. Any unicode is accepted; bytes are interpreted as a UTF-8 string and you get one alphabet *codepoint* per output position, so a 64-codepoint alphabet still yields a 21-character (=21-codepoint) ID even if some characters are multi-byte.\n- `size` (number) — output length in codepoints; must be in `[1, 1024]`.\n\n```\nnanoid(\"prod/api-gateway\")\n→ \"3WiSbLYRP4_xQAYVk2DcN\"   (deterministic)\n\nnanoid(\"prod/api-gateway\", { size = 10 })\nnanoid(\"prod/api-gateway\", { alphabet = \"0123456789\", size = 6 })\n```\n\nThis function is a derivation, not a CSPRNG — outputs leak nothing about the seed, but two callers seeded with the same secret will produce the same ID. Use it for naming, not for credentials.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "seed",
				Description: "Stable seed string. Any non-empty string. The empty string is allowed and produces a deterministic but obviously-not-unique ID.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "Optional options object: { alphabet = string, size = number }. At most one allowed.",
		},
		Return: function.StringReturn{},
	}
}

// nanoidOptions parses the optional options object. Returns the resolved (alphabet, size) plus a function error if the object is malformed.
func nanoidOptions(opts []types.Dynamic) (string, int, *function.FuncError) {
	alphabet := nanoidDefaultAlphabet
	size := nanoidDefaultSize
	if len(opts) == 0 {
		return alphabet, size, nil
	}
	if len(opts) > 1 {
		return "", 0, function.NewArgumentFuncError(1, "at most one options argument may be provided")
	}
	obj, ok := opts[0].UnderlyingValue().(basetypes.ObjectValue)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return "", 0, function.NewArgumentFuncError(1, "options must be an object literal, e.g. { size = 10 }")
	}
	for k, val := range obj.Attributes() {
		switch k {
		case "alphabet":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return "", 0, function.NewArgumentFuncError(1, "options.alphabet must be a string")
			}
			alphabet = s.ValueString()
		case "size":
			n, err := numberAttrToInt(val)
			if err != nil {
				return "", 0, function.NewArgumentFuncError(1, "options.size must be a whole number: "+err.Error())
			}
			size = n
		default:
			return "", 0, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are alphabet, size", k))
		}
	}
	return alphabet, size, nil
}

// numberAttrToInt converts a Terraform Number attr.Value (carries a *big.Float internally) into a Go int. Errors when the value is null/unknown, non-integral, or out of int range.
func numberAttrToInt(v attr.Value) (int, error) {
	num, ok := v.(basetypes.NumberValue)
	if !ok {
		return 0, fmt.Errorf("expected a number, got %T", v)
	}
	if num.IsNull() || num.IsUnknown() {
		return 0, fmt.Errorf("value is null or unknown")
	}
	bf := num.ValueBigFloat()
	bi, accuracy := bf.Int(nil)
	if accuracy != big.Exact {
		return 0, fmt.Errorf("not a whole number")
	}
	if !bi.IsInt64() {
		return 0, fmt.Errorf("out of int range")
	}
	return int(bi.Int64()), nil
}

// uniqueRunes verifies the alphabet has no duplicate codepoints. Duplicates would silently bias the output toward those characters.
func uniqueRunes(s string) bool {
	seen := make(map[rune]struct{}, len(s))
	for _, r := range s {
		if _, dup := seen[r]; dup {
			return false
		}
		seen[r] = struct{}{}
	}
	return true
}

// hmacBlock returns HMAC-SHA-256(seed, label || counter_be) — 32 bytes.
func hmacBlock(seed, label []byte, counter uint64) []byte {
	mac := hmac.New(sha256.New, seed)
	mac.Write(label)
	var ctr [8]byte
	binary.BigEndian.PutUint64(ctr[:], counter)
	mac.Write(ctr[:])
	return mac.Sum(nil)
}

func (f *NanoidFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var seed string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &seed, &optsArgs))
	if resp.Error != nil {
		return
	}

	alphabet, size, ferr := nanoidOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	if alphabet == "" {
		resp.Error = function.NewArgumentFuncError(1, "alphabet must be non-empty")
		return
	}
	if !uniqueRunes(alphabet) {
		resp.Error = function.NewArgumentFuncError(1, "alphabet must have no duplicate characters")
		return
	}
	if size < 1 || size > nanoidMaxSize {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("size must be in [1, %d]; received %d", nanoidMaxSize, size))
		return
	}

	runes := []rune(alphabet)
	if len(runes) > 256 {
		// We index the alphabet with one byte per output character. Lifting this would mean drawing 2+ bytes per char, which we can do if a real use case ever appears, but a 256-codepoint cap is plenty for any human-readable alphabet.
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("alphabet may have at most 256 codepoints; received %d", len(runes)))
		return
	}
	mod := byte(len(runes))

	seedBytes := []byte(seed)
	label := []byte("burnham/nanoid:" + alphabet)
	var b strings.Builder
	b.Grow(size * utf8.UTFMax)

	// One HMAC block produces 32 bytes; counter advances when consumed.
	var block []byte
	var bytesUsed int
	var counter uint64
	refill := func() {
		block = hmacBlock(seedBytes, label, counter)
		counter++
		bytesUsed = 0
	}
	refill()

	for i := 0; i < size; i++ {
		if bytesUsed >= len(block) {
			refill()
		}
		idx := block[bytesUsed] % mod
		bytesUsed++
		b.WriteRune(runes[idx])
	}

	out := b.String()
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
