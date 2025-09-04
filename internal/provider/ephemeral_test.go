// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testEphemeralExternalConfig_basic = `
ephemeral "external" "test" {
  program = ["%s", "cheese"]

  query = {
    value = "pizza"
    output_file = "%s"
  }
}
`

func TestEphemeralExternal_basic(t *testing.T) {
	programPath, err := buildEphemeralTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "result.json")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testEphemeralExternalConfig_basic, programPath, outputFile),
				Check: func(_ *terraform.State) error {
					return validateEphemeralOutput(outputFile, map[string]string{
						"result":      "yes",
						"query_value": "pizza",
						"argument":    "cheese",
						"value":       "pizza",
						"output_file": outputFile,
					})
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
	programPath, err := buildEphemeralTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	resource.UnitTest(t, resource.TestCase{
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
    working_dir_file = "%s"
  }
}
`

func TestEphemeralExternal_workingDirectory(t *testing.T) {
	programPath, err := buildEphemeralTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	tempDir := t.TempDir()
	workingDirFile := filepath.Join(tempDir, "working_dir.txt")
	workingDir := "/tmp"

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testEphemeralExternalConfig_workingDir, programPath, workingDir, workingDirFile),
				Check: func(_ *terraform.State) error {
					return validateWorkingDirectory(workingDirFile, workingDir)
				},
			},
		},
	})
}

const testEphemeralExternalConfig_emptyQuery = `
ephemeral "external" "test" {
  program = ["%s"]

  query = {
    output_file = "%s"
  }
}
`

func TestEphemeralExternal_emptyQuery(t *testing.T) {
	programPath, err := buildEphemeralTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "result.json")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testEphemeralExternalConfig_emptyQuery, programPath, outputFile),
				Check: func(_ *terraform.State) error {
					return validateEphemeralOutput(outputFile, map[string]string{
						"result":      "yes",
						"output_file": outputFile,
					})
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
	programPath, err := buildEphemeralTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "result.json")

	resource.UnitTest(t, resource.TestCase{
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
							output_file = %[2]q
						}
					}
				`, programPath, outputFile),
				Check: func(_ *terraform.State) error {
					return validateEphemeralOutput(outputFile, map[string]string{
						"result":      "yes",
						"query_value": "valuetest",
						"value":       "valuetest",
						"output_file": outputFile,
					})
				},
			},
		},
	})
}

func TestEphemeralExternal_Program_EmptyStringAndNullValues(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
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
	programPath, err := buildEphemeralTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "result.json")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					ephemeral "external" "test" {
						program = [%[1]q]
				
						query = {
							value = ""
							output_file = %[2]q
						}
					}
				`, programPath, outputFile),
				Check: func(_ *terraform.State) error {
					return validateEphemeralOutput(outputFile, map[string]string{
						"result":      "yes",
						"query_value": "",
						"value":       "",
						"output_file": outputFile,
					})
				},
			},
		},
	})
}

func TestEphemeralExternal_Query_NullElementValue(t *testing.T) {
	programPath, err := buildEphemeralTestProgram()
	if err != nil {
		t.Fatal(err)
		return
	}

	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "result.json")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: protoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				ephemeral "external" "test" {
				  program = [%[1]q]
				
				  query = {
					# Program will return exit status 1 if the "fail" key is present.
					fail = null
					output_file = %[2]q
				  }
				}
				`, programPath, outputFile),
				Check: func(_ *terraform.State) error {
					// Validate that the program ran successfully and null values were filtered
					data, err := os.ReadFile(outputFile)
					if err != nil {
						return fmt.Errorf("failed to read output file: %v", err)
					}

					var result map[string]string
					if err := json.Unmarshal(data, &result); err != nil {
						return fmt.Errorf("failed to parse JSON: %v", err)
					}

					// The "fail" key should not be present due to null filtering
					if _, exists := result["fail"]; exists {
						return fmt.Errorf("unexpected 'fail' key in result, null values should be filtered")
					}

					return nil
				},
			},
		},
	})
}

func buildEphemeralTestProgram() (string, error) {
	// We have a simple Go program that we use as a stub for testing.
	cmd := exec.Command(
		"go", "install",
		"github.com/terraform-providers/terraform-provider-external/internal/provider/test-programs/tf-acc-external-ephemeral",
	)
	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf("failed to build test stub program: %s", err)
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME") + "/go")
	}

	programPath := path.Join(
		filepath.SplitList(gopath)[0], "bin", "tf-acc-external-ephemeral",
	)
	return programPath, nil
}

func validateEphemeralOutput(outputFile string, expectedResults map[string]string) error {
	data, err := os.ReadFile(outputFile)
	if err != nil {
		return fmt.Errorf("failed to read output file: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	for key, expectedValue := range expectedResults {
		actualValue, exists := result[key]
		if !exists {
			return fmt.Errorf("missing key '%s' in result", key)
		}
		if actualValue != expectedValue {
			return fmt.Errorf("key '%s': expected '%s', got '%s'", key, expectedValue, actualValue)
		}
	}

	return nil
}

func validateWorkingDirectory(workingDirFile, expectedWorkingDir string) error {
	data, err := os.ReadFile(workingDirFile)
	if err != nil {
		return fmt.Errorf("failed to read working directory file: %v", err)
	}

	actualWorkingDir := string(data)
	if actualWorkingDir != expectedWorkingDir {
		return fmt.Errorf("working directory: expected '%s', got '%s'", expectedWorkingDir, actualWorkingDir)
	}

	return nil
}
