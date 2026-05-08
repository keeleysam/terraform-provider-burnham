package provider

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/keeleysam/terraform-burnham/internal/provider/cryptography"
	"github.com/keeleysam/terraform-burnham/internal/provider/dataformat"
	"github.com/keeleysam/terraform-burnham/internal/provider/geographic"
	"github.com/keeleysam/terraform-burnham/internal/provider/identifiers"
	"github.com/keeleysam/terraform-burnham/internal/provider/network"
	"github.com/keeleysam/terraform-burnham/internal/provider/numerics"
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
		Description: "A pure provider-defined function provider organised into eight families: cryptography (HMAC, HKDF, PEM/X.509/CSR/ASN.1 inspection), dataformat (CBOR, HCL, HuJSON, INI, KDL, MessagePack, NDJSON, plist, REG, VDF, YAML, JSON, CSV, .env, .properties, Apple .strings), geographic (geohash, Plus codes), identifiers (deterministic UUIDv5/v7, Nano ID, petname), network (IPv4/IPv6/CIDR helpers, NAT64, NPTv6, plus pigeon throughput from RFC 1149/2549), numerics (statistics, math helpers, Pi via RFC 3091 PDGP backed by an embedded ⌊π × 10⁶⌋-digit DPD-packed table), text (Unicode normalize, slugify, Levenshtein, wrap, cowsay, QR ASCII), and transform (JMESPath, JSONPath, JSON Patch / Merge Patch). No resources, no data sources, no remote API calls — every function evaluates at plan time.",
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
		cryptography.Functions(),
		dataformat.Functions(),
		geographic.Functions(),
		identifiers.Functions(),
		network.Functions(),
		numerics.Functions(),
		text.Functions(),
		transform.Functions(),
	)
}
