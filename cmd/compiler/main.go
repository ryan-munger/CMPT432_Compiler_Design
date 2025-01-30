package main

import (
	"flag"
	"fmt"
	"os"
	"gopiler/internal/lexer"
)

func main() {
	inputFile := flag.String("f", "", "Path to source for compilation")
	flag.Parse()

	// Ensure we got a file
	if *inputFile == "" {
		fmt.Println("Error: No input file specified.")
		fmt.Println("Usage: mycompiler -input <source-file>")
		os.Exit(1)
	}

	// omg we compiled! no need for the rest of the semester
	fmt.Printf("Compiling file: %s\n", *inputFile)

	lexer.Test()
}

