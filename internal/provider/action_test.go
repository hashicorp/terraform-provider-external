// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// NOTE: These tests have a lot of hardcoded expectations for my specific machine, see TODO:Works_On_My_Machine comments.
func TestAction_raw(t *testing.T) {
	ctx := context.Background()

	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
	}

	actionTypeName := "external"
	externalProvider := GetExternalProvider(t)
	actionConfigType := GetExternalActionSchemaType(t, ctx)
	InitializeProvider(t, ctx, externalProvider)

	testCases := map[string]struct {
		config               map[string]tftypes.Value
		expectedValidateResp tfprotov5.ValidateActionConfigResponse
		expectedPlanResp     tfprotov5.PlanActionResponse
		expectedInvokeEvents []tfprotov5.InvokeActionEvent
	}{
		"test-null-program": {
			// terraform-plugin-go equivalent of:
			//
			// action "external" "test" {
			//   config {}
			// }
			//
			config: map[string]tftypes.Value{
				"program": tftypes.NewValue(
					tftypes.List{ElementType: tftypes.String},
					nil,
				),
				"query":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
				"working_dir": tftypes.NewValue(tftypes.String, nil),
			},
			expectedValidateResp: tfprotov5.ValidateActionConfigResponse{
				// Framework-defined validation error
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity: tfprotov5.DiagnosticSeverityError,
						Summary:  "Missing Configuration for Required Attribute",
						Detail: "Must set a configuration value for the program attribute as the provider has marked it as required.\n\n" +
							"Refer to the provider documentation or contact the provider developers for additional information about configurable attributes that are required.",
						Attribute: tftypes.NewAttributePath().WithAttributeName("program"),
					},
				},
			},
		},
		"test-empty-program": {
			// terraform-plugin-go equivalent of:
			//
			// action "external" "test" {
			//   config {
			//     program = []
			//   }
			// }
			//
			config: map[string]tftypes.Value{
				"program": tftypes.NewValue(
					tftypes.List{ElementType: tftypes.String},
					[]tftypes.Value{},
				),
				"query":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
				"working_dir": tftypes.NewValue(tftypes.String, nil),
			},
			expectedValidateResp: tfprotov5.ValidateActionConfigResponse{
				// Provider-defined validation error
				Diagnostics: []*tfprotov5.Diagnostic{
					{
						Severity:  tfprotov5.DiagnosticSeverityError,
						Summary:   "Invalid Attribute Value",
						Detail:    "Attribute program list must contain at least 1 elements, got: 0",
						Attribute: tftypes.NewAttributePath().WithAttributeName("program"),
					},
				},
			},
		},
		"test-go-program": {
			// terraform-plugin-go equivalent of:
			//
			// action "external" "test" {
			//   config {
			//     program = ["<programPath>", "cheese"]
			//
			//     query = {
			//       value = "pizza"
			//     }
			//   }
			// }
			//
			config: map[string]tftypes.Value{
				"program": tftypes.NewValue(
					tftypes.List{ElementType: tftypes.String},
					[]tftypes.Value{
						tftypes.NewValue(tftypes.String, programPath),
						tftypes.NewValue(tftypes.String, "cheese"),
					},
				),
				"query": tftypes.NewValue(
					tftypes.Map{ElementType: tftypes.String},
					map[string]tftypes.Value{
						"value": tftypes.NewValue(tftypes.String, "pizza"),
					},
				),
				"working_dir": tftypes.NewValue(tftypes.String, nil),
			},
			expectedPlanResp: tfprotov5.PlanActionResponse{},
			expectedInvokeEvents: []tfprotov5.InvokeActionEvent{
				{
					Type: tfprotov5.ProgressInvokeActionEventType{
						Message: `Executing external program "/Users/austin.valle/go/bin/tf-acc-external-data-source cheese"`, // TODO:Works_On_My_Machine
					},
				},
				{
					Type: tfprotov5.ProgressInvokeActionEventType{
						Message: `Output: {"argument":"cheese","query_value":"pizza","result":"yes","value":"pizza"}`,
					},
				},
				{
					Type: tfprotov5.CompletedInvokeActionEventType{},
				},
			},
		},
		"test-curl": {
			// terraform-plugin-go equivalent of:
			//
			// action "external" "test" {
			//   config {
			//     program = ["curl", "https://checkpoint-api.hashicorp.com/v1/check/terraform"]
			//   }
			// }
			//
			config: map[string]tftypes.Value{
				"program": tftypes.NewValue(
					tftypes.List{ElementType: tftypes.String},
					[]tftypes.Value{
						tftypes.NewValue(tftypes.String, "curl"),
						tftypes.NewValue(tftypes.String, "https://checkpoint-api.hashicorp.com/v1/check/terraform"),
					},
				),
				"query":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
				"working_dir": tftypes.NewValue(tftypes.String, nil),
			},
			expectedPlanResp: tfprotov5.PlanActionResponse{},
			expectedInvokeEvents: []tfprotov5.InvokeActionEvent{
				{
					Type: tfprotov5.ProgressInvokeActionEventType{
						Message: `Executing external program "/usr/bin/curl https://checkpoint-api.hashicorp.com/v1/check/terraform"`,
					},
				},
				{
					Type: tfprotov5.ProgressInvokeActionEventType{
						Message: `Output: {"product":"terraform","current_version":"1.13.3","current_release":1758116478,"current_download_url":"https://releases.hashicorp.com/terraform/1.13.3","current_changelog_url":"https://github.com/hashicorp/terraform/blob/v1.13/CHANGELOG.md","project_website":"https://www.terraform.io","alerts":[]}`,
					},
				},
				{
					Type: tfprotov5.CompletedInvokeActionEventType{},
				},
			},
		},
		"test-docker-version-error": {
			// terraform-plugin-go equivalent of:
			//
			// action "external" "test" {
			//   config {
			//     program = ["docker", "-v"]
			//   }
			// }
			//
			config: map[string]tftypes.Value{
				"program": tftypes.NewValue(
					tftypes.List{ElementType: tftypes.String},
					[]tftypes.Value{
						tftypes.NewValue(tftypes.String, "docker"),
						tftypes.NewValue(tftypes.String, "-v"),
					},
				),
				"query":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
				"working_dir": tftypes.NewValue(tftypes.String, nil),
			},
			expectedPlanResp: tfprotov5.PlanActionResponse{},
			expectedInvokeEvents: []tfprotov5.InvokeActionEvent{
				{
					Type: tfprotov5.ProgressInvokeActionEventType{
						Message: `Executing external program "/usr/local/bin/docker -v"`, // TODO:Works_On_My_Machine
					},
				},
				{
					Type: tfprotov5.ProgressInvokeActionEventType{
						Message: "Output: Docker version 28.3.2, build 578ccf6\n", // TODO:Works_On_My_Machine
					},
				},
				{
					Type: tfprotov5.CompletedInvokeActionEventType{},
				},
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testProgramConfig, err := tfprotov5.NewDynamicValue(actionConfigType, tftypes.NewValue(actionConfigType, tc.config))
			if err != nil {
				t.Fatal(err)
			}

			validateResp, err := externalProvider.ValidateActionConfig(ctx, &tfprotov5.ValidateActionConfigRequest{
				ActionType: actionTypeName,
				Config:     &testProgramConfig,
			})
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(*validateResp, tc.expectedValidateResp); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}

			// Don't plan/invoke if we had diagnostics during validate
			if len(validateResp.Diagnostics) > 0 {
				return
			}

			planResp, err := externalProvider.PlanAction(ctx, &tfprotov5.PlanActionRequest{
				ActionType: actionTypeName,
				Config:     &testProgramConfig,
			})
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(*planResp, tc.expectedPlanResp); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}

			// Don't invoke if we had diagnostics during plan
			if len(planResp.Diagnostics) > 0 {
				return
			}

			invokeResp, err := externalProvider.InvokeAction(ctx, &tfprotov5.InvokeActionRequest{
				ActionType: actionTypeName,
				Config:     &testProgramConfig,
			})
			if err != nil {
				t.Fatal(err)
			}

			// Grab all the events
			events := slices.Collect(invokeResp.Events)
			if diff := cmp.Diff(events, tc.expectedInvokeEvents); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestAction_real(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "terraform_data" "fake_resource" {
  input = "fake-string"

  lifecycle {
    action_trigger {
      events  = [before_create]
      actions = [action.external.test_pizza]
    }
  }
}

action "external" "test_pizza" {
  config {
    program = ["%s", "cheese"]
    query = {
      value = "pizza"
    }
  }
}`, programPath),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("terraform_data.fake_resource", plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("terraform_data.fake_resource", tfjsonpath.New("output"), knownvalue.StringExact("fake-string")),
				},
			},
		},
	})
}

func GetExternalProvider(t *testing.T) tfprotov5.ProviderServerWithActions {
	t.Helper()

	provider, err := providerserver.NewProtocol5WithError(New())()
	if err != nil {
		t.Fatal(err)
	}

	return provider.(tfprotov5.ProviderServerWithActions)
}

func GetExternalActionSchemaType(t *testing.T, ctx context.Context) tftypes.Type {
	t.Helper()

	actionSchemaResp := action.SchemaResponse{}
	NewExternalAction().Schema(ctx, action.SchemaRequest{}, &actionSchemaResp)

	return actionSchemaResp.Schema.Type().TerraformType(ctx)
}

func InitializeProvider(t *testing.T, ctx context.Context, externalProvider tfprotov5.ProviderServer) {
	t.Helper()

	// Just for show in our case here, since the external provider isn't muxed, so technically you could skip this.
	// Mimicking what core will eventually do.
	schemaResp, err := externalProvider.GetProviderSchema(ctx, &tfprotov5.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}

	// NOTE: Since we already have the schema/type in the same codebase, we don't need to read the schema
	// so this is just asserting that GetProviderSchema is actually returning the action schema as expected.
	if len(schemaResp.ActionSchemas) == 0 {
		t.Fatalf("expected to find action schemas and didn't find any!")
	}

	nullObject, err := tfprotov5.NewDynamicValue(tftypes.Object{}, tftypes.NewValue(tftypes.Object{}, nil))
	if err != nil {
		t.Fatal(err)
	}

	// Just for show in our case here, since the external provider has no configuration.
	// Mimicking what core will eventually do.
	_, err = externalProvider.ConfigureProvider(ctx, &tfprotov5.ConfigureProviderRequest{
		Config: &nullObject,
	},
	)
	if err != nil {
		t.Fatal(err)
	}
}
