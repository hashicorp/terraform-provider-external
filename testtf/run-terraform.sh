#!/bin/bash

# Helper script to run Terraform with the local provider build

# Set the Terraform CLI config to use our local provider
export TF_CLI_CONFIG_FILE="$(pwd)/.terraformrc"

# Run terraform with whatever arguments were passed
terraform "$@"
