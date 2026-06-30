/*
RFC 1751 — A Convention for Human-Readable 128-bit Keys (S/Key).

`btoe` (bytes-to-english) and `etob` (english-to-bytes) carry the RFC's own
function names. They encode a binary key as a sequence of short English words
and back: each 64-bit block becomes six words drawn from a fixed 2048-word
dictionary, with two parity bits appended so a transcription error is caught on
decode. The classic use is reading a key or one-time password aloud, but it is a
general byte↔words codec in the same human-readable-bytes spirit as a fingerprint
word list.

This is a faithful port of the reference C in the RFC's appendix — the bit
`extract`/`insert` helpers, the 2-bit parity, and the `standard()` input
normalization — verified against the three worked examples in the RFC body.
*/

package cryptography

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// rfc1751Index maps each dictionary word to its index, built once from the
// embedded table. The words are already uppercase, matching standard() output.
var rfc1751Index = func() map[string]int {
	m := make(map[string]int, len(rfc1751Words))
	for i, w := range rfc1751Words {
		m[w] = i
	}
	return m
}()

// rfc1751Extract reads `length` (≤ 11) bits starting at bit `start` from s,
// treated as a big-endian bit stream (bit 0 is the MSB of s[0]). Faithful port
// of the RFC's extract(); s must have at least start/8+3 readable bytes, so the
// 8-byte payload lives in an 11-byte buffer.
func rfc1751Extract(s []byte, start, length int) uint {
	cl := uint(s[start/8])
	cc := uint(s[start/8+1])
	cr := uint(s[start/8+2])
	x := ((cl<<8 | cc) << 8) | cr
	x >>= uint(24 - (length + start%8))
	x &= 0xffff >> uint(16-length)
	return x
}

// rfc1751Insert OR-s the low `length` bits of x into s at bit offset `start`.
// s must be zeroed first. Faithful port of the RFC's insert().
func rfc1751Insert(s []byte, x, start, length int) {
	shift := (8 - ((start + length) % 8)) % 8
	y := uint(x) << uint(shift)
	cl := byte((y >> 16) & 0xff)
	cc := byte((y >> 8) & 0xff)
	cr := byte(y & 0xff)
	switch {
	case shift+length > 16:
		s[start/8] |= cl
		s[start/8+1] |= cc
		s[start/8+2] |= cr
	case shift+length > 8:
		s[start/8] |= cc
		s[start/8+1] |= cr
	default:
		s[start/8] |= cr
	}
}

// rfc1751Parity returns the low two bits of the sum of the 32 two-bit groups in
// the first 64 bits of s — the parity quantity the RFC appends as bits 64-65.
func rfc1751Parity(s []byte) uint {
	var p uint
	for i := 0; i < 64; i += 2 {
		p += rfc1751Extract(s, i, 2)
	}
	return p & 3
}

// rfc1751Standard normalizes a word for lookup as the RFC's standard() does:
// uppercase, then the OCR-friendly digit substitutions 1→L, 0→O, 5→S.
func rfc1751Standard(w string) string {
	b := []byte(strings.ToUpper(w))
	for i := range b {
		switch b[i] {
		case '1':
			b[i] = 'L'
		case '0':
			b[i] = 'O'
		case '5':
			b[i] = 'S'
		}
	}
	return string(b)
}

// btoeBlock encodes exactly eight bytes as six space-separated words.
func btoeBlock(block []byte) string {
	cp := make([]byte, 11) // 8 payload bytes + parity, padded for extract()'s 3-byte reads
	copy(cp, block)
	cp[8] = byte(rfc1751Parity(cp) << 6)
	words := make([]string, 6)
	for i := 0; i < 6; i++ {
		words[i] = rfc1751Words[rfc1751Extract(cp, i*11, 11)]
	}
	return strings.Join(words, " ")
}

// btoeBytes encodes a byte slice whose length is a non-zero multiple of 8 into
// the RFC 1751 word representation, one six-word group per 64-bit block.
func btoeBytes(b []byte) (string, error) {
	if len(b) == 0 || len(b)%8 != 0 {
		return "", fmt.Errorf("input must be a non-zero multiple of 8 bytes, got %d", len(b))
	}
	groups := make([]string, 0, len(b)/8)
	for i := 0; i < len(b); i += 8 {
		groups = append(groups, btoeBlock(b[i:i+8]))
	}
	return strings.Join(groups, " "), nil
}

