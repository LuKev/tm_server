package main

import (
	"fmt"
	"os"

	"github.com/lukev/tm_server/internal/notation"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: convert_to_concise <input_file>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	concise, err := notation.ConvertSnellmanToConcise(string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to concise: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(concise)
}
