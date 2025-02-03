package main

import (
	"flag"
	"fmt"
	"os"
	"gopiler/internal/lexer"
)

func main() {
	inputFile := flag.String("f", "", "Path to source for compilation")
	verboseMode := flag.Bool("v", false, "Toggle Verbose Mode")
	flag.Parse()

	// Ensure we got a file
	if *inputFile == "" {
		fmt.Println("Error: No input file specified.")
		flag.Usage()
		os.Exit(1)
	}

	// omg we compiled! no need for the rest of the semester
	fmt.Printf("Compiling file: %s\n", *inputFile)
	fmt.Printf("Verbose mode: %t\n", *verboseMode)

	lexer.Test()
}

