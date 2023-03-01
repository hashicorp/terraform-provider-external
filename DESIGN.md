# External Provider Design

 The External Provider allows external programs to interact with Terraform by implementing a specific protocol. 

Below we have a collection of _Goals_ and _Patterns_: they represent the guiding principles applied during the
development of this provider. Some are in place, others are ongoing processes, others are still just inspirational.

## Goals

* [_Stability over features_](.github/CONTRIBUTING.md)
* Provide an "escape hatch" for external programs to interact with Terraform in exceptional situations where a first-class Terraform provider is not more appropriate.
  * This provider is **not** intended to work with any external program with no modifications, but for small "glue" programs tailored to work with Terraform.
* Define a protocol for external programs to implement in order to communicate with Terraform through the provider.
* Provide comprehensive documentation 
* Highlight intended and unadvisable usages

## Patterns

Specific to this provider:

* The data source protocol uses JSON objects to communicate between the provider and the external program.

General to development:

* **Avoid repetition**: the entities managed can sometimes require similar pieces of logic and/or schema to be realised.
  When this happens it's important to keep the code shared in communal sections, so to avoid having to modify code in
  multiple places when they start changing.
* **Test expectations as well as bugs**: While it's typical to write tests to exercise a new functionality, it's key to
  also provide tests for issues that get identified and fixed, so to prove resolution as well as avoid regression.
* **Automate boring tasks**: Processes that are manual, repetitive and can be automated, should be. In addition to be a
  time-saving practice, this ensures consistency and reduces human error (ex. static code analysis).
* **Semantic versioning**: Adhering to HashiCorp's own
  [Versioning Specification](https://www.terraform.io/plugin/sdkv2/best-practices/versioning#versioning-specification)
  ensures we provide a consistent practitioner experience, and a clear process to deprecation and decommission.