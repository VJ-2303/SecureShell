package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"safeshell/pkg/utils"
)

func TestSnapshotAndRollback(t *testing.T) {
	tmp := t.TempDir()
	workspaceDir := filepath.Join(tmp, "workspace")
	snapshotDir := filepath.Join(tmp, "snapshots")

	if err := os.MkdirAll(filepath.Join(workspaceDir, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "docs", "file.txt"), []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Create Snapshot
	snap, err := CreateSnapshot(workspaceDir, snapshotDir)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	if snap == nil || snap.SnapshotID == "" {
		t.Fatalf("invalid snapshot returned")
	}

	// Sleep slightly to test ordering
	time.Sleep(10 * time.Millisecond)

	// 2. Modify workspace
	if err := os.WriteFile(filepath.Join(workspaceDir, "newfile.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(workspaceDir, "docs", "file.txt")); err != nil {
		t.Fatal(err)
	}

	// 3. Get latest snapshot
	latest, err := GetLatestSnapshot(snapshotDir)
	if err != nil {
		t.Fatalf("GetLatestSnapshot failed: %v", err)
	}
	if latest.SnapshotID != snap.SnapshotID {
		t.Fatalf("expected snapshot %s, got %s", snap.SnapshotID, latest.SnapshotID)
	}

	// 4. Rollback
	if err := Rollback(latest, snapshotDir, workspaceDir); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify workspace matches original
	match, reason, err := utils.CompareDirs(filepath.Join(snapshotDir, snap.SnapshotID, WorkspaceDataSubdir), workspaceDir)
	if err != nil {
		t.Fatalf("CompareDirs failed: %v", err)
	}
	if !match {
		t.Fatalf("Rollback verification failed: %s", reason)
	}

	// Verify newfile.txt is gone and docs/file.txt is restored
	if _, err := os.Stat(filepath.Join(workspaceDir, "newfile.txt")); !os.IsNotExist(err) {
		t.Errorf("expected newfile.txt to be removed after rollback")
	}
	content, err := os.ReadFile(filepath.Join(workspaceDir, "docs", "file.txt"))
	if err != nil || string(content) != "initial" {
		t.Errorf("expected docs/file.txt with content 'initial', got %s, err: %v", string(content), err)
	}
}
