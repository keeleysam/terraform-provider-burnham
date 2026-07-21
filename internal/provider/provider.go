package provider

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/keeleysam/terraform-burnham/internal/provider/cedar"
	"github.com/keeleysam/terraform-burnham/internal/provider/cel"
	"github.com/keeleysam/terraform-burnham/internal/provider/color"
	"github.com/keeleysam/terraform-burnham/internal/provider/compression"
	"github.com/keeleysam/terraform-burnham/internal/provider/cryptography"
	"github.com/keeleysam/terraform-burnham/internal/provider/dataformat"
	"github.com/keeleysam/terraform-burnham/internal/provider/documents"
	"github.com/keeleysam/terraform-burnham/internal/provider/encoding"
	"github.com/keeleysam/terraform-burnham/internal/provider/geographic"
	"github.com/keeleysam/terraform-burnham/internal/provider/identifiers"
	"github.com/keeleysam/terraform-burnham/internal/provider/image"
	"github.com/keeleysam/terraform-burnham/internal/provider/network"
	"github.com/keeleysam/terraform-burnham/internal/provider/numerics"
	"github.com/keeleysam/terraform-burnham/internal/provider/oel"
	"github.com/keeleysam/terraform-burnham/internal/provider/promql"
	"github.com/keeleysam/terraform-burnham/internal/provider/regex"
	"github.com/keeleysam/terraform-burnham/internal/provider/text"
	"github.com/keeleysam/terraform-burnham/internal/provider/transform"
)

var (
	_ provider.Provider              = (*BurnhamProvider)(nil)
	_ provider.ProviderWithFunctions = (*BurnhamProvider)(nil)
)

type BurnhamProvider struct{}

func New() provider.Provider {
	return &BurnhamProvider{}
}

func (p *BurnhamProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "burnham"
}

