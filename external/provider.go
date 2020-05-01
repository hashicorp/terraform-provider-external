package external

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"external": dataSource(),
		},
		ResourcesMap: map[string]*schema.Resource{},
	}
}
