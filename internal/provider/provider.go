package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func New() *schema.Provider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"external":           dataSource(),
			"external_sensitive": dataSourceSensitive(),
		},
		ResourcesMap: map[string]*schema.Resource{},
	}
}
