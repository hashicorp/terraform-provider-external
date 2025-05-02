## 2.3.5 (May 01, 2025)

NOTES:

* Update dependencies  ([#411](https://github.com/hashicorp/terraform-provider-external/issues/411))

## 2.3.5-alpha1 (April 28, 2025)

NOTES:

* all: This release is being used to test new build and release actions. ([#412](https://github.com/hashicorp/terraform-provider-external/issues/412))

## 2.3.4 (September 10, 2024)

NOTES:

* all: This release introduces no functional changes. It does however include dependency updates which address upstream CVEs. ([#357](https://github.com/hashicorp/terraform-provider-external/issues/357))

## 2.3.3 (February 12, 2024)

NOTES:

* data-source/external: The stderr output of the executed program will now always be logged at the TRACE level, regardless of exit code. ([#67](https://github.com/hashicorp/terraform-provider-external/issues/67))

## 2.3.2 (November 21, 2023)

BUG FIXES:

* data-source/external: Prevent regression since v2.3.1 where null `query` element values would be sent to the program as an empty string ([#208](https://github.com/hashicorp/terraform-provider-external/issues/208))

## 2.3.1 (March 06, 2023)

BUG FIXES:

* data-source/external: Remove query validation to allow null and empty string values to be passed to the external program ([#193](https://github.com/hashicorp/terraform-provider-external/issues/193))

## 2.3.0 (March 06, 2023)

NOTES:

* provider: Rewritten to use the [`terraform-plugin-framework`](https://www.terraform.io/plugin/framework) ([#184](https://github.com/hashicorp/terraform-provider-external/issues/184))

## 2.2.3 (November 9, 2022)

BUG FIXES:

* data-source/external: Prevented unexpected error after 20 minutes of program execution ([#165](https://github.com/terraform-providers/terraform-provider-external/issues/165))

## 2.2.2 (March 14, 2022)

NOTES:

* This release is a republishing of the 2.2.1 release to include a missing release asset. It is identical otherwise.

## 2.2.1 (March 14, 2022)

BUG FIXES:

* data-source/external: Prevented panics with empty string (`""`) elements in `program` argument

## 2.2.0 (January 3, 2022)

ENHANCEMENTS:

* data-source/external: Program execution will now exit immediately when receiving an interrupt signal (Ctrl-c) from Terraform ([#91](https://github.com/terraform-providers/terraform-provider-external/issues/91))
* data-source/external: Enhanced error messaging to include more troubleshooting information and recommendations ([#93](https://github.com/terraform-providers/terraform-provider-external/issues/93))
* data-source/external: Added trace log for program being executed ([#95](https://github.com/terraform-providers/terraform-provider-external/issues/95))

## 2.1.1 (December 14, 2021)

NOTES:

* The release process was upgraded to use Go 1.16.12 to mitigate CVE-2021-44717.

## 2.1.0 (February 19, 2021)

Binary releases of this provider now include the darwin-arm64 platform. This version contains no further changes.

## 2.0.0 (October 12, 2020)

Binary releases of this provider now include the linux-arm64 platform.

BREAKING CHANGES:

* Upgrade to version 2 of the Terraform Plugin SDK, which drops support for Terraform 0.11. This provider will continue to work as expected for users of Terraform 0.11, which will not download the new version. ([#47](https://github.com/terraform-providers/terraform-provider-external/issues/47))

BUG FIXES:

* In Debugging mode, print the JSON from external data source as a string ([#46](https://github.com/terraform-providers/terraform-provider-external/issues/46))

## 1.2.0 (June 19, 2019)

IMPROVEMENTS

* Trace logging added for JSON output ([#36](https://github.com/terraform-providers/terraform-provider-external/issues/36))

## 1.1.2 (April 30, 2019)

* This release includes another Terraform SDK upgrade intended to align with that being used for other providers as we prepare for the Core v0.12.0 release. It should have no significant changes in behavior for this provider.

## 1.1.1 (April 12, 2019)

* This release includes only a Terraform SDK upgrade intended to align with that being used for other providers as we prepare for the Core v0.12.0 release. It should have no significant changes in behavior for this provider.

## 1.1.0 (March 20, 2019)

ENHANCEMENTS:

* The provider is now compatible with Terraform v0.12, while retaining compatibility with prior versions.
* `external` data source now accepts `working_dir` argument to set the working directory for the child process. ([#12](https://github.com/terraform-providers/terraform-provider-external/issues/12))

## 1.0.0 (September 14, 2017)

* No changes from 0.1.0; just adjusting to [the new version numbering scheme](https://www.hashicorp.com/blog/hashicorp-terraform-provider-versioning/).

## 0.1.0 (June 20, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
