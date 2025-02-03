package main

import (
	"flag"
	"fmt"
	"gopiler/internal"
	"os"
)

func verifyFile(inputFile string) string {
	// Ensure we got a file
	if inputFile == "" {
		fmt.Println("Error: No input file specified.")
		flag.Usage()
		os.Exit(1)
	}
	filebytes, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Println("Error processing file:", err)
		os.Exit(1)
	}
	var filedata string = string(filebytes)
	// fmt.Println("File contents:")
	// fmt.Println(string(filedata))
	return filedata
}

func main() {
	inputFile := flag.String("f", "", "Path to source for compilation")
	verboseMode := flag.Bool("v", false, "Toggle Verbose Mode")
	flag.Parse()

	var filedata string = verifyFile(*inputFile)
	internal.SetVerbose(*verboseMode)

	internal.Log(fmt.Sprintf("Starting compilation of: %s with verbose mode: %t", *inputFile, *verboseMode), "GOPILER", true)

	internal.Lex(filedata)
}
