package executor

import (
	"os"
	"path/filepath"
	"testing"

	"safeshell/pkg/models"
)

func TestExecuteAndVerify(t *testing.T) {
	tmp := t.TempDir()
	workspaceDir := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. mkdir
	cmd := &models.ParsedCommand{Raw: "mkdir docs", Name: "mkdir", Path: "docs", Args: []string{"docs"}}
	effect := &models.EffectReport{CreatedDirs: []string{"docs"}}
	execStatus, verifyStatus, err := ExecuteAndVerify(cmd, workspaceDir, effect)
	if err != nil {
		t.Fatalf("ExecuteAndVerify failed: %v", err)
	}
	if execStatus != "success" || verifyStatus != "matched" {
		t.Errorf("expected success/matched, got %s/%s", execStatus, verifyStatus)
	}

	// 2. touch
	cmd = &models.ParsedCommand{Raw: "touch notes.txt", Name: "touch", Path: "notes.txt", Args: []string{"notes.txt"}}
	effect = &models.EffectReport{CreatedFiles: []string{"notes.txt"}}
	execStatus, verifyStatus, err = ExecuteAndVerify(cmd, workspaceDir, effect)
	if err != nil {
		t.Fatalf("ExecuteAndVerify failed: %v", err)
	}
	if execStatus != "success" || verifyStatus != "matched" {
		t.Errorf("expected success/matched, got %s/%s", execStatus, verifyStatus)
	}

	// 3. rm
	cmd = &models.ParsedCommand{Raw: "rm notes.txt", Name: "rm", Path: "notes.txt", Args: []string{"notes.txt"}}
	effect = &models.EffectReport{DeletedFiles: []string{"notes.txt"}}
	execStatus, verifyStatus, err = ExecuteAndVerify(cmd, workspaceDir, effect)
	if err != nil {
		t.Fatalf("ExecuteAndVerify failed: %v", err)
	}
	if execStatus != "success" || verifyStatus != "matched" {
		t.Errorf("expected success/matched, got %s/%s", execStatus, verifyStatus)
	}
}
