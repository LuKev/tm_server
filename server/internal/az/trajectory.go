package az

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lukev/tm_server/internal/game"
	"github.com/lukev/tm_server/internal/models"
)

type TrajectoryManifest struct {
	FormatVersion    int                   `json:"format_version"`
	RulesVersion     int                   `json:"rules_version"`
	StateVersion     int                   `json:"state_version"`
	ActionVersion    int                   `json:"action_version"`
	EngineCommit     string                `json:"engine_commit"`
	ModelID          string                `json:"model_id"`
	CheckpointSHA256 string                `json:"checkpoint_sha256,omitempty"`
	Seed             int64                 `json:"seed"`
	Factions         [2]models.FactionType `json:"factions"`
}

type TrajectoryStep struct {
	State         json.RawMessage `json:"state"`
	StateHash     string          `json:"state_hash"`
	Actions       []SearchAction  `json:"actions"`
	Visits        []int           `json:"visits"`
	ChosenIndex   int             `json:"chosen_index"`
	DecisionSeat  int             `json:"decision_seat"`
	NextStateHash string          `json:"next_state_hash"`
	Outcome       float32         `json:"outcome"`
	FinalVP       int             `json:"final_vp"`
	VPMargin      int             `json:"vp_margin"`
}

type Trajectory struct {
	Manifest          TrajectoryManifest `json:"manifest"`
	Steps             []TrajectoryStep   `json:"steps"`
	FinalVP           [2]int             `json:"final_vp"`
	PlyCount          int                `json:"ply_count"`
	Completed         bool               `json:"completed"`
	TerminalState     json.RawMessage    `json:"terminal_state"`
	TerminalStateHash string             `json:"terminal_state_hash"`
}

type ShardRef struct {
	Path   string
	SHA256 string
	Bytes  int
}

func WriteTrajectoryShard(directory string, trajectory Trajectory) (ShardRef, error) {
	if err := validateTrajectory(trajectory); err != nil {
		return ShardRef{}, err
	}
	raw, err := json.Marshal(trajectory)
	if err != nil {
		return ShardRef{}, err
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	writer.Header.ModTime = time.Unix(0, 0)
	writer.Header.OS = 255
	if _, err := writer.Write(raw); err != nil {
		return ShardRef{}, err
	}
	if err := writer.Close(); err != nil {
		return ShardRef{}, err
	}
	digest := sha256.Sum256(compressed.Bytes())
	hash := hex.EncodeToString(digest[:])
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return ShardRef{}, err
	}
	path := filepath.Join(directory, "trajectory-"+hash+".json.gz")
	if existing, err := os.ReadFile(path); err == nil {
		if !bytes.Equal(existing, compressed.Bytes()) {
			return ShardRef{}, fmt.Errorf("content-addressed shard collision at %s", path)
		}
		return ShardRef{Path: path, SHA256: hash, Bytes: len(existing)}, nil
	} else if !os.IsNotExist(err) {
		return ShardRef{}, err
	}
	temporary, err := os.CreateTemp(directory, ".trajectory-*.tmp")
	if err != nil {
		return ShardRef{}, err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if _, err := temporary.Write(compressed.Bytes()); err != nil {
		return ShardRef{}, err
	}
	if err := temporary.Sync(); err != nil {
		return ShardRef{}, err
	}
	if err := temporary.Close(); err != nil {
		return ShardRef{}, err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return ShardRef{}, err
	}
	committed = true
	if err := os.Chmod(path, 0o444); err != nil {
		return ShardRef{}, err
	}
	return ShardRef{Path: path, SHA256: hash, Bytes: compressed.Len()}, nil
}

func ReadTrajectoryShard(path string) (Trajectory, ShardRef, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return Trajectory{}, ShardRef{}, err
	}
	digest := sha256.Sum256(compressed)
	hash := hex.EncodeToString(digest[:])
	wantName := "trajectory-" + hash + ".json.gz"
	if filepath.Base(path) != wantName {
		return Trajectory{}, ShardRef{}, fmt.Errorf("trajectory hash mismatch: file %s hashes to %s", filepath.Base(path), hash)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return Trajectory{}, ShardRef{}, err
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		return Trajectory{}, ShardRef{}, err
	}
	if err := reader.Close(); err != nil {
		return Trajectory{}, ShardRef{}, err
	}
	var trajectory Trajectory
	if err := json.Unmarshal(raw, &trajectory); err != nil {
		return Trajectory{}, ShardRef{}, err
	}
	if err := validateTrajectory(trajectory); err != nil {
		return Trajectory{}, ShardRef{}, err
	}
	return trajectory, ShardRef{Path: path, SHA256: hash, Bytes: len(compressed)}, nil
}

