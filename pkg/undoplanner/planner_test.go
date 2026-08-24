package undoplanner

import (
	"testing"

	"safeshell/pkg/models"
)

func TestTemplatePlanner(t *testing.T) {
	planner := NewTemplatePlanner()

	// 1. mkdir
	mkdirCmd := &models.ParsedCommand{Raw: "mkdir docs", Name: "mkdir", Path: "docs"}
	mkdirEffect := &models.EffectReport{CreatedDirs: []string{"docs"}}
	plan, err := planner.GeneratePlan(mkdirCmd, mkdirEffect)
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}
	if plan.Strategy != "template" {
		t.Errorf("expected strategy 'template', got %s", plan.Strategy)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Type != models.ActionRemoveDir || plan.Actions[0].Path != "docs" {
		t.Errorf("unexpected actions: %+v", plan.Actions)
	}

	// 2. touch
	touchCmd := &models.ParsedCommand{Raw: "touch notes.txt", Name: "touch", Path: "notes.txt"}
	touchEffect := &models.EffectReport{CreatedFiles: []string{"notes.txt"}}
	plan, err = planner.GeneratePlan(touchCmd, touchEffect)
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Type != models.ActionRemoveFile || plan.Actions[0].Path != "notes.txt" {
		t.Errorf("unexpected actions: %+v", plan.Actions)
	}

	// 3. rm
	rmCmd := &models.ParsedCommand{Raw: "rm file.txt", Name: "rm", Path: "file.txt"}
	rmEffect := &models.EffectReport{DeletedFiles: []string{"file.txt"}}
	plan, err = planner.GeneratePlan(rmCmd, rmEffect)
	if err != nil {
		t.Fatalf("GeneratePlan failed: %v", err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Type != models.ActionRestoreFile || plan.Actions[0].Path != "file.txt" {
		t.Errorf("unexpected actions: %+v", plan.Actions)
	}
}
