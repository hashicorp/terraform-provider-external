#!/bin/bash

# Exit if any of the intermediate steps fail
set -e

# Extract "foo" and "baz" arguments from the input into
# FOO and BAZ shell variables.
# jq will ensure that the values are properly quoted
# and escaped for consumption by the shell.
eval "$(jq -r '@sh "FOO=\(.foo) BAZ=\(.baz)"')"

# Placeholder for whatever data-fetching logic your script implements
FOOBAZ="$FOO $BAZ"

# Safely produce a JSON object containing the result value.
# jq will ensure that the value is properly quoted
# and escaped to produce a valid JSON string.
jq -n --arg foobaz "$FOOBAZ" '{"foobaz":$foobaz}'