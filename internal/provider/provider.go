// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = (*externalProvider)(nil)

func New() provider.Provider {
	return &externalProvider{}
}

type externalProvider struct{}

func (p *externalProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "external"
}

func (p *externalProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

}

func (p *externalProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExternalDataSource,
	}
}

func (p *externalProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *externalProvider) Schema(context.Context, provider.SchemaRequest, *provider.SchemaResponse) {
}
