// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os/exec"
	"runtime"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = (*externalDataSource)(nil)
)

func NewExternalDataSource() datasource.DataSource {
	return &externalDataSource{}
}

type externalDataSource struct{}

func (n *externalDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName
}

func (n *externalDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
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

		Attributes: map[string]schema.Attribute{
			"program": schema.ListAttribute{
				Description: "A list of strings, whose first element is the program to run and whose " +
					"subsequent elements are optional command line arguments to the program. Terraform does " +
					"not execute the program through a shell, so it is not necessary to escape shell " +
					"metacharacters nor add quotes around arguments containing spaces.",
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},

			"working_dir": schema.StringAttribute{
				Description: "Working directory of the program. If not supplied, the program will run " +
					"in the current directory.",
				Optional: true,
			},

			"query": schema.MapAttribute{
				Description: "A map of string values to pass to the external program as the query " +
					"arguments. If not supplied, the program will receive an empty object as its input.",
				ElementType: types.StringType,
				Optional:    true,
			},

			"result": schema.DynamicAttribute{
				Description: "A dynamic value returned from the external program. This can be any JSON value " +
					"including strings, numbers, booleans, lists, maps, and objects.",
				Computed: true,
			},

			"id": schema.StringAttribute{
				Description: "The id of the data source. This will always be set to `-`",
				Computed:    true,
			},
		},
	}
}

func (n *externalDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config externalDataSourceModelV0

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var program []types.String

	diags = config.Program.ElementsAs(ctx, &program, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filteredProgram := make([]string, 0, len(program))

	for _, programArgRaw := range program {
		if programArgRaw.IsNull() || programArgRaw.ValueString() == "" {
			continue
		}

		filteredProgram = append(filteredProgram, programArgRaw.ValueString())
	}

	if len(filteredProgram) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("program"),
			"External Program Missing",
			"The data source was configured without a program to execute. Verify the configuration contains at least one non-empty value.",
		)
		return
	}

	var query map[string]types.String

	diags = config.Query.ElementsAs(ctx, &query, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filteredQuery := make(map[string]string)
	for key, value := range query {
		// Preserve v2.2.3 and earlier behavior of filtering whole map elements
		// with null values.
		// Reference: https://github.com/hashicorp/terraform-provider-external/issues/208
		//
		// The external program protocol could be updated to support null values
		// as a breaking change by marshaling map[string]*string to JSON.
		// Reference: https://github.com/hashicorp/terraform-provider-external/issues/209
		if value.IsNull() {
			continue
		}

		filteredQuery[key] = value.ValueString()
	}

	queryJson, err := json.Marshal(filteredQuery)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("query"),
			"Query Handling Failed",
			"The data source received an unexpected error while attempting to parse the query. "+
				"This is always a bug in the external provider code and should be reported to the provider developers."+
				fmt.Sprintf("\n\nError: %s", err),
		)
		return
	}

	// first element is assumed to be an executable command, possibly found
	// using the PATH environment variable.
	_, err = exec.LookPath(filteredProgram[0])

	// This is a workaround to preserve pre-existing behaviour prior to the upgrade to Go 1.19.
	// Reference: https://github.com/hashicorp/terraform-provider-external/pull/192
	//
	// This workaround will be removed once a warning is being issued to notify practitioners
	// of a change in behaviour.
	// Reference: https://github.com/hashicorp/terraform-provider-external/issues/197
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("program"),
			"External Program Lookup Failed",
			"The data source received an unexpected error while attempting to parse the query. "+
				`The data source received an unexpected error while attempting to find the program.

The program must be accessible according to the platform where Terraform is running.

If the expected program should be automatically found on the platform where Terraform is running, ensure that the program is in an expected directory. On Unix-based platforms, these directories are typically searched based on the '$PATH' environment variable. On Windows-based platforms, these directories are typically searched based on the '%PATH%' environment variable.

If the expected program is relative to the Terraform configuration, it is recommended that the program name includes the interpolated value of 'path.module' before the program name to ensure that it is compatible with varying module usage. For example: "${path.module}/my-program"

The program must also be executable according to the platform where Terraform is running. On Unix-based platforms, the file on the filesystem must have the executable bit set. On Windows-based platforms, no action is typically necessary.
`+
				fmt.Sprintf("\nPlatform: %s", runtime.GOOS)+
				fmt.Sprintf("\nProgram: %s", program[0])+
				fmt.Sprintf("\nError: %s", err),
		)
		return
	}

	workingDir := config.WorkingDir.ValueString()

	cmd := exec.CommandContext(ctx, filteredProgram[0], filteredProgram[1:]...)

	// This is a workaround to preserve pre-existing behaviour prior to the upgrade to Go 1.19.
	// Reference: https://github.com/hashicorp/terraform-provider-external/pull/192
	//
	// This workaround will be removed once a warning is being issued to notify practitioners
	// of a change in behaviour.
	// Reference: https://github.com/hashicorp/terraform-provider-external/issues/197
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}

	cmd.Dir = workingDir
	cmd.Stdin = bytes.NewReader(queryJson)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	tflog.Trace(ctx, "Executing external program", map[string]interface{}{"program": cmd.String()})

	resultJson, err := cmd.Output()

	stderrStr := stderr.String()

	tflog.Trace(ctx, "Executed external program", map[string]interface{}{"program": cmd.String(), "output": string(resultJson), "stderr": stderrStr})

	if err != nil {
		if len(stderrStr) > 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("program"),
				"External Program Execution Failed",
				"The data source received an unexpected error while attempting to execute the program."+
					fmt.Sprintf("\n\nProgram: %s", cmd.Path)+
					fmt.Sprintf("\nError Message: %s", stderrStr)+
					fmt.Sprintf("\nState: %s", err),
			)
			return
		}

		resp.Diagnostics.AddAttributeError(
			path.Root("program"),
			"External Program Execution Failed",
			"The data source received an unexpected error while attempting to execute the program.\n\n"+
				"The program was executed, however it returned no additional error messaging."+
				fmt.Sprintf("\n\nProgram: %s", cmd.Path)+
				fmt.Sprintf("\nState: %s", err),
		)
		return
	}

	var result interface{}
	err = json.Unmarshal(resultJson, &result)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("program"),
			"Unexpected External Program Results",
			`The data source received unexpected results after executing the program.

Program output must be valid JSON.

If the error is unclear, the output can be viewed by enabling Terraform's logging at TRACE level. Terraform documentation on logging: https://www.terraform.io/internals/debugging
`+
				fmt.Sprintf("\nProgram: %s", cmd.Path)+
				fmt.Sprintf("\nResult Error: %s", err),
		)
		return
	}

	config.Result, diags = convertJSONToDynamic(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.ID = types.StringValue("-")

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}

