data "external" "example" {
  program = ["bash", "${path.module}/example-data-source.sh"]

  query = {
    # arbitrary map from strings to strings, passed
    # to the external program as the data query.
    foo = "abc123"
    baz = "def456"
  }
}
