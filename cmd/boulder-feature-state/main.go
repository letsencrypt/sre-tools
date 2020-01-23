package main

import "fmt"

// TODO(@cpu): required constants.
//
// * Some kind of file suffix pattern for input files?
// * Regex for extracting hostname/boulder component from file names?

// TODO(@cpu): required types.
//
// * Input "Shape" type to unmarshal into for finding the feature flags
// * Output type to marshal to disk

func main() {
	// TODO(@cpu): Command line flag handling with `flag`.
	//
	// Required CLI flags:
	//   -directory -> specify a directory to process for templated files
	//   -outputDirectory -> specify a place to write processed output files.

	// TODO(@cpu): High level control flow.
	//
	// 1. Handle an input directory to find input files.
	// 2. Process each input file to find feature flag settings
	// 3. Output feature flag settings and context to the outputDirectory
	// 4. ???
	// 5. Profit
	fmt.Printf("hello world\n")
}
