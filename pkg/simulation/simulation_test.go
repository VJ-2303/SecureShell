package simulation

import (
	"os"
	"path/filepath"
	"testing"

	"safeshell/pkg/models"
)

func TestSimulate(t *testing.T) {
	tmp := t.TempDir()
	workspaceDir := filepath.Join(tmp, "workspace")
	tmpDir := filepath.Join(tmp, "tmp")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. Simulate mkdir
	mkdirCmd := &models.ParsedCommand{Raw: "mkdir docs", Name: "mkdir", Path: "docs", Args: []string{"docs"}}
	report, simDir, err := Simulate(mkdirCmd, workspaceDir, tmpDir)
	if err != nil {
		t.Fatalf("Simulate mkdir failed: %v", err)
	}
	if report.Simulation != "passed" {
		t.Errorf("expected simulation passed, got %s", report.Simulation)
	}
	if len(report.CreatedDirs) != 1 || report.CreatedDirs[0] != "docs" {
		t.Errorf("unexpected created dirs: %v", report.CreatedDirs)
	}
	// Verify real workspace was NOT touched
	if _, err := os.Stat(filepath.Join(workspaceDir, "docs")); !os.IsNotExist(err) {
		t.Errorf("real workspace should not have been modified by simulation")
	}
	_ = os.RemoveAll(simDir)

	// 2. Simulate touch
	touchCmd := &models.ParsedCommand{Raw: "touch notes.txt", Name: "touch", Path: "notes.txt", Args: []string{"notes.txt"}}
	report, simDir, err = Simulate(touchCmd, workspaceDir, tmpDir)
	if err != nil {
		t.Fatalf("Simulate touch failed: %v", err)
	}
	if len(report.CreatedFiles) != 1 || report.CreatedFiles[0] != "notes.txt" {
		t.Errorf("unexpected created files: %v", report.CreatedFiles)
	}
	_ = os.RemoveAll(simDir)

	// 3. Simulate rm on existing file
	if err := os.WriteFile(filepath.Join(workspaceDir, "test.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	rmCmd := &models.ParsedCommand{Raw: "rm test.txt", Name: "rm", Path: "test.txt", Args: []string{"test.txt"}}
	report, simDir, err = Simulate(rmCmd, workspaceDir, tmpDir)
	if err != nil {
		t.Fatalf("Simulate rm failed: %v", err)
	}
	if len(report.DeletedFiles) != 1 || report.DeletedFiles[0] != "test.txt" {
		t.Errorf("unexpected deleted files: %v", report.DeletedFiles)
	}
	_ = os.RemoveAll(simDir)
}
