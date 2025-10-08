// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// This is a minimal implementation of the external data source protocol
// intended only for use in the provider acceptance tests.
//
// In practice it's likely not much harder to just write a real Terraform
// plugin if you're going to be writing your data source in Go anyway;
// this example is just in Go because we want to avoid introducing
// additional language runtimes into the test environment.
func main() {
	queryBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var query map[string]interface{}
	err = json.Unmarshal(queryBytes, &query)
	if err != nil {
		panic(err)
	}

	if _, ok := query["fail"]; ok {
		fmt.Fprintf(os.Stderr, "I was asked to fail\n")
		os.Exit(1)
	}

	var result = map[string]string{
		"result":           "yes",
		"serialized_query": string(queryBytes),
	}

	if queryValue, ok := query["value"]; ok && queryValue != nil {
		// Only set value if query["value"] is a string
		if queryValue, ok := queryValue.(string); ok {
			result["query_value"] = queryValue
		}
	}

	if len(os.Args) >= 2 {
		result["argument"] = os.Args[1]
	}

	for queryKey, queryValue := range query {
		if queryValue, ok := queryValue.(string); ok {
			result[queryKey] = queryValue
		}
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}

	os.Stdout.Write(resultBytes)
	os.Exit(0)
}
