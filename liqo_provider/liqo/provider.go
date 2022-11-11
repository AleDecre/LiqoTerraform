package liqo

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &liqoProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New() provider.Provider {
	return &liqoProvider{}
}

// liqoProvider is the provider implementation.
type liqoProvider struct{}

// liqoProviderModel maps provider schema data to a Go type.
type liqoProviderModel struct {
}

// Metadata returns the provider type name.
func (p *liqoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "liqo"
}

// GetSchema defines the provider-level schema for configuration data.
func (p *liqoProvider) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{}, nil
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *liqoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
}

// DataSources defines the data sources implemented in the provider.
func (p *liqoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *liqoProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPeeringResource, NewGenerateResource,
	}
}
