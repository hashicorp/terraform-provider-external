package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"input": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Default:     "",
				Description: "Pass this attribute as additional input to resources (eg. as short-term credentials).",
			},
		},
		ConfigureContextFunc: func(ctx context.Context, data *schema.ResourceData) (meta interface{}, diags diag.Diagnostics) {
			meta = data
			return
		},
		DataSourcesMap: map[string]*schema.Resource{
			"external": dataSource(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"external": resourceExternal(),
		},
	}
}
