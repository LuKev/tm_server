package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lukev/tm_server/internal/az/model"
	"github.com/lukev/tm_server/internal/az/train"
)

func main() {
	input := flag.String("input", "", "self-play JSONL input")
	output := flag.String("output", "az_model.json", "table model output")
	flag.Parse()
	if *input == "" {
		exitf("-input is required")
	}
	table, err := train.TrainFile(*input)
	if err != nil {
		exitf("train: %v", err)
	}
	if err := model.SaveTableModel(*output, table); err != nil {
		exitf("save model: %v", err)
	}
}

func exitf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
