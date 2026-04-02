package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
	resp.Schema = schema.Schema{}
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
	return []func() function.Function{
		NewJSONEncodeFunction,
		NewHuJSONDecodeFunction,
		NewHuJSONEncodeFunction,
		NewPlistDecodeFunction,
		NewPlistEncodeFunction,
		NewPlistDateFunction,
		NewPlistDataFunction,
		NewPlistRealFunction,
		NewINIDecodeFunction,
		NewINIEncodeFunction,
	}
}
