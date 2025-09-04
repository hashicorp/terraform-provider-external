// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// This is a minimal implementation of the external ephemeral resource protocol
// intended only for use in the provider acceptance tests.
//
// The main difference from the data source test program is that this writes
// output to files specified in the query, allowing tests to inspect the
// behavior of ephemeral resources that can't use output blocks.
func main() {
	queryBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	var query map[string]*string
	err = json.Unmarshal(queryBytes, &query)
	if err != nil {
		panic(err)
	}

	if _, ok := query["fail"]; ok {
		fmt.Fprintf(os.Stderr, "I was asked to fail\n")
		os.Exit(1)
	}

	var result = map[string]string{
		"result": "yes",
	}

	if queryValue, ok := query["value"]; ok && queryValue != nil {
		result["query_value"] = *queryValue
	}

	if len(os.Args) >= 2 {
		result["argument"] = os.Args[1]
	}

	// Add working directory to result if requested
	if wd, err := os.Getwd(); err == nil {
		result["working_dir"] = wd
	}

	for queryKey, queryValue := range query {
		if queryValue != nil {
			result[queryKey] = *queryValue
		}
	}

	// Write results to stdout (standard protocol)
	resultBytes, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(resultBytes)

	// Additionally, write results to a file if output_file is specified
	if outputFile, ok := query["output_file"]; ok && outputFile != nil {
		// Create directory if it doesn't exist
		dir := filepath.Dir(*outputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}

		// Write JSON result to file
		err := os.WriteFile(*outputFile, resultBytes, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to file %s: %v\n", *outputFile, err)
			os.Exit(1)
		}
	}

	// Write working directory to a separate file if working_dir_file is specified
	if wdFile, ok := query["working_dir_file"]; ok && wdFile != nil {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get working directory: %v\n", err)
			os.Exit(1)
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(*wdFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}

		err = os.WriteFile(*wdFile, []byte(wd), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write working dir to file %s: %v\n", *wdFile, err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}
