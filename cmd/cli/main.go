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
	inputFile := flag.String("f", "", "String; Path to source for compilation")
	verboseMode := flag.Bool("v", false, "Bool; Toggle Verbose Mode")
	flag.Parse()

	var filedata string = verifyFile(*inputFile)
	internal.SetVerbose(*verboseMode)
	internal.SetWebMode(false)

	internal.Info(fmt.Sprintf("Starting compilation of: %s with verbose mode: %t", *inputFile, *verboseMode), "GOPILER", true)

	if len(filedata) == 0 {
		internal.Warn("Source file empty. No compilation will be executed.", "GOPILER")
	} else {
		internal.Lex(filedata)
	}

	internal.Info("All compilations complete.", "GOPILER", true)
}
