/*
UUID functions: deterministic v5 (SHA-1 namespace UUIDs, RFC 9562 §5.5), deterministic v7 (sortable Unix-time UUIDs, RFC 9562 §5.7), and inspection of any RFC 4122 / RFC 9562 UUID.

Both `uuid_v5` and `uuid_v7` are pure: same inputs always produce the same UUID. v5 is deterministic by construction. v7 is normally non-deterministic (it carries fresh randomness in the rand_a / rand_b fields), but for plan-time use we derive those bits from a caller-supplied `entropy` string via HMAC-SHA-256, so a stable (timestamp, entropy) pair always yields the same UUID.

`uuid_inspect` is a thin wrapper around `github.com/google/uuid` for parsing + version/variant decoding, plus a small chunk of byte-level decoding for the v7 timestamp because it lives in a different layout from v1/v6.
*/

package identifiers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// parseNamespace accepts either one of the four predefined RFC 4122 namespace short names ("dns", "url", "oid", "x500") or any well-formed UUID string. Short-name matching is case-insensitive so "DNS" and "Dns" both work.
func parseNamespace(s string) (uuid.UUID, error) {
	switch strings.ToLower(s) {
	case "dns":
		return uuid.NameSpaceDNS, nil
	case "url":
		return uuid.NameSpaceURL, nil
	case "oid":
		return uuid.NameSpaceOID, nil
	case "x500":
		return uuid.NameSpaceX500, nil
	}
	return uuid.Parse(s)
}

// ──────────────────────────────────────────────────────────────────────
// uuid_v5
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*UUIDv5Function)(nil)

type UUIDv5Function struct{}

func NewUUIDv5Function() function.Function { return &UUIDv5Function{} }

func (f *UUIDv5Function) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "uuid_v5"
}

//go:embed descriptions/uuid_v5.md
var uuidV5Description string

func (f *UUIDv5Function) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Deterministic name-based UUID (version 5, RFC 9562 §5.5)",
		MarkdownDescription: uuidV5Description,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "namespace",
				Description: "Namespace UUID, or one of the predefined short names: \"dns\", \"url\", \"oid\", \"x500\".",
			},
			function.StringParameter{
				Name:        "name",
				Description: "The name to hash within the namespace.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *UUIDv5Function) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var namespaceArg, name string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &namespaceArg, &name))
	if resp.Error != nil {
		return
	}
	ns, err := parseNamespace(namespaceArg)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("namespace must be \"dns\"/\"url\"/\"oid\"/\"x500\" or a UUID; received %q: %s", namespaceArg, err.Error()))
		return
	}
	out := uuid.NewSHA1(ns, []byte(name)).String()
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ──────────────────────────────────────────────────────────────────────
// uuid_v7
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*UUIDv7Function)(nil)

type UUIDv7Function struct{}

func NewUUIDv7Function() function.Function { return &UUIDv7Function{} }

func (f *UUIDv7Function) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "uuid_v7"
}

//go:embed descriptions/uuid_v7.md
var uuidV7Description string

func (f *UUIDv7Function) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Deterministic time-ordered UUID (version 7, RFC 9562 §5.7)",
		MarkdownDescription: uuidV7Description,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "timestamp",
				Description: "RFC 3339 timestamp embedded in the UUID's leading 48 bits as Unix milliseconds.",
			},
			function.StringParameter{
				Name:        "entropy",
				Description: "Salt fed into HMAC-SHA-256 to derive the 74 random-ish bits. Same entropy + same timestamp = same UUID.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *UUIDv7Function) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var timestamp, entropy string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &timestamp, &entropy))
	if resp.Error != nil {
		return
	}
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		// Try plain RFC 3339 as a fallback for callers that don't include sub-second precision.
		t, err = time.Parse(time.RFC3339, timestamp)
	}
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("timestamp must be RFC 3339; received %q: %s", timestamp, err.Error()))
		return
	}

	unixMs := t.UnixMilli()
	if unixMs < 0 || unixMs >= (1<<48) {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("timestamp must fall in the 48-bit Unix-millisecond range [1970-01-01T00:00:00Z, ~+10889 AD); received %q (%d ms)", timestamp, unixMs))
		return
	}

	// Derive 74 random-ish bits from HMAC-SHA-256(entropy, big-endian unix_ts_ms). 32 bytes is far more than we need; we take the leading 10 bytes (80 bits) and mask off the version and variant fields.
	var tsBE [8]byte
	binary.BigEndian.PutUint64(tsBE[:], uint64(unixMs))
	mac := hmac.New(sha256.New, []byte(entropy))
	mac.Write(tsBE[2:]) // first 2 bytes are always zero (unixMs < 2^48); HMAC the meaningful 6
	digest := mac.Sum(nil)

	var u [16]byte
	// Bytes 0–5: 48-bit unix_ts_ms, big-endian.
	u[0] = byte(unixMs >> 40)
	u[1] = byte(unixMs >> 32)
	u[2] = byte(unixMs >> 24)
	u[3] = byte(unixMs >> 16)
	u[4] = byte(unixMs >> 8)
	u[5] = byte(unixMs)
	// Byte 6: high nibble = version 7, low nibble = first 4 bits of rand_a.
	u[6] = 0x70 | (digest[0] & 0x0f)
	// Byte 7: low 8 bits of rand_a.
	u[7] = digest[1]
	// Byte 8: variant (top 2 bits = 10) + first 6 bits of rand_b.
	u[8] = 0x80 | (digest[2] & 0x3f)
	// Bytes 9–15: remaining 56 bits of rand_b.
	copy(u[9:], digest[3:10])

	out := uuid.UUID(u).String()
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ──────────────────────────────────────────────────────────────────────
// uuid_inspect
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*UUIDInspectFunction)(nil)

