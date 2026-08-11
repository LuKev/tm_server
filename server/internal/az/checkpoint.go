package az

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ResolveCheckpointReference resolves exactly one explicit snapshot or atomic
// latest reference and verifies latest.json's content hash before use.
func ResolveCheckpointReference(checkpoint, latestPath string) (string, error) {
	if checkpoint != "" && latestPath != "" {
		return "", fmt.Errorf("checkpoint and latest reference are mutually exclusive")
	}
	if latestPath == "" {
		return checkpoint, nil
	}
	var latest struct {
		Path             string `json:"path"`
		CheckpointSHA256 string `json:"checkpoint_sha256"`
	}
	raw, err := os.ReadFile(latestPath)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(raw, &latest); err != nil || latest.Path == "" || latest.CheckpointSHA256 == "" {
		return "", fmt.Errorf("invalid checkpoint latest reference %s", latestPath)
	}
	target := filepath.Join(filepath.Dir(latestPath), latest.Path)
	checkpointBytes, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	if fmt.Sprintf("%x", sha256.Sum256(checkpointBytes)) != latest.CheckpointSHA256 {
		return "", fmt.Errorf("latest checkpoint hash mismatch for %s", target)
	}
	return target, nil
}