// etobBlock decodes exactly six words back into eight bytes, verifying parity.
func etobBlock(words []string) ([]byte, error) {
	b := make([]byte, 11)
	for i, w := range words {
		idx, ok := rfc1751Index[rfc1751Standard(w)]
		if !ok {
			return nil, fmt.Errorf("word %q is not in the RFC 1751 dictionary", w)
		}
		rfc1751Insert(b, idx, i*11, 11)
	}
	if rfc1751Parity(b) != rfc1751Extract(b, 64, 2) {
		return nil, fmt.Errorf("parity check failed (likely a transcription error)")
	}
	return b[:8], nil
}

// etobWords decodes an RFC 1751 phrase whose word count is a non-zero multiple
// of six back into bytes.
func etobWords(s string) ([]byte, error) {
	words := strings.Fields(s)
	if len(words) == 0 || len(words)%6 != 0 {
		return nil, fmt.Errorf("input must be a non-zero multiple of 6 words, got %d", len(words))
	}
	out := make([]byte, 0, len(words)/6*8)
	for i := 0; i < len(words); i += 6 {
		block, err := etobBlock(words[i : i+6])
		if err != nil {
			return nil, err
		}
		out = append(out, block...)
	}
	return out, nil
}

// ─── btoe ───────────────────────────────────────────────────────

var _ function.Function = (*BtoeFunction)(nil)

type BtoeFunction struct{}

func NewBtoeFunction() function.Function { return &BtoeFunction{} }

func (f *BtoeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "btoe"
}

func (f *BtoeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a key as RFC 1751 English words (bytes-to-english)",
		MarkdownDescription: "Encodes a binary key as a sequence of short English words per [RFC 1751](https://www.rfc-editor.org/rfc/rfc1751) (\"A Convention for Human-Readable 128-bit Keys\"). `btoe` is the RFC's own name for this direction — *bytes-to-english*; `etob` reverses it.\n\nEach 64-bit block of the key becomes six words drawn from a fixed 2048-word dictionary, with two parity bits appended so `etob` can catch a transcription error on the way back. The classic use is reading a key or S/Key one-time password aloud, but it works as a general human-readable encoding for any key material.\n\nThe input is a hex string whose decoded length is a **non-zero multiple of 8 bytes** (64-bit blocks); whitespace in the hex is ignored, so `\"EB33 F77E E73D 4053\"` is accepted. A 128-bit key yields 12 words.\n\n```\nbtoe(\"EB33F77EE73D4053\")\n→ \"TIDE ITCH SLOW REIN RULE MOT\"\n```",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "hex", Description: "The key as a hex string; decoded length must be a non-zero multiple of 8 bytes. Whitespace is ignored."},
		},
		Return: function.StringReturn{},
	}
}

func (f *BtoeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var hexInput string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &hexInput))
	if resp.Error != nil {
		return
	}
	raw, err := hex.DecodeString(strings.Join(strings.Fields(hexInput), ""))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "invalid hex input: "+err.Error())
		return
	}
	words, err := btoeBytes(raw)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &words))
}

// ─── etob ───────────────────────────────────────────────────────

var _ function.Function = (*EtobFunction)(nil)

type EtobFunction struct{}

func NewEtobFunction() function.Function { return &EtobFunction{} }

func (f *EtobFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "etob"
}

func (f *EtobFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode RFC 1751 English words back to a key (english-to-bytes)",
		MarkdownDescription: "Decodes a sequence of [RFC 1751](https://www.rfc-editor.org/rfc/rfc1751) English words back into the original key, returning lowercase hex. `etob` is the RFC's own name for this direction — *english-to-bytes*; `btoe` produces the words.\n\nThe input is a phrase whose word count is a **non-zero multiple of six** (each six words decode to one 64-bit block). Words are matched case-insensitively and the RFC's `standard()` normalization is applied (`1`→`L`, `0`→`O`, `5`→`S`), so dictation/OCR slips are tolerated. The two parity bits embedded by `btoe` are verified; a phrase that fails the parity check (a likely transcription error) is rejected, as is any word not in the dictionary.\n\n```\netob(\"TIDE ITCH SLOW REIN RULE MOT\")\n→ \"eb33f77ee73d4053\"\n```",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "words", Description: "An RFC 1751 word phrase; word count must be a non-zero multiple of 6."},
		},
		Return: function.StringReturn{},
	}
}

func (f *EtobFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var words string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &words))
	if resp.Error != nil {
		return
	}
	raw, err := etobWords(words)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	out := hex.EncodeToString(raw)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
