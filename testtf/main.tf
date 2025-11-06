terraform {
  required_providers {
    external = {
      source = "hashicorp/external"
    }
  }
}

# Configure the external provider to use our locally built version
# Comment this out if you want to use the published version
provider "external" {}

# External data source that calls our bash script
data "external" "example" {
  program = ["bash", "${path.module}/example-data-source.sh"]

  query = {
    # arbitrary map from strings to strings, passed
    # to the external program as the data query.
    foo = "abc123"
    baz = "def456"
  }
}

# Output the entire result to see the dynamic structure
output "full_result" {
  value       = data.external.example.result
  description = "The full dynamic result from the external program"
}

# Access specific fields from the dynamic result
output "message" {
  value       = data.external.example.result.message
  description = "The message field"
}

output "count" {
  value       = data.external.example.result.count
  description = "The count field (number)"
}

output "enabled" {
  value       = data.external.example.result.enabled
  description = "The enabled field (boolean)"
}

output "items" {
  value       = data.external.example.result.items
  description = "The items field (list)"
}

output "nested_foo" {
  value       = data.external.example.result.nested.foo
  description = "Access nested object fields"
}

# Null resource with local-exec provisioner to demonstrate usage
resource "null_resource" "example" {
  # Trigger this resource whenever the external data changes
  triggers = {
    result_hash = sha256(jsonencode(data.external.example.result))
  }

  provisioner "local-exec" {
    command = <<-EOT
      echo "Message: ${data.external.example.result.message}"
      echo "Count: ${data.external.example.result.count}"
      echo "Enabled: ${data.external.example.result.enabled}"
      echo "First item: ${data.external.example.result.items[0]}"
      echo "Nested foo: ${data.external.example.result.nested.foo}"
      echo "Full result: ${jsonencode(data.external.example.result)}"
    EOT
  }
}
