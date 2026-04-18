package notation

import (
	"testing"

	"github.com/lukev/tm_server/internal/game/board"
)

func TestConvertRiverCoordToAxial_UsesRiverRowIndexing(t *testing.T) {
	tests := []struct {
		coord string
		want  board.Hex
	}{
		{coord: "R~B1", want: board.NewHex(1, 1)},
		{coord: "R~C3", want: board.NewHex(2, 2)},
	}

	for _, tt := range tests {
		got, err := ConvertRiverCoordToAxial(tt.coord)
		if err != nil {
			t.Fatalf("ConvertRiverCoordToAxial(%q) error = %v", tt.coord, err)
		}
		if got != tt.want {
			t.Fatalf("ConvertRiverCoordToAxial(%q) = %v, want %v", tt.coord, got, tt.want)
		}
	}
}
