package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"safeshell/pkg/models"
	"safeshell/pkg/utils"
)

// SnapshotDirName is the folder inside a snapshot that holds workspace data.
const WorkspaceDataSubdir = "data"

// CreateSnapshot captures a point-in-time copy of workspaceDir into snapshotDir.
func CreateSnapshot(workspaceDir, snapshotDir string) (*models.Snapshot, error) {
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	now := time.Now().UTC()
	snapshotID := now.Format("2006-01-02-150405.000000")

	snapPath := filepath.Join(snapshotDir, snapshotID)
	dataPath := filepath.Join(snapPath, WorkspaceDataSubdir)

	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot data dir: %w", err)
	}

	// Copy workspace content into snapshot data
	if _, err := os.Stat(workspaceDir); err == nil {
		if err := utils.CopyDir(workspaceDir, dataPath); err != nil {
			return nil, fmt.Errorf("failed to copy workspace into snapshot: %w", err)
		}
	}

	snap := &models.Snapshot{
		SnapshotID: snapshotID,
		CreatedAt:  now,
		Workspace:  "workspace",
	}

	metaBytes, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot metadata: %w", err)
	}

	metaFile := filepath.Join(snapPath, "metadata.json")
	if err := os.WriteFile(metaFile, metaBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write snapshot metadata: %w", err)
	}

	return snap, nil
}

// GetLatestSnapshot returns the most recent snapshot in snapshotDir.
func GetLatestSnapshot(snapshotDir string) (*models.Snapshot, error) {
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return nil, errors.New("no snapshots found: snapshot directory does not exist")
	}

	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	var snapshotIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			metaFile := filepath.Join(snapshotDir, entry.Name(), "metadata.json")
			if _, err := os.Stat(metaFile); err == nil {
				snapshotIDs = append(snapshotIDs, entry.Name())
			}
		}
	}

	if len(snapshotIDs) == 0 {
		return nil, errors.New("no valid snapshots found")
	}

	sort.Strings(snapshotIDs)
	latestID := snapshotIDs[len(snapshotIDs)-1]

	metaFile := filepath.Join(snapshotDir, latestID, "metadata.json")
	metaBytes, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read latest snapshot metadata: %w", err)
	}

	var snap models.Snapshot
	if err := json.Unmarshal(metaBytes, &snap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot metadata: %w", err)
	}

	return &snap, nil
}

// Rollback restores workspaceDir to the exact state saved in snapshot.
func Rollback(snap *models.Snapshot, snapshotDir, workspaceDir string) error {
	if snap == nil {
		return errors.New("snapshot is nil")
	}

	snapDataPath := filepath.Join(snapshotDir, snap.SnapshotID, WorkspaceDataSubdir)
	if _, err := os.Stat(snapDataPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot data path %s does not exist", snapDataPath)
	}

	// 1. Clear workspace
	if err := utils.ClearDir(workspaceDir); err != nil {
		return fmt.Errorf("failed to clear workspace during rollback: %w", err)
	}

	// 2. Restore snapshot data into workspace
	if err := utils.CopyDir(snapDataPath, workspaceDir); err != nil {
		return fmt.Errorf("failed to copy snapshot data to workspace: %w", err)
	}

	return nil
}
