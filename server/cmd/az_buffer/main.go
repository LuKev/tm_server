package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lukev/tm_server/internal/az/buffer"
)

type sourceFlags []string

func (s *sourceFlags) String() string {
	return strings.Join(*s, ",")
}

func (s *sourceFlags) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("empty source")
	}
	*s = append(*s, value)
	return nil
}

func main() {
	var sources sourceFlags
	flag.Var(&sources, "source", "self-play JSONL source path, optionally path@limit; repeatable")
	output := flag.String("output", "", "replay-buffer JSONL output path")
	summary := flag.String("summary", "", "optional JSON summary output path")
	seed := flag.Int64("seed", 1, "deterministic sampling seed")
	flag.Parse()

	parsed, err := parseSources(sources)
	if err != nil {
		exitf("parse sources: %v", err)
	}
	result, err := buffer.Build(buffer.Config{
		Sources:     parsed,
		OutputPath:  *output,
		SummaryPath: *summary,
		Seed:        *seed,
	})
	if err != nil {
		exitf("build buffer: %v", err)
	}
	raw, _ := json.Marshal(result)
	_, _ = fmt.Fprintln(os.Stderr, string(raw))
}

func parseSources(values []string) ([]buffer.Source, error) {
	out := make([]buffer.Source, 0, len(values))
	for _, value := range values {
		source, err := parseSource(value)
		if err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	return out, nil
}

func parseSource(value string) (buffer.Source, error) {
	path := strings.TrimSpace(value)
	limit := 0
	if at := strings.LastIndex(path, "@"); at >= 0 {
		rawLimit := strings.TrimSpace(path[at+1:])
		path = strings.TrimSpace(path[:at])
		if rawLimit == "" {
			return buffer.Source{}, fmt.Errorf("missing limit in %q", value)
		}
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return buffer.Source{}, fmt.Errorf("invalid limit in %q: %w", value, err)
		}
		if parsed <= 0 {
			return buffer.Source{}, fmt.Errorf("limit must be positive in %q", value)
		}
		limit = parsed
	}
	if path == "" {
		return buffer.Source{}, fmt.Errorf("missing path in %q", value)
	}
	return buffer.Source{Path: path, Limit: limit}, nil
}

func exitf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
