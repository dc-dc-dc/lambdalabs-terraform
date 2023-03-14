package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &LambdaProvider{}

type LambdaProvider struct {
	version string
}

// LambdaProviderModel describes the provider data model.
type LambdaProviderModel struct {
	ApiKey types.String `tfsdk:"api_key"`
}

func (p *LambdaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "lambdalabs"
	resp.Version = p.version
}

func (p *LambdaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Sensitive:   true,
				Optional:    true,
				Description: "Lambda API key to use",
			},
		},
	}
}

func (p *LambdaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	apiKey := os.Getenv("LAMBDA_API_KEY")

	var data LambdaProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.ApiKey.IsNull() {
		apiKey = data.ApiKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API key Configuration",
			"While configuring the provider, the API key was not found in "+
				"the LAMBDA_API_KEY environment variable or provider "+
				"configuration block api_key attribute.",
		)
	}

	// Example client configuration for data sources and resources
	resp.ResourceData = apiKey
}

func (p *LambdaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewInstanceResource,
		NewSSHKeyResource,
	}
}

func (p *LambdaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &LambdaProvider{
			version: version,
		}
	}
}