func (p *BurnhamProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A pure provider-defined function provider organised into fifteen families: expression languages (CEL: celencode: build a CEL / Common Expression Language string from an HCL data tree mirroring the cel/expr/syntax.proto canonical AST, in a readable surface notation or the canonical field-name notation, mixable; celvalidate: report whether a CEL string is syntactically valid (bool); celformat: canonicalize and pretty-print a hand-written CEL string; celdecode: parse a CEL string back into the celencode data tree in a chosen notation, so celencode/celdecode round-trip; celevaluate: evaluate a standard CEL expression at plan time against variable bindings; the encode/validate/decode functions are syntax-only and dialect-neutral so they suit GCP IAM / Access Context Manager, Kubernetes CEL, and any other CEL sink, while celevaluate evaluates cel-go's standard library plus extensions; Okta Expression Language: oelencode: build an OEL string from an HCL data tree so Okta group-rule and profile-mapping expressions are assembled from Terraform data instead of hand-escaped strings, oelvalidate: report whether a string is syntactically valid OEL, oelformat: canonicalize a hand-written OEL string, oeldecode: parse an OEL string back into the oelencode data tree so oelencode and oeldecode round-trip, all backed by github.com/stevenewson/okta-expression-parser; Cedar: the authorization policy language behind Amazon Verified Permissions, where cedarencode and cedardecode convert a policy between its DSL and canonical EST JSON forms, cedarvalidate and cedarformat check and canonicalize a policy document, and cedarevaluate authorizes a request against it using cedar-go, the official implementation, so the decision matches the engine Amazon Verified Permissions runs; PromQL: promqlencode: build a PromQL / Prometheus Query Language query from an HCL data tree modeled on the Prometheus AST so a query is assembled from Terraform data with correct matcher quoting instead of fragile string interpolation, promqldecode: parse a query back into that data tree so promqlencode and promqldecode round-trip, promqlvalidate: report whether a string is a valid PromQL expression (bool, type-checked while parsing, never fails the plan), promqlformat: canonicalize and optionally pretty-print a hand-written PromQL query, all backed by github.com/prometheus/prometheus's own parser so a query that validates here is valid in Prometheus), compression (base64zopfli, a tighter RFC 1952 gzip drop-in for base64gzip via Zopfli's iterative DEFLATE encoder; base64brotli, RFC 7932 Brotli; both pure-Go, CGO-free, and deterministic for plan stability), cryptography (HMAC, HKDF, PEM/X.509/CSR/ASN.1 inspection, and a deterministic signing pipeline (`{ecdsa_p256,ed25519}_key_from_seed` + `x509_self_sign` + `pkcs7_sign`) for byte-stable Terraform-driven CMS/PKCS#7 signing, ECDSA via RFC 6979 deterministic `k` and Ed25519 via naturally-deterministic PureEdDSA per RFC 8032 / RFC 8419, plus a deterministic JOSE stack: jwt_sign / jwt_decode / jwt_verify for compact JWS/JWT (RFC 7515 / 7519, HS/ES256/EdDSA/RS families, ES256 signed via RFC 6979 as fixed R||S per RFC 7518) and jwk_encode / jwk_decode / jwk_thumbprint / jwks for JWK (RFC 7517 / 7638) via go-jose), dataformat (round-trip encoders/decoders for CBOR, HCL, HuJSON, INI, KDL, MessagePack, NDJSON, plist, REG, VDF, .env, .properties, Apple .strings; encode-only for JSON (pretty and RFC 8785 canonical via json_canonicalize), CSV, YAML, since Terraform ships the matching decoders as builtins), encoding (hex and base64 byte codecs: RFC 4648 base64 with URL-safe / no-padding options and a lenient decoder, plus the hex decode Terraform core lacks), geographic (geohash, Plus codes), color (parse/reformat CSS colors, WCAG contrast and readable-text selection, N deterministic distinct colors, blend, ramp, harmony-scheme palettes, snap-to-nearest-in-palette, and OKLCh channel adjustment, all perceptually-uniform and backed by go-colorful), image (svg_render: rasterize an SVG document to a PNG via resvg compiled to WebAssembly and run under wazero, CGO-free and byte-identical across architectures, rendering gradients, clipping, masks, filters, text, and native color emoji), documents (typst_pdf / typst_png / typst_svg / typst_html: typeset a Typst document to a PDF, to PNGs and SVGs per page, or to an experimental self-contained HTML string, via the Typst engine compiled to WebAssembly and run under wazero, with structured HCL passed straight into the document as sys.inputs; CGO-free and deterministic unless the document calls a non-deterministic Typst builtin such as datetime.today()), identifiers (deterministic UUIDv5/v7, Nano ID, petname), network (IPv4/IPv6/CIDR helpers, NAT64, NPTv6, plus pigeon throughput from RFC 1149/2549), numerics (statistics, math helpers (clamp, mod_floor, gcd, lcm), bitwise integer operations (AND/OR/XOR over a list, width-parameterized NOT, arithmetic shifts, popcount, single-bit set/clear/test) that Terraform's language omits entirely, all arbitrary-precision via math/big, and Pi via RFC 3091 PDGP backed by an embedded ⌊π × 10⁶⌋-digit DPD-packed table), text (Unicode normalize, slugify, Levenshtein, wrap, dedent, parse_kv, cowsay, QR ASCII), regex (pcre_match / pcre_captures / pcre_find_all / pcre_replace / pcre_split: PCRE-flavored regular expressions with backreferences and lookaround via the fancy-regex engine compiled to WebAssembly and run under wazero, the features Terraform's RE2-based regex functions omit; CGO-free and deterministic), and transform (jq, JMESPath, JSONata, JSONPath, JSON Patch / Merge Patch, where jsonata_query evaluates a JSONata 2.x expression to query, aggregate, and reshape a value with the non-deterministic builtins $now / $millis / $random rejected so plan equals apply, and jsonata_validate reports whether an expression is syntactically valid without ever failing the plan). No resources, no data sources, no remote API calls: every function evaluates at plan time.",
	}
}

func (p *BurnhamProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *BurnhamProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *BurnhamProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *BurnhamProvider) Functions(_ context.Context) []func() function.Function {
	return slices.Concat(
		cedar.Functions(),
		cel.Functions(),
		color.Functions(),
		compression.Functions(),
		cryptography.Functions(),
		dataformat.Functions(),
		documents.Functions(),
		encoding.Functions(),
		geographic.Functions(),
		identifiers.Functions(),
		image.Functions(),
		network.Functions(),
		numerics.Functions(),
		oel.Functions(),
		promql.Functions(),
		regex.Functions(),
		text.Functions(),
		transform.Functions(),
	)
}
