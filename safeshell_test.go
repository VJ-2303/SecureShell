package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"safeshell/pkg/models"
	"safeshell/pkg/utils"
)

func setupTestSafeShell(t *testing.T) (*SafeShell, string) {
	tmpDir := t.TempDir()
	app := NewSafeShell(tmpDir)
	if err := app.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	return app, tmpDir
}

// Test Acceptance Test 1: Safe mkdir
func TestAcceptanceTest1_SafeMkdir(t *testing.T) {
	app, tmpDir := setupTestSafeShell(t)

	code := app.RunCommand([]string{"mkdir", "docs"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Verify directory created
	docPath := filepath.Join(tmpDir, "workspace", "docs")
	info, err := os.Stat(docPath)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected workspace/docs directory to exist")
	}

	// Verify evidence
	latest, err := evidenceRead(app.EvidenceDir)
	if err != nil {
		t.Fatalf("failed to read latest evidence: %v", err)
	}
	if latest.Command != "mkdir docs" || latest.Policy != "approved" || latest.Execution != "success" || latest.Verification != "matched" {
		t.Fatalf("unexpected evidence: %+v", latest)
	}
	if latest.UndoPlan == nil || len(latest.UndoPlan.Actions) != 1 || latest.UndoPlan.Actions[0].Type != models.ActionRemoveDir || latest.UndoPlan.Actions[0].Path != "docs" {
		t.Fatalf("unexpected undo plan: %+v", latest.UndoPlan)
	}
}

// Test Acceptance Test 2: Safe touch
func TestAcceptanceTest2_SafeTouch(t *testing.T) {
	app, tmpDir := setupTestSafeShell(t)

	code := app.RunCommand([]string{"touch", "notes.txt"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Verify file created
	filePath := filepath.Join(tmpDir, "workspace", "notes.txt")
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		t.Fatalf("expected workspace/notes.txt file to exist")
	}

	// Verify evidence
	latest, err := evidenceRead(app.EvidenceDir)
	if err != nil {
		t.Fatalf("failed to read latest evidence: %v", err)
	}
	if latest.Command != "touch notes.txt" || latest.Policy != "approved" {
		t.Fatalf("unexpected evidence: %+v", latest)
	}
	if latest.UndoPlan == nil || len(latest.UndoPlan.Actions) != 1 || latest.UndoPlan.Actions[0].Type != models.ActionRemoveFile || latest.UndoPlan.Actions[0].Path != "notes.txt" {
		t.Fatalf("unexpected undo plan: %+v", latest.UndoPlan)
	}
}

// Test Acceptance Test 3: Rollback
func TestAcceptanceTest3_Rollback(t *testing.T) {
	app, tmpDir := setupTestSafeShell(t)

	// Step 1: Create docs
	if code := app.RunCommand([]string{"mkdir", "docs"}); code != 0 {
		t.Fatalf("mkdir failed")
	}
	// Step 2: Create notes.txt
	if code := app.RunCommand([]string{"touch", "notes.txt"}); code != 0 {
		t.Fatalf("touch failed")
	}

	// Verify both exist
	if _, err := os.Stat(filepath.Join(tmpDir, "workspace", "docs")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "workspace", "notes.txt")); err != nil {
		t.Fatal(err)
	}

	// Rollback latest (which was before touch notes.txt)
	if code := app.RollbackLatest(); code != 0 {
		t.Fatalf("Rollback failed")
	}

	// docs should exist (since snapshot before touch had docs), but notes.txt should NOT
	if _, err := os.Stat(filepath.Join(tmpDir, "workspace", "docs")); err != nil {
		t.Errorf("expected docs to still exist after restoring snapshot taken before touch")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "workspace", "notes.txt")); !os.IsNotExist(err) {
		t.Errorf("expected notes.txt to be removed after rollback")
	}

	// Verify rollback evidence
	evContent, err := os.ReadFile(filepath.Join(app.EvidenceDir, "latest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rb models.RollbackEvidenceRecord
	if err := json.Unmarshal(evContent, &rb); err != nil {
		t.Fatal(err)
	}
	if rb.Command != "rollback" || rb.Rollback != "success" {
		t.Errorf("unexpected rollback evidence: %+v", rb)
	}
}

// Test Acceptance Test 4: Dangerous command (rm -rf /)
func TestAcceptanceTest4_DangerousCommand(t *testing.T) {
	app, _ := setupTestSafeShell(t)

	code := app.RunCommand([]string{"rm", "-rf", "/"})
	if code == 0 {
		t.Fatalf("expected non-zero exit code for dangerous command")
	}

	// Verify rejection evidence
	evContent, err := os.ReadFile(filepath.Join(app.EvidenceDir, "latest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ev models.EvidenceRecord
	if err := json.Unmarshal(evContent, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Policy != "rejected" {
		t.Errorf("expected policy rejected, got %s", ev.Policy)
	}
	if ev.RejectionReason == "" {
		t.Errorf("expected non-empty rejection reason")
	}
}

// Test Acceptance Test 5: Path escape (mkdir ../../evil)
func TestAcceptanceTest5_PathEscape(t *testing.T) {
	app, _ := setupTestSafeShell(t)

	code := app.RunCommand([]string{"mkdir", "../../evil"})
	if code == 0 {
		t.Fatalf("expected non-zero exit code for path escape")
	}

	// Verify rejection evidence
	evContent, err := os.ReadFile(filepath.Join(app.EvidenceDir, "latest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ev models.EvidenceRecord
	if err := json.Unmarshal(evContent, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Policy != "rejected" {
		t.Errorf("expected policy rejected, got %s", ev.Policy)
	}
	if !strings.Contains(ev.RejectionReason, "prohibited") && !strings.Contains(ev.RejectionReason, "escapes") {
		t.Errorf("unexpected rejection reason: %s", ev.RejectionReason)
	}
}

// Test Safe rm and undo/rollback
func TestSafeRmAndUndo(t *testing.T) {
	app, tmpDir := setupTestSafeShell(t)

	// Create initial file
	filePath := filepath.Join(tmpDir, "workspace", "file.txt")
	if err := os.WriteFile(filePath, []byte("important content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run rm file.txt
	code := app.RunCommand([]string{"rm", "file.txt"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	// Verify file is removed
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected file.txt to be removed")
	}

	// Verify evidence
	latest, err := evidenceRead(app.EvidenceDir)
	if err != nil {
		t.Fatalf("failed to read latest evidence: %v", err)
	}
	if latest.Command != "rm file.txt" || latest.Policy != "approved" {
		t.Fatalf("unexpected evidence: %+v", latest)
	}
	if latest.UndoPlan == nil || len(latest.UndoPlan.Actions) != 1 || latest.UndoPlan.Actions[0].Type != models.ActionRestoreFile {
		t.Fatalf("unexpected undo plan for rm: %+v", latest.UndoPlan)
	}

	// Rollback
	if code := app.RollbackLatest(); code != 0 {
		t.Fatalf("Rollback failed")
	}

	// Verify file is restored with original content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("file not restored: %v", err)
	}
	if string(content) != "important content" {
		t.Fatalf("restored file content mismatch: got '%s', want 'important content'", string(content))
	}
}

// Test flags: --simulate-only, --plan-only, --validate-only
func TestPreviewFlags(t *testing.T) {
	app, tmpDir := setupTestSafeShell(t)

	// 1. --simulate-only
	code := app.RunCommand([]string{"mkdir", "preview_dir", "--simulate-only"})
	if code != 0 {
		t.Fatalf("expected exit code 0 for simulate-only")
	}
	// Verify workspace was NOT modified
	if _, err := os.Stat(filepath.Join(tmpDir, "workspace", "preview_dir")); !os.IsNotExist(err) {
		t.Errorf("simulate-only should not modify real workspace")
	}

	// 2. --plan-only
	code = app.RunCommand([]string{"touch", "preview_file.txt", "--plan-only"})
	if code != 0 {
		t.Fatalf("expected exit code 0 for plan-only")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "workspace", "preview_file.txt")); !os.IsNotExist(err) {
		t.Errorf("plan-only should not modify real workspace")
	}

	// 3. --validate-only
	code = app.RunCommand([]string{"mkdir", "validated_dir", "--validate-only"})
	if code != 0 {
		t.Fatalf("expected exit code 0 for validate-only")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "workspace", "validated_dir")); !os.IsNotExist(err) {
		t.Errorf("validate-only should not modify real workspace")
	}
}

func evidenceRead(evidenceDir string) (*models.EvidenceRecord, error) {
	data, err := os.ReadFile(filepath.Join(evidenceDir, "latest.json"))
	if err != nil {
		return nil, err
	}
	var ev models.EvidenceRecord
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}

func TestCompareDirsIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "a")
	dirB := filepath.Join(tmpDir, "b")

	_ = os.MkdirAll(dirA, 0755)
	_ = os.MkdirAll(dirB, 0755)

	match, _, err := utils.CompareDirs(dirA, dirB)
	if err != nil || !match {
		t.Fatalf("empty dirs should match, match=%v, err=%v", match, err)
	}
}
