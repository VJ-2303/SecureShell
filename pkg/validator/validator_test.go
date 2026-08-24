package validator

import (
	"os"
	"path/filepath"
	"testing"

	"safeshell/pkg/models"
	"safeshell/pkg/undoplanner"
)

func TestValidator(t *testing.T) {
	tmp := t.TempDir()
	workspaceDir := filepath.Join(tmp, "workspace")
	tmpDir := filepath.Join(tmp, "tmp")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	planner := undoplanner.NewTemplatePlanner()

	// 1. Validate mkdir
	mkdirCmd := &models.ParsedCommand{Raw: "mkdir docs", Name: "mkdir", Path: "docs", Args: []string{"docs"}}
	plan, err := planner.GeneratePlan(mkdirCmd, &models.EffectReport{CreatedDirs: []string{"docs"}})
	if err != nil {
		t.Fatal(err)
	}
	res, reason, err := Validate(mkdirCmd, plan, workspaceDir, tmpDir)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if res.Validation != "passed" {
		t.Errorf("expected validation passed, got %s (reason: %s)", res.Validation, reason)
	}

	// 2. Validate touch
	touchCmd := &models.ParsedCommand{Raw: "touch notes.txt", Name: "touch", Path: "notes.txt", Args: []string{"notes.txt"}}
	plan, err = planner.GeneratePlan(touchCmd, &models.EffectReport{CreatedFiles: []string{"notes.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	res, reason, err = Validate(touchCmd, plan, workspaceDir, tmpDir)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if res.Validation != "passed" {
		t.Errorf("expected validation passed, got %s (reason: %s)", res.Validation, reason)
	}

	// 3. Validate rm on existing file
	if err := os.WriteFile(filepath.Join(workspaceDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	rmCmd := &models.ParsedCommand{Raw: "rm file.txt", Name: "rm", Path: "file.txt", Args: []string{"file.txt"}}
	plan, err = planner.GeneratePlan(rmCmd, &models.EffectReport{DeletedFiles: []string{"file.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	res, reason, err = Validate(rmCmd, plan, workspaceDir, tmpDir)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if res.Validation != "passed" {
		t.Errorf("expected validation passed, got %s (reason: %s)", res.Validation, reason)
	}

	// 4. Test faulty undo plan fails validation
	badPlan := &models.UndoPlan{
		Strategy: "template",
		Actions: []models.UndoAction{
			{Type: models.ActionRemoveFile, Path: "nonexistent.txt"},
		},
	}
	res, _, err = Validate(touchCmd, badPlan, workspaceDir, tmpDir)
	if err != nil {
		t.Fatalf("Validate unexpected error: %v", err)
	}
	if res.Validation != "failed" {
		t.Errorf("expected bad plan to fail validation")
	}
}