// convertJSONToDynamic converts a Go interface{} value (from json.Unmarshal) to a Terraform Dynamic value
func convertJSONToDynamic(ctx context.Context, value interface{}) (types.Dynamic, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value == nil {
		return types.DynamicNull(), diags
	}

	switch v := value.(type) {
	case bool:
		return types.DynamicValue(types.BoolValue(v)), diags
	case float64:
		// Convert float64 to *big.Float for NumberValue
		bf := big.NewFloat(v)
		return types.DynamicValue(types.NumberValue(bf)), diags
	case string:
		return types.DynamicValue(types.StringValue(v)), diags
	case []interface{}:
		// Convert to tuple (heterogeneous list)
		elements := make([]attr.Value, len(v))
		elementTypes := make([]attr.Type, len(v))
		for i, elem := range v {
			dynElem, elemDiags := convertJSONToDynamic(ctx, elem)
			diags.Append(elemDiags...)
			if diags.HasError() {
				return types.DynamicNull(), diags
			}
			elements[i] = dynElem
			elementTypes[i] = types.DynamicType
		}
		tupleVal, tupleDiags := types.TupleValue(elementTypes, elements)
		diags.Append(tupleDiags...)
		if diags.HasError() {
			return types.DynamicNull(), diags
		}
		return types.DynamicValue(tupleVal), diags
	case map[string]interface{}:
		// Convert to object
		attrValues := make(map[string]attr.Value)
		attrTypes := make(map[string]attr.Type)
		for key, val := range v {
			dynVal, valDiags := convertJSONToDynamic(ctx, val)
			diags.Append(valDiags...)
			if diags.HasError() {
				return types.DynamicNull(), diags
			}
			attrValues[key] = dynVal
			attrTypes[key] = types.DynamicType
		}
		objVal, objDiags := types.ObjectValue(attrTypes, attrValues)
		diags.Append(objDiags...)
		if diags.HasError() {
			return types.DynamicNull(), diags
		}
		return types.DynamicValue(objVal), diags
	default:
		diags.AddError(
			"Unsupported JSON Type",
			fmt.Sprintf("The JSON value type %T is not supported for conversion to Terraform Dynamic type", v),
		)
		return types.DynamicNull(), diags
	}
}

type externalDataSourceModelV0 struct {
	Program    types.List    `tfsdk:"program"`
	WorkingDir types.String  `tfsdk:"working_dir"`
	Query      types.Map     `tfsdk:"query"`
	Result     types.Dynamic `tfsdk:"result"`
	ID         types.String  `tfsdk:"id"`
}
