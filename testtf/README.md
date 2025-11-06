# Terraform External Provider - Dynamic Data Types Example

This directory demonstrates the enhanced external provider that supports dynamic data types, allowing external programs to return any JSON structure.

## Files

- **example-data-source.sh**: A bash script that returns complex JSON with various data types (strings, numbers, booleans, arrays, objects)
- **main.tf**: Terraform configuration demonstrating how to use the external data source and access dynamic fields

## Prerequisites

- Terraform installed
- `jq` command-line JSON processor installed
- The enhanced external provider (built from this repository)

## Testing with the Local Build

To test with your locally built provider:

1. Build the provider:
   ```bash
   cd ..
   go build -o terraform-provider-external
   ```

2. The provider override configuration is already set up in `.terraformrc` in this directory.

3. Use the helper script to run Terraform commands:
   ```bash
   ./run-terraform.sh validate
   ./run-terraform.sh plan
   ./run-terraform.sh apply
   ```

   Or set the environment variable manually:
   ```bash
   export TF_CLI_CONFIG_FILE="$(pwd)/.terraformrc"
   terraform validate
   terraform plan
   terraform apply
   ```

## What's New

The enhanced external provider now supports **dynamic data types** in the `result` attribute. This means your external programs can return:

- **Strings**: `"message": "hello"`
- **Numbers**: `"count": 42`
- **Booleans**: `"enabled": true`
- **Arrays**: `"items": ["a", "b", "c"]`
- **Objects**: `"nested": {"key": "value"}`
- **Mixed structures**: Any combination of the above

## Usage Examples

### Accessing Fields

```hcl
# String field
data.external.example.result.message

# Number field
data.external.example.result.count

# Boolean field
data.external.example.result.enabled

# Array access
data.external.example.result.items[0]

# Nested object
data.external.example.result.nested.foo
```

### Using in Local-Exec

```hcl
resource "null_resource" "example" {
  provisioner "local-exec" {
    command = "echo ${data.external.example.result.message}"
  }
}
```

### Using in Outputs

```hcl
output "message" {
  value = data.external.example.result.message
}
```

## Expected Output

When you run `terraform apply`, you should see outputs like:

```
full_result = {
  "count" = 42
  "enabled" = true
  "foobaz" = "abc123 def456"
  "items" = ["apple", "banana", "cherry"]
  "message" = "hello from external"
  "nested" = {
    "baz" = "def456"
    "foo" = "abc123"
    "numbers" = [1, 2, 3]
  }
}
message = "hello from external"
count = 42
enabled = true
items = ["apple", "banana", "cherry"]
nested_foo = "abc123"
```

## Backward Compatibility

The provider maintains full backward compatibility. External programs that return simple `map[string]string` JSON objects will continue to work as before.
