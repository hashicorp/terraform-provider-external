---
layout: "external"
page_title: "External: external_resource"
sidebar_current: "docs-external-resource"
description: |-
  Sample resource in the Terraform provider external.
---

# External Resource

The `external` resource allows a set of (3 to 4) external programs
implementing a specific protocol (defined below) to act as
a fully-fledged Terraform resource.

~> **Warning** This mechanism is provided as an "escape hatch" for exceptional
situations where a first-class Terraform provider is not more appropriate.
Its capabilities are limited in comparison to a true data source, and
implementing a resource via an external program is likely to hurt the
portability of your Terraform configuration by creating dependencies on
external programs and libraries that may not be available (or may need to
be used differently) on different operating systems.

~> **Warning** Terraform Enterprise does not guarantee availability of any
particular language runtimes or external programs beyond standard shell
utilities, so it is not recommended to use this data source within
configurations that are applied within Terraform Enterprise.


## External Program Protocol

1. Program execution environment:

    1. Program executes in current working directory
        ([`${path.cwd}`](https://www.terraform.io/docs/configuration/expressions.html#path-cwd)). 

    1. Program receives temporary directory path in 
        `${TF_EXTERNAL_DIR}`/`${TF_EXTERNAL_DIR_ABS}` environment variables.

1. Program interacts with files named after resource attributes
    plus `id` and `old_state` files.

    1. Program reads data from `input`/`input_sensitive`
    
    1. Program must fill-in `id` during create and update 
        representing current and future values of `external_resource.*.id`:
    
        1. Program empties `id` (eg. `echo -n > "${TF_EXTERNAL_DIR}/id"`)
            during read when the resource disappeared (externally).
    
    1. Program stores the managed data in `state`:
    
        1. Program can read previous version of managed data from `old_state`,
        2. Program should write to `state` during read,
    
    1. Program writes to `output`/`output_sensitive` to expose additional data.

### Files summary

Following files will be created in directory defined by 
`${TF_EXTERNAL_DIR}`/`${TF_EXTERNAL_DIR_ABS}`:
- `input` - (read-only) - inputs passed down from the resource's attribute,
    see arguments reference
- `input_sensitive` - (read-only) - inputs passed down from 
    resource's attribute, see arguments reference
- `id` - (read-write) - holds the current (and future if changed)
    value of `id` attribute,
    resource will be considered non-existing if empty,
- `state` - (read-write) - reflects current (and future if changed)
    managed state of the world, see arguments reference
- `old_state` - (read-only) - reflects previous state of the world,
- `output` - (write-only) - holds current unmanaged state of the world,
    see arguments reference
- `output_sensitive` - (write-only) - holds current (unmanaged)
    state of the world, see arguments reference

## Example Usage

```hcl
locals {
  script = <<EOT
  set -xeuo pipefail

  main() {
	"cmd_$@"
  }

  cmd_update() {
	file_name="$(cat "$TF_EXTERNAL_DIR/input" | tee "$TF_EXTERNAL_DIR/id" "$TF_EXTERNAL_DIR/output")"
	cat "$TF_EXTERNAL_DIR/state" | tee "$TF_EXTERNAL_DIR/state" > "$file_name"
  }

  cmd_read() {
	file_name="$(cat "$TF_EXTERNAL_DIR/input")"
	cat "$file_name"
	cat "$TF_EXTERNAL_DIR/state"
	echo -n "$file_name" > "$TF_EXTERNAL_DIR/output"
	cat "$file_name" > "$TF_EXTERNAL_DIR/state"
  }
  
  cmd_delete() {
	rm "$(cat "$TF_EXTERNAL_DIR/input")"
  }

  main "$@"
  EOT

  program = ["sh", "-c", local.script, "command_string"]
}

resource "external" "foo" {
  input = "/tmp/terraform-provider-external_resource_test"
  state = "qwe"

  program_create = concat(local.program, ["update"])
  program_read = concat(local.program, ["read"])
  program_update = concat(local.program, ["update"])
  program_delete = concat(local.program, ["delete"])
}
```

## Argument Reference

The following arguments are supported:

* `program_create` - (Optional) program to run on `create` operation, 
    runs `program_update` instead if not provided. 
    Be sure to write to `${TF_EXTERNAL_DIR}/id`.
* `program_read` - (Required) program to run on `read` operation. 
    It should update `${TF_EXTERNAL_DIR}/state` to reflect world's state.
    Emptying `${TF_EXTERNAL_DIR}/id` will inform Terraform that resource
    does not exist anymore.
* `program_update` - (Required) program to run on `update` 
    (and optionally `create`) operations.
    It should update `${TF_EXTERNAL_DIR}/state` to reflect world's state.
* `program_delete` - (Required) program to run on `destroy` operation.
* `input` - (Optional) (`${TF_EXTERNAL_DIR}/input` read-only) 
    unmanaged/to-be-interpolated parts of resource's desired state
* `input_sensitive` - (Optional) 
    (`${TF_EXTERNAL_DIR}/input_sensitive` read-only) same as `input`, 
    but the content won't be printed during planning.
* `state` - (Optional) (`${TF_EXTERNAL_DIR}/state` read-write and
    `${TF_EXTERNAL_DIR}/old_state` read-only) managed parts of
    resource's real state, it should be written to 
    (during course of `create`, `read` and `update` commands)
    to reflect current state of the world.

## Attributes Reference

The following attributes are exported:

* `output` - (`${TF_EXTERNAL_DIR}/output` write-only)
    additional (relative to `state` attribute) data
    the resource is providing.
* `output_sensitive` - (`${TF_EXTERNAL_DIR}/output_sensitive` write-only) 
    same as `output`, but content won't be printed during planning.

## Resource design

The resource's primary goal is to be *quickly finished*
as a feature-complete MVP (Minimal Viable Product)
enabling Terraform developers to define custom logic.

Below design decisions are helping with above:

1. don't introduce new nice to have features:
    - less code to maintain in the provider,
    - most of those can be provided by wrapping
        resources in modules using new Terraform 0.12/0.13+ features

1. don't impose any code structure on the scripts:
    - just run the user-provided arguments list as-is
 
1. plain-text (`string` type) resource attributes:
    - use Terraform/HCL features to handle them as structured data,
        eg: `jsonencode()` and `jsondecode()` pair,
    - provider users are free to handle the data as they see fit,
    - only plumbing required to share the data with the `program`,

1. interface with the `program` through files in a temporary directory:
    - temporary directory path is exposed to the program through
        `${TF_EXTERNAL_DIR}`/`${TF_EXTERNAL_DIR_ABS}` environment variables,
    - filesystem permissions reflect what can be done with them,

1. attribute names map directly to file names:
    - so we don't need to pass anything other than
        temporary directory location to the `program`,

1. only one way to pass the data down to `program`:
    - through string attributes mapped to files,
    - environment variables are NOT configurable,
        if you really need them you can 
        `source ${TF_EXTERNAL_DIR}/input_sensitive` in a shell program,

1. well defined `program` interface files
