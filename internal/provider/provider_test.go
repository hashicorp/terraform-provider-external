package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProvider(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

var testProviders = map[string]*schema.Provider{
	"external": New(),
}

func TestMain(m *testing.M) {
	acctest.UseBinaryDriver("external", New)
	resource.TestMain(m)
}
