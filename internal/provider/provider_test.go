// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func protoV5ProviderFactories() map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){
		"external": providerserver.NewProtocol5WithError(New()),
	}
}

func providerVersion223() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"external": {
			VersionConstraint: "2.2.3",
			Source:            "hashicorp/external",
		},
	}
}
