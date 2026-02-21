package game

import (
	"testing"

	"github.com/lukev/tm_server/internal/models"
)

func TestManagerExecuteActionWithMeta_RevisionAndIdempotency(t *testing.T) {
	mgr := NewManager()
	if err := mgr.CreateGameWithOptions("g1", []string{"p1", "p2"}, CreateGameOptions{RandomizeTurnOrder: false}); err != nil {
		t.Fatalf("failed creating game: %v", err)
	}

	first := &SelectFactionAction{PlayerID: "p1", FactionType: models.FactionWitches}
	result, err := mgr.ExecuteActionWithMeta("g1", first, ActionMeta{
		ActionID:         "a1",
		ExpectedRevision: 0,
		SeatID:           "p1",
	})
	if err != nil {
		t.Fatalf("expected first action success, got error: %v", err)
	}
	if result == nil || result.Revision != 1 || result.Duplicate {
		t.Fatalf("unexpected first action result: %+v", result)
	}

	dupResult, dupErr := mgr.ExecuteActionWithMeta("g1", first, ActionMeta{
		ActionID:         "a1",
		ExpectedRevision: 0,
		SeatID:           "p1",
	})
	if dupErr != nil {
		t.Fatalf("expected duplicate action replay success, got error: %v", dupErr)
	}
	if dupResult == nil || !dupResult.Duplicate || dupResult.Revision != 1 {
		t.Fatalf("unexpected duplicate result: %+v", dupResult)
	}

	second := &SelectFactionAction{PlayerID: "p2", FactionType: models.FactionAuren}
	_, staleErr := mgr.ExecuteActionWithMeta("g1", second, ActionMeta{
		ActionID:         "a2",
		ExpectedRevision: 0,
		SeatID:           "p2",
	})
	if staleErr == nil {
		t.Fatalf("expected stale revision error")
	}
	if _, ok := staleErr.(*RevisionMismatchError); !ok {
		t.Fatalf("expected RevisionMismatchError, got %T (%v)", staleErr, staleErr)
	}

	rev, ok := mgr.GetRevision("g1")
	if !ok {
		t.Fatalf("expected revision to exist for game")
	}
	if rev != 1 {
		t.Fatalf("expected revision 1 after stale request, got %d", rev)
	}
}