func validateTrajectory(trajectory Trajectory) error {
	manifest := trajectory.Manifest
	if manifest.FormatVersion != TrajectoryFormatVersion || manifest.RulesVersion != 1 || manifest.StateVersion != StateSchemaVersion || manifest.ActionVersion != ActionSchemaVersion {
		return fmt.Errorf("trajectory schema mismatch: format=%d rules=%d state=%d action=%d", manifest.FormatVersion, manifest.RulesVersion, manifest.StateVersion, manifest.ActionVersion)
	}
	if manifest.EngineCommit == "" {
		return fmt.Errorf("trajectory engine commit is required")
	}
	if manifest.ModelID == "" {
		return fmt.Errorf("trajectory model ID is required")
	}
	if strings.HasPrefix(manifest.ModelID, "model-") {
		decoded, err := hex.DecodeString(manifest.CheckpointSHA256)
		if err != nil || len(decoded) != sha256.Size {
			return fmt.Errorf("trained trajectory checkpoint hash is required")
		}
	}
	if !trajectory.Completed || len(trajectory.Steps) == 0 || trajectory.PlyCount != len(trajectory.Steps) {
		return fmt.Errorf("trajectory must be a naturally completed non-empty game")
	}
	for i, step := range trajectory.Steps {
		if len(step.State) == 0 || step.StateHash == "" || len(step.Actions) == 0 || len(step.Actions) != len(step.Visits) {
			return fmt.Errorf("trajectory step %d has incomplete state, action, or visit data", i)
		}
		if step.DecisionSeat < 0 || step.DecisionSeat > 1 || step.ChosenIndex < 0 || step.ChosenIndex >= len(step.Actions) {
			return fmt.Errorf("trajectory step %d has invalid seat or chosen action", i)
		}
		digest := sha256.Sum256(step.State)
		if step.StateHash != hex.EncodeToString(digest[:16]) {
			return fmt.Errorf("trajectory step %d state hash mismatch", i)
		}
		selfFaction, err := validateCanonicalRecord(step.State, false)
		if err != nil {
			return fmt.Errorf("trajectory step %d: %w", i, err)
		}
		if selfFaction != manifest.Factions[step.DecisionSeat] {
			return fmt.Errorf("trajectory step %d decision seat does not match canonical self faction", i)
		}
		seenActions := make(map[string]struct{}, len(step.Actions))
		for actionIndex, action := range step.Actions {
			raw, err := json.Marshal(action)
			if err != nil || string(raw) != action.Key() {
				return fmt.Errorf("trajectory step %d action %d is not canonical", i, actionIndex)
			}
			if _, exists := seenActions[action.Key()]; exists {
				return fmt.Errorf("trajectory step %d contains duplicate actions", i)
			}
			seenActions[action.Key()] = struct{}{}
		}
		totalVisits := 0
		for _, visits := range step.Visits {
			if visits < 0 {
				return fmt.Errorf("trajectory step %d has negative visits", i)
			}
			totalVisits += visits
		}
		if totalVisits == 0 || step.Visits[step.ChosenIndex] == 0 {
			return fmt.Errorf("trajectory step %d has no searched chosen action", i)
		}
		if step.Outcome != -1 && step.Outcome != 0 && step.Outcome != 1 {
			return fmt.Errorf("trajectory step %d has invalid outcome %g", i, step.Outcome)
		}
		seat := step.DecisionSeat
		wantMargin := trajectory.FinalVP[seat] - trajectory.FinalVP[1-seat]
		wantOutcome := float32(0)
		if wantMargin > 0 {
			wantOutcome = 1
		} else if wantMargin < 0 {
			wantOutcome = -1
		}
		if step.FinalVP != trajectory.FinalVP[seat] || step.VPMargin != wantMargin || step.Outcome != wantOutcome {
			return fmt.Errorf("trajectory step %d result is inconsistent with final VP", i)
		}
		wantNext := trajectory.TerminalStateHash
		if i+1 < len(trajectory.Steps) {
			wantNext = trajectory.Steps[i+1].StateHash
		}
		if step.NextStateHash != wantNext {
			return fmt.Errorf("trajectory step %d breaks the state hash chain", i)
		}
	}
	terminalDigest := sha256.Sum256(trajectory.TerminalState)
	if trajectory.TerminalStateHash != hex.EncodeToString(terminalDigest[:16]) {
		return fmt.Errorf("trajectory terminal state hash mismatch")
	}
	if _, err := validateCanonicalRecord(trajectory.TerminalState, true); err != nil {
		return fmt.Errorf("trajectory terminal state: %w", err)
	}
	return nil
}

func validateCanonicalRecord(raw json.RawMessage, terminal bool) (models.FactionType, error) {
	var record struct {
		RulesVersion  int             `json:"rules_version"`
		StateVersion  int             `json:"state_version"`
		ActionVersion int             `json:"action_version"`
		State         json.RawMessage `json:"state"`
		FactionState  json.RawMessage `json:"faction_state"`
	}
	if !json.Valid(raw) || json.Unmarshal(raw, &record) != nil {
		return models.FactionUnknown, fmt.Errorf("state is not valid JSON")
	}
	canonical, err := json.Marshal(record)
	if err != nil || !bytes.Equal(raw, canonical) {
		return models.FactionUnknown, fmt.Errorf("state is not canonical JSON")
	}
	if record.RulesVersion != 1 || record.StateVersion != StateSchemaVersion || record.ActionVersion != ActionSchemaVersion {
		return models.FactionUnknown, fmt.Errorf("state record schema mismatch")
	}
	var state struct {
		Phase game.GamePhase `json:"phase"`
	}
	if len(record.State) == 0 || json.Unmarshal(record.State, &state) != nil {
		return models.FactionUnknown, fmt.Errorf("state record is missing game state")
	}
	if terminal != (state.Phase == game.PhaseEnd) {
		return models.FactionUnknown, fmt.Errorf("state terminal phase mismatch")
	}
	var factionState []struct {
		Type models.FactionType `json:"type"`
	}
	if json.Unmarshal(record.FactionState, &factionState) != nil || len(factionState) != 2 {
		return models.FactionUnknown, fmt.Errorf("state faction roles are missing")
	}
	return factionState[0].Type, nil
}
