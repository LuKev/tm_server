package main

import (
	"fmt"
	"os"

	"github.com/lukev/tm_server/internal/replay"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: replay_validator <game_log.txt>")
		os.Exit(1)
	}

	logFile := os.Args[1]

	// Validate coordinate conversion
	fmt.Println("Validating coordinate conversion...")
	if err := replay.ValidateCoordinateConversion(); err != nil {
		fmt.Printf("Coordinate conversion validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Coordinate conversion validated")

	// Validate terrain layout
	fmt.Println("Validating terrain layout...")
	if err := replay.ValidateTerrainLayout(); err != nil {
		fmt.Printf("Terrain layout validation failed: %v\n", err)
		fmt.Println("\nThe terrain layout in terrain_layout.go needs to be updated to match the snellman base map.")
		os.Exit(1)
	}
	fmt.Println("✓ Terrain layout validated")

	fmt.Printf("\nLoading game log: %s\n", logFile)

	// Create validator
	validator := replay.NewGameValidator()

	// Load game log
	if err := validator.LoadGameLog(logFile); err != nil {
		fmt.Printf("Failed to load game log: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d log entries\n", len(validator.LogEntries))

	// Count different entry types
	comments := 0
	actions := 0
	for _, entry := range validator.LogEntries {
		if entry.IsComment {
			comments++
		} else {
			actions++
		}
	}
	fmt.Printf("  Comments: %d\n", comments)
	fmt.Printf("  Actions: %d\n", actions)

	fmt.Println("\n✓ Game log parsed successfully")

	// Replay the game
	fmt.Println("\nReplaying game...")
	if err := validator.ReplayGame(); err != nil {
		fmt.Printf("\n❌ Game replay failed: %v\n", err)

		// Show error summary
		if validator.HasErrors() {
			fmt.Println("\nValidation errors:")
			fmt.Println(validator.GetErrorSummary())
		}

		os.Exit(1)
	}

	fmt.Println("✓ Game replayed successfully!")

	// Show summary
	if validator.HasErrors() {
		fmt.Printf("\n⚠ Found %d validation errors (non-fatal)\n", len(validator.Errors))
		fmt.Println(validator.GetErrorSummary())
	} else {
		fmt.Println("\n✅ All validations passed!")
		fmt.Printf("Successfully validated %d game actions\n", actions)
	}
}
