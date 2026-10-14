// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	// EnvTfAccExternalTimeoutTest is the name of the environment variable used
	// to enable the 20 minute timeout test. The environment variable can be
	// set to any value to enable the test.
	EnvTfAccExternalTimeoutTest = "TF_ACC_EXTERNAL_TIMEOUT_TEST"
)

const testDataSourceConfig_basic = `
data "external" "test" {
  program = ["%s", "cheese"]

  query = {
    value = "pizza"
  }
}

output "query_value" {
  value = "${data.external.test.result["query_value"]}"
}

output "argument" {
  value = "${data.external.test.result["argument"]}"
}
`

func TestDataSource_basic(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_basic, programPath),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.external.test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["argument"] == nil {
						return fmt.Errorf("missing 'argument' output")
					}
					if outputs["query_value"] == nil {
						return fmt.Errorf("missing 'query_value' output")
					}

					if outputs["argument"].Value != "cheese" {
						return fmt.Errorf(
							"'argument' output is %q; want 'cheese'",
							outputs["argument"].Value,
						)
					}
					if outputs["query_value"].Value != "pizza" {
						return fmt.Errorf(
							"'query_value' output is %q; want 'pizza'",
							outputs["query_value"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

const testDataSourceConfig_error = `
data "external" "test" {
  program = ["%s"]

  query = {
    fail = "true"
  }
}
`

func TestDataSource_error(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testDataSourceConfig_error, programPath),
				ExpectError: regexp.MustCompile("I was asked to fail"),
			},
		},
	})
}

// Reference: https://github.com/hashicorp/terraform-provider-external/issues/110
func TestDataSource_Program_OnlyEmptyString(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
					data "external" "test" {
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

// Reference: https://github.com/hashicorp/terraform-provider-external/issues/110
func TestDataSource_Program_PathAndEmptyString(t *testing.T) {
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
					data "external" "test" {
						program = [
							%[1]q,
							"", # e.g. a variable that became empty
						]
				
						query = {
							value = "valuetest"
						}
					}
				`, programPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.external.test", "result.query_value", "valuetest"),
				),
			},
		},
	})
}

func TestDataSource_Program_EmptyStringAndNullValues(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
					data "external" "test" {
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

func TestDataSource_Query_EmptyElementValue(t *testing.T) {
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
					data "external" "test" {
						program = [%[1]q]
				
						query = {
							value = "",
						}
					}
				`, programPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.external.test", "result.value", ""),
				),
			},
		},
	})
}

// Reference: https://github.com/hashicorp/terraform-provider-external/issues/208
func TestDataSource_Query_NullElementValue(t *testing.T) {
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
				data "external" "test" {
				  program = [%q]
				
				  query = {
					# Program will return exit status 1 if the "fail" key is present.
					fail = null
				  }
				}
				`, programPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("data.external.test", "result.fail"),
				),
			},
		},
	})
}

func TestDataSource_CurrentDir(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot create get current working dir: %s", err)
	}

	tempDir := t.TempDir()

	binaryName := "tf-acc-external-data-source"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	err = os.Rename(filepath.FromSlash(programPath), filepath.Join(tempDir, binaryName))
	if err != nil {
		t.Fatalf("cannot move tf-acc-external-data-source from go bin to temp dir: %s", err)
	}

	tempDirRel, err := filepath.Rel(wd, tempDir)
	if err != nil {
		t.Fatalf("could not obtain relative directory: %s", err)
	}

	p := os.Getenv("PATH")
	t.Setenv("PATH", fmt.Sprintf("%s%c%s", p, os.PathListSeparator, tempDirRel))

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "external" "test" {
						program = [%[1]q]
				
						query = {
							value = "test",
						}
					}
				`, binaryName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.external.test", "result.value", "test"),
				),
			},
		},
	})
}

