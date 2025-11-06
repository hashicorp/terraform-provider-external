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

# Now let's return a complex JSON structure to demonstrate dynamic types
jq -n \
  --arg foobaz "$FOOBAZ" \
  --arg foo "$FOO" \
  --arg baz "$BAZ" \
  '{
    "message": "hello from external",
    "foobaz": $foobaz,
    "count": 42,
    "enabled": true,
    "items": ["apple", "banana", "cherry"],
    "nested": {
      "foo": $foo,
      "baz": $baz,
      "numbers": [1, 2, 3]
    }
  }'
