package buffer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

const scannerBufferMax = 16 * 1024 * 1024

type Source struct {
	Path  string `json:"path"`
	Limit int    `json:"limit,omitempty"`
}

type Config struct {
	Sources     []Source
	OutputPath  string
	SummaryPath string
	Seed        int64
}

type SourceSummary struct {
	Path          string `json:"path"`
	Limit         int    `json:"limit,omitempty"`
	InputRecords  int    `json:"inputRecords"`
	OutputRecords int    `json:"outputRecords"`
}

type Summary struct {
	Output             string          `json:"output"`
	TotalInputRecords  int             `json:"totalInputRecords"`
	TotalOutputRecords int             `json:"totalOutputRecords"`
	Sources            []SourceSummary `json:"sources"`
}

func Build(config Config) (Summary, error) {
	if len(config.Sources) == 0 {
		return Summary{}, fmt.Errorf("at least one source is required")
	}
	if config.OutputPath == "" {
		return Summary{}, fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(config.OutputPath), 0755); err != nil {
		return Summary{}, fmt.Errorf("create output dir: %w", err)
	}
	out, err := os.Create(config.OutputPath)
	if err != nil {
		return Summary{}, fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	defer writer.Flush()

	summary := Summary{
		Output:  config.OutputPath,
		Sources: make([]SourceSummary, 0, len(config.Sources)),
	}
	for i, source := range config.Sources {
		sourceSummary, err := appendSource(writer, source, rand.New(rand.NewSource(config.Seed+int64(i)*1000003)))
		if err != nil {
			return Summary{}, err
		}
		summary.TotalInputRecords += sourceSummary.InputRecords
		summary.TotalOutputRecords += sourceSummary.OutputRecords
		summary.Sources = append(summary.Sources, sourceSummary)
	}
	if config.SummaryPath != "" {
		if err := writeSummary(config.SummaryPath, summary); err != nil {
			return Summary{}, err
		}
	}
	return summary, nil
}

func appendSource(writer io.Writer, source Source, rng *rand.Rand) (SourceSummary, error) {
	if strings.TrimSpace(source.Path) == "" {
		return SourceSummary{}, fmt.Errorf("source path is required")
	}
	file, err := os.Open(source.Path)
	if err != nil {
		return SourceSummary{}, fmt.Errorf("open source %s: %w", source.Path, err)
	}
	defer file.Close()

	summary := SourceSummary{Path: source.Path, Limit: source.Limit}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), scannerBufferMax)

	if source.Limit <= 0 {
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			summary.InputRecords++
			if _, err := fmt.Fprintln(writer, line); err != nil {
				return SourceSummary{}, fmt.Errorf("write output: %w", err)
			}
			summary.OutputRecords++
		}
		if err := scanner.Err(); err != nil {
			return SourceSummary{}, fmt.Errorf("scan source %s: %w", source.Path, err)
		}
		return summary, nil
	}

	reservoir := make([]string, 0, source.Limit)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		summary.InputRecords++
		if len(reservoir) < source.Limit {
			reservoir = append(reservoir, line)
			continue
		}
		j := rng.Intn(summary.InputRecords)
		if j < source.Limit {
			reservoir[j] = line
		}
	}
	if err := scanner.Err(); err != nil {
		return SourceSummary{}, fmt.Errorf("scan source %s: %w", source.Path, err)
	}
	for _, line := range reservoir {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			return SourceSummary{}, fmt.Errorf("write output: %w", err)
		}
		summary.OutputRecords++
	}
	return summary, nil
}

func writeSummary(path string, summary Summary) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create summary dir: %w", err)
	}
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("encode summary: %w", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}