// Reference: https://github.com/hashicorp/terraform-provider-external/issues/197
func TestExternalDataSource_Read_currentDirWarning(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get current working dir: %s", err)
	}

	tempDir := t.TempDir()
	binaryName := "tf-acc-external-data-source"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	err = os.Rename(filepath.FromSlash(programPath), filepath.Join(tempDir, binaryName))
	if err != nil {
		t.Fatalf("cannot move binary to temp dir: %s", err)
	}

	tempDirRel, err := filepath.Rel(wd, tempDir)
	if err != nil {
		t.Fatalf("could not obtain relative directory: %s", err)
	}

	p := os.Getenv("PATH")
	t.Setenv("PATH", fmt.Sprintf("%s%c%s", p, os.PathListSeparator, tempDirRel))

	ds := NewExternalDataSource()
	var schemaResp datasource.SchemaResponse
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)

	cfgVal := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"program":     tftypes.List{ElementType: tftypes.String},
			"working_dir": tftypes.String,
			"query":       tftypes.Map{ElementType: tftypes.String},
			"result":      tftypes.Map{ElementType: tftypes.String},
			"id":          tftypes.String,
		},
	}, map[string]tftypes.Value{
		"program": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, binaryName),
		}),
		"working_dir": tftypes.NewValue(tftypes.String, nil),
		"query":       tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{}),
		"result":      tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil),
		"id":          tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Raw:    cfgVal,
			Schema: schemaResp.Schema,
		},
	}
	resp := datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %s", resp.Diagnostics.Errors())
	}

	warnings := resp.Diagnostics.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}

	if warnings[0].Summary() != "Executing Program in Current Directory Deprecated" {
		t.Errorf("unexpected warning summary: %s", warnings[0].Summary())
	}
}

func TestDataSource_upgrade(t *testing.T) {
	programPath, err := buildDataSourceTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: providerVersion223(),
				Config:            fmt.Sprintf(testDataSourceConfig_basic, programPath),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.external.test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["argument"] == nil {
						return fmt.Errorf("missing 'argument' output")
					}
					if outputs["query_value"] == nil {
						return fmt.Errorf("missing 'query_value' output")
					}

					if outputs["argument"].Value != "cheese" {
						return fmt.Errorf(
							"'argument' output is %q; want 'cheese'",
							outputs["argument"].Value,
						)
					}
					if outputs["query_value"].Value != "pizza" {
						return fmt.Errorf(
							"'query_value' output is %q; want 'pizza'",
							outputs["query_value"].Value,
						)
					}

					return nil
				},
			},
			{
				ProtoV5ProviderFactories: protoV5ProviderFactories(),
				Config:                   fmt.Sprintf(testDataSourceConfig_basic, programPath),
				PlanOnly:                 true,
			},
			{
				ProtoV5ProviderFactories: protoV5ProviderFactories(),
				Config:                   fmt.Sprintf(testDataSourceConfig_basic, programPath),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.external.test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["argument"] == nil {
						return fmt.Errorf("missing 'argument' output")
					}
					if outputs["query_value"] == nil {
						return fmt.Errorf("missing 'query_value' output")
					}

					if outputs["argument"].Value != "cheese" {
						return fmt.Errorf(
							"'argument' output is %q; want 'cheese'",
							outputs["argument"].Value,
						)
					}
					if outputs["query_value"].Value != "pizza" {
						return fmt.Errorf(
							"'query_value' output is %q; want 'pizza'",
							outputs["query_value"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

func buildDataSourceTestProgram() (string, error) {
	binaryName := "tf-acc-external-data-source"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	// We have a simple Go program that we use as a stub for testing.
	cmd := exec.Command(
		"go", "install",
		"github.com/terraform-providers/terraform-provider-external/internal/provider/test-programs/tf-acc-external-data-source",
	)
	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("failed to build test stub program: %s", err)
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine user home directory: %s", err)
		}
		gopath = filepath.Join(home, "go")
	}

	gobin := os.Getenv("GOBIN")
	if gobin == "" {
		gobin = filepath.Join(filepath.SplitList(gopath)[0], "bin")
	}

	programPath := filepath.Join(gobin, binaryName)
	return filepath.ToSlash(programPath), nil
}

// Reference: https://github.com/hashicorp/terraform-provider-external/issues/145
func TestDataSource_20MinuteTimeout(t *testing.T) {
	if os.Getenv(EnvTfAccExternalTimeoutTest) == "" {
		t.Skipf("Skipping this test since the %s environment variable is not set to any value. "+
			"This test requires 20 minutes to run, so it is disabled by default.",
			EnvTfAccExternalTimeoutTest,
		)
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
					data "external" "test" {
						program = ["sleep", "1205"] # over 20 minutes
					}
				`,
				// Not External Program Execution Failed / State: signal: killed
				ExpectError: regexp.MustCompile(`Unexpected External Program Results`),
			},
		},
	})
}
