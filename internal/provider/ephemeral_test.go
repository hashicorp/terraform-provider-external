// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const testEphemeralExternalConfig_basic = `
ephemeral "external" "test" {
  program = ["%s", "cheese"]

  query = {
    value = "pizza"
  }
}

provider "echo" {
  data = ephemeral.external.test.result
}

resource "echo" "test" {}
`

func TestEphemeralExternal_basic(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"echo": echoprovider.NewProviderServer(),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testEphemeralExternalConfig_basic, programPath),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("result"),
						knownvalue.StringExact("yes"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("query_value"),
						knownvalue.StringExact("pizza"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("argument"),
						knownvalue.StringExact("cheese"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("value"),
						knownvalue.StringExact("pizza"),
					),
				},
			},
		},
	})
}

const testEphemeralExternalConfig_error = `
ephemeral "external" "test" {
  program = ["%s"]

  query = {
    fail = "true"
  }
}
`

func TestEphemeralExternal_error(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testEphemeralExternalConfig_error, programPath),
				ExpectError: regexp.MustCompile("I was asked to fail"),
			},
		},
	})
}

const testEphemeralExternalConfig_workingDir = `
ephemeral "external" "test" {
  program = ["%s"]
  working_dir = "%s"

  query = {
    value = "test"
  }
}

provider "echo" {
  data = ephemeral.external.test.result
}

resource "echo" "test" {}
`

func TestEphemeralExternal_workingDirectory(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	workingDir := "/tmp"

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"echo": echoprovider.NewProviderServer(),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testEphemeralExternalConfig_workingDir, programPath, workingDir),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("working_dir"),
						knownvalue.StringExact(workingDir),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("result"),
						knownvalue.StringExact("yes"),
					),
				},
			},
		},
	})
}

const testEphemeralExternalConfig_emptyQuery = `
ephemeral "external" "test" {
  program = ["%s"]

  query = {}
}

provider "echo" {
  data = ephemeral.external.test.result
}

resource "echo" "test" {}
`

func TestEphemeralExternal_emptyQuery(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"echo": echoprovider.NewProviderServer(),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testEphemeralExternalConfig_emptyQuery, programPath),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("result"),
						knownvalue.StringExact("yes"),
					),
				},
			},
		},
	})
}

const testEphemeralExternalConfig_missingProgram = `
ephemeral "external" "test" {
  program = []
}
`

func TestEphemeralExternal_missingProgram(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testEphemeralExternalConfig_missingProgram,
				ExpectError: regexp.MustCompile("Invalid Attribute Value"),
			},
		},
	})
}

func TestEphemeralExternal_Program_OnlyEmptyString(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
					ephemeral "external" "test" {
						program = [
							"", # e.g. a variable that became empty
						]

						query = {
							value = "valuetest"
						}
					}
				`,
				ExpectError: regexp.MustCompile(`External Program Missing`),
			},
		},
	})
}

func TestEphemeralExternal_Program_PathAndEmptyString(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"echo": echoprovider.NewProviderServer(),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					ephemeral "external" "test" {
						program = [
							%[1]q,
							"", # e.g. a variable that became empty
						]

						query = {
							value = "valuetest"
						}
					}

					provider "echo" {
					  data = ephemeral.external.test.result
					}

					resource "echo" "test" {}
				`, programPath),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("result"),
						knownvalue.StringExact("yes"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("query_value"),
						knownvalue.StringExact("valuetest"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("value"),
						knownvalue.StringExact("valuetest"),
					),
				},
			},
		},
	})
}

func TestEphemeralExternal_Program_EmptyStringAndNullValues(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
					ephemeral "external" "test" {
						program = [
							null, "", # e.g. a variable that became empty
						]

						query = {
							value = "valuetest"
						}
					}
				`,
				ExpectError: regexp.MustCompile(`External Program Missing`),
			},
		},
	})
}

func TestEphemeralExternal_Query_EmptyElementValue(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"echo": echoprovider.NewProviderServer(),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					ephemeral "external" "test" {
						program = [%[1]q]

						query = {
							value = ""
						}
					}

					provider "echo" {
					  data = ephemeral.external.test.result
					}

					resource "echo" "test" {}
				`, programPath),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("result"),
						knownvalue.StringExact("yes"),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("query_value"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("value"),
						knownvalue.StringExact(""),
					),
				},
			},
		},
	})
}

func TestEphemeralExternal_Query_NullElementValue(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_10_0),
		},
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"echo": echoprovider.NewProviderServer(),
		},
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				ephemeral "external" "test" {
				  program = [%[1]q]

				  query = {
					# Program will return exit status 1 if the "fail" key is present.
					fail = null
				  }
				}

				provider "echo" {
				  data = ephemeral.external.test.result
				}

				resource "echo" "test" {}
				`, programPath),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"echo.test",
						tfjsonpath.New("data").AtMapKey("result"),
						knownvalue.StringExact("yes"),
					),
					// The test passes by successfully running without the "fail" key
					// causing the external program to exit with status 1
				},
			},
		},
	})
}
