package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lukev/tm_server/internal/notation"
)

func main() {
	replay := flag.Bool("replay", false, "use ConvertSnellmanToConciseForReplay (line-preserving)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: convert_to_concise [--replay] <input_file>")
		os.Exit(1)
	}

	inputFile := flag.Arg(0)
	content, err := os.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var concise string
	if *replay {
		concise, err = notation.ConvertSnellmanToConciseForReplay(string(content))
	} else {
		concise, err = notation.ConvertSnellmanToConcise(string(content))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting to concise: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(concise)
}
