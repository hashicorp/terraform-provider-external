data "external" "example" {
  # don't forget to assure the machine running terraform commands 
  # has the specified binary to run the program, in this example bash,
  # could be python, powershell and etc.
  program = ["bash", "${path.module}/example-data-source.sh"]
  
  query = {
    # arbitrary map from strings to strings, passed
    # to the external program as the data query.
    foo = "abc123"
    baz = "def456"
  }
}
