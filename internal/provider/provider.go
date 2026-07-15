package provider

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/keeleysam/terraform-burnham/internal/provider/cel"
	"github.com/keeleysam/terraform-burnham/internal/provider/compression"
	"github.com/keeleysam/terraform-burnham/internal/provider/cryptography"
	"github.com/keeleysam/terraform-burnham/internal/provider/dataformat"
	"github.com/keeleysam/terraform-burnham/internal/provider/encoding"
	"github.com/keeleysam/terraform-burnham/internal/provider/geographic"
	"github.com/keeleysam/terraform-burnham/internal/provider/identifiers"
	"github.com/keeleysam/terraform-burnham/internal/provider/network"
	"github.com/keeleysam/terraform-burnham/internal/provider/numerics"
	"github.com/keeleysam/terraform-burnham/internal/provider/oel"
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
		Description: "A pure provider-defined function provider organised into eleven families: expression languages (CEL: celencode: build a CEL / Common Expression Language string from an HCL data tree mirroring the cel/expr/syntax.proto canonical AST, in a readable surface notation or the canonical field-name notation, mixable; celvalidate: report whether a CEL string is syntactically valid (bool); celformat: canonicalize and pretty-print a hand-written CEL string; celdecode: parse a CEL string back into the celencode data tree in a chosen notation, so celencode/celdecode round-trip; celevaluate: evaluate a standard CEL expression at plan time against variable bindings; the encode/validate/decode functions are syntax-only and dialect-neutral so they suit GCP IAM / Access Context Manager, Kubernetes CEL, and any other CEL sink, while celevaluate evaluates cel-go's standard library plus extensions; Okta Expression Language: oelencode: build an OEL string from an HCL data tree so Okta group-rule and profile-mapping expressions are assembled from Terraform data instead of hand-escaped strings, oelvalidate: report whether a string is syntactically valid OEL, oelformat: canonicalize a hand-written OEL string, oeldecode: parse an OEL string back into the oelencode data tree so oelencode and oeldecode round-trip, all backed by github.com/stevenewson/okta-expression-parser), compression (base64zopfli — a tighter RFC 1952 gzip drop-in for base64gzip via Zopfli's iterative DEFLATE encoder; base64brotli — RFC 7932 Brotli; both pure-Go, CGO-free, and deterministic for plan stability), cryptography (HMAC, HKDF, PEM/X.509/CSR/ASN.1 inspection, and a deterministic signing pipeline — `{ecdsa_p256,ed25519}_key_from_seed` + `x509_self_sign` + `pkcs7_sign` — for byte-stable Terraform-driven CMS/PKCS#7 signing, ECDSA via RFC 6979 deterministic `k` and Ed25519 via naturally-deterministic PureEdDSA per RFC 8032 / RFC 8419), dataformat (round-trip encoders/decoders for CBOR, HCL, HuJSON, INI, KDL, MessagePack, NDJSON, plist, REG, VDF, .env, .properties, Apple .strings; encode-only for JSON, CSV, YAML — Terraform ships the matching decoders as builtins), encoding (hex and base64 byte codecs — RFC 4648 base64 with URL-safe / no-padding options and a lenient decoder, plus the hex decode Terraform core lacks), geographic (geohash, Plus codes), identifiers (deterministic UUIDv5/v7, Nano ID, petname), network (IPv4/IPv6/CIDR helpers, NAT64, NPTv6, plus pigeon throughput from RFC 1149/2549), numerics (statistics, math helpers, Pi via RFC 3091 PDGP backed by an embedded ⌊π × 10⁶⌋-digit DPD-packed table), text (Unicode normalize, slugify, Levenshtein, wrap, cowsay, QR ASCII), and transform (JMESPath, JSONPath, JSON Patch / Merge Patch). No resources, no data sources, no remote API calls — every function evaluates at plan time.",
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
		cel.Functions(),
		compression.Functions(),
		cryptography.Functions(),
		dataformat.Functions(),
		encoding.Functions(),
		geographic.Functions(),
		identifiers.Functions(),
		network.Functions(),
		numerics.Functions(),
		oel.Functions(),
		text.Functions(),
		transform.Functions(),
	)
}