type UUIDInspectFunction struct{}

func NewUUIDInspectFunction() function.Function { return &UUIDInspectFunction{} }

// uuidInspectAttrs is the fixed object schema returned by uuid_inspect. Defined as a package-level var so the Definition and Run methods stay in sync without copy-paste.
var uuidInspectAttrs = map[string]attr.Type{
	"version":    types.Int64Type,
	"variant":    types.StringType,
	"timestamp":  types.StringType, // null if version doesn't carry one
	"unix_ts_ms": types.Int64Type,  // null unless version == 7
}

func (f *UUIDInspectFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "uuid_inspect"
}

//go:embed descriptions/uuid_inspect.md
var uuidInspectDescription string

func (f *UUIDInspectFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode an RFC 4122 / RFC 9562 UUID into version, variant, and timestamp",
		MarkdownDescription: uuidInspectDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "uuid", Description: "The UUID to inspect, in canonical form (with hyphens) or compact form."},
		},
		Return: function.ObjectReturn{AttributeTypes: uuidInspectAttrs},
	}
}

// variantString maps google/uuid's Variant enum to the human-readable variant names from RFC 4122 §4.1.1 / RFC 9562 §4.1.
func variantString(v uuid.Variant) string {
	switch v {
	case uuid.RFC4122:
		return "RFC 4122"
	case uuid.Reserved:
		return "NCS"
	case uuid.Microsoft:
		return "Microsoft"
	case uuid.Future:
		return "Future"
	case uuid.Invalid:
		return "Invalid"
	default:
		return "Invalid"
	}
}

func (f *UUIDInspectFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var s string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &s))
	if resp.Error != nil {
		return
	}
	u, err := uuid.Parse(s)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("not a valid UUID: %s", err.Error()))
		return
	}

	version := int64(u.Version())
	variant := variantString(u.Variant())

	// Default: no timestamp.
	timestamp := types.StringNull()
	unixTsMs := types.Int64Null()

	switch u.Version() {
	case 1, 6:
		// google/uuid's Time() returns a Time int64 = 100ns intervals since 1582-10-15 (Gregorian start). UnixTime() converts to (sec, nsec) since Unix epoch.
		sec, nsec := u.Time().UnixTime()
		timestamp = types.StringValue(time.Unix(sec, nsec).UTC().Format(time.RFC3339Nano))
	case 7:
		// v7 layout: first 6 bytes are 48-bit unix_ts_ms, big-endian.
		ms := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 | int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5])
		unixTsMs = types.Int64Value(ms)
		timestamp = types.StringValue(time.UnixMilli(ms).UTC().Format(time.RFC3339Nano))
	}

	out, diags := types.ObjectValue(uuidInspectAttrs, map[string]attr.Value{
		"version":    types.Int64Value(version),
		"variant":    types.StringValue(variant),
		"timestamp":  timestamp,
		"unix_ts_ms": unixTsMs,
	})
	if diags.HasError() {
		e := diags.Errors()[0]
		resp.Error = function.NewFuncError(fmt.Sprintf("building inspect result: %s: %s", e.Summary(), e.Detail()))
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
