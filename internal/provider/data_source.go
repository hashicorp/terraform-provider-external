package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSource() *schema.Resource {
	return &schema.Resource{
		Description: "The `external` data source allows an external program implementing a specific protocol " +
			"(defined below) to act as a data source, exposing arbitrary data for use elsewhere in the Terraform " +
			"configuration.\n" +
			"\n" +
			"**Warning** This mechanism is provided as an \"escape hatch\" for exceptional situations where a " +
			"first-class Terraform provider is not more appropriate. Its capabilities are limited in comparison " +
			"to a true data source, and implementing a data source via an external program is likely to hurt the " +
			"portability of your Terraform configuration by creating dependencies on external programs and " +
			"libraries that may not be available (or may need to be used differently) on different operating " +
			"systems.\n" +
			"\n" +
			"**Warning** Terraform Enterprise does not guarantee availability of any particular language runtimes " +
			"or external programs beyond standard shell utilities, so it is not recommended to use this data source " +
			"within configurations that are applied within Terraform Enterprise.",

		ReadContext: dataSourceRead,

		Schema: map[string]*schema.Schema{
			"program": {
				Description: "A list of strings, whose first element is the program to run and whose " +
					"subsequent elements are optional command line arguments to the program. Terraform does " +
					"not execute the program through a shell, so it is not necessary to escape shell " +
					"metacharacters nor add quotes around arguments containing spaces.",
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				MinItems: 1,
			},

			"working_dir": {
				Description: "Working directory of the program. If not supplied, the program will run " +
					"in the current directory.",
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"query": {
				Description: "A map of string values to pass to the external program as the query " +
					"arguments. If not supplied, the program will receive an empty object as its input.",
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"result": {
				Description: "A map of string values returned from the external program.",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	programI := d.Get("program").([]interface{})
	workingDir := d.Get("working_dir").(string)
	query := d.Get("query").(map[string]interface{})

	program := make([]string, 0, len(programI))

	for _, programArgRaw := range programI {
		programArg, ok := programArgRaw.(string)

		if !ok || programArg == "" {
			continue
		}

		program = append(program, programArg)
	}

	if len(program) == 0 {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "External Program Missing",
				Detail:        "The data source was configured without a program to execute. Verify the configuration contains at least one non-empty value.",
				AttributePath: cty.GetAttrPath("program"),
			},
		}
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Query Handling Failed",
				Detail: "The data source received an unexpected error while attempting to parse the query. " +
					"This is always a bug in the external provider code and should be reported to the provider developers." +
					fmt.Sprintf("\n\nError: %s", err),
				AttributePath: cty.GetAttrPath("query"),
			},
		}
	}

	// first element is assumed to be an executable command, possibly found
	// using the PATH environment variable.
	_, err = exec.LookPath(program[0])

	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "External Program Lookup Failed",
				Detail: `The data source received an unexpected error while attempting to find the program.

The program must be accessible according to the platform where Terraform is running.

If the expected program should be automatically found on the platform where Terraform is running, ensure that the program is in an expected directory. On Unix-based platforms, these directories are typically searched based on the '$PATH' environment variable. On Windows-based platforms, these directories are typically searched based on the '%PATH%' environment variable.

If the expected program is relative to the Terraform configuration, it is recommended that the program name includes the interpolated value of 'path.module' before the program name to ensure that it is compatible with varying module usage. For example: "${path.module}/my-program"

The program must also be executable according to the platform where Terraform is running. On Unix-based platforms, the file on the filesystem must have the executable bit set. On Windows-based platforms, no action is typically necessary.
` +
					fmt.Sprintf("\nPlatform: %s", runtime.GOOS) +
					fmt.Sprintf("\nProgram: %s", program[0]) +
					fmt.Sprintf("\nError: %s", err),
				AttributePath: cty.GetAttrPath("program"),
			},
		}
	}

	cmd := exec.CommandContext(ctx, program[0], program[1:]...)
	cmd.Dir = workingDir
	cmd.Stdin = bytes.NewReader(queryJson)

	tflog.Trace(ctx, "Executing external program", "program", cmd.String())

	resultJson, err := cmd.Output()

	tflog.Trace(ctx, "Executed external program", "program", cmd.String(), "output", string(resultJson))

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.Stderr != nil && len(exitErr.Stderr) > 0 {
				return diag.Diagnostics{
					{
						Severity: diag.Error,
						Summary:  "External Program Execution Failed",
						Detail: "The data source received an unexpected error while attempting to execute the program." +
							fmt.Sprintf("\n\nProgram: %s", cmd.Path) +
							fmt.Sprintf("\nError Message: %s", string(exitErr.Stderr)) +
							fmt.Sprintf("\nState: %s", err),
						AttributePath: cty.GetAttrPath("program"),
					},
				}
			}

			return diag.Diagnostics{
				{
					Severity: diag.Error,
					Summary:  "External Program Execution Failed",
					Detail: "The data source received an unexpected error while attempting to execute the program.\n\n" +
						"The program was executed, however it returned no additional error messaging." +
						fmt.Sprintf("\n\nProgram: %s", cmd.Path) +
						fmt.Sprintf("\nState: %s", err),
					AttributePath: cty.GetAttrPath("program"),
				},
			}
		}

		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "External Program Execution Failed",
				Detail: "The data source received an unexpected error while attempting to execute the program." +
					fmt.Sprintf("\n\nProgram: %s", cmd.Path) +
					fmt.Sprintf("\nError: %s", err),
				AttributePath: cty.GetAttrPath("program"),
			},
		}
	}

	result := map[string]string{}
	err = json.Unmarshal(resultJson, &result)
	if err != nil {
		return diag.Diagnostics{
			{
				Severity: diag.Error,
				Summary:  "Unexpected External Program Results",
				Detail: `The data source received unexpected results after executing the program.

Program output must be a JSON encoded map of string keys and string values.

If the error is unclear, the output can be viewed by enabling Terraform's logging at TRACE level. Terraform documentation on logging: https://www.terraform.io/internals/debugging
` +
					fmt.Sprintf("\nProgram: %s", cmd.Path) +
					fmt.Sprintf("\nResult Error: %s", err),
				AttributePath: cty.GetAttrPath("program"),
			},
		}
	}

	d.Set("result", result)

	d.SetId("-")
	return nil
}
