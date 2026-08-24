package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"safeshell/pkg/models"
	"safeshell/pkg/simulation"
	"safeshell/pkg/utils"
)

// ApplyUndoPlan executes the actions of an undo plan inside targetDir, using baselineDir as restore source.
func ApplyUndoPlan(plan *models.UndoPlan, targetDir, baselineDir string) error {
	if plan == nil {
		return fmt.Errorf("undo plan is nil")
	}

	for _, action := range plan.Actions {
		targetPath := filepath.Join(targetDir, action.Path)
		switch action.Type {
		case models.ActionRemoveDir:
			if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
				// Fallback to RemoveAll if non-empty
				if err := os.RemoveAll(targetPath); err != nil {
					return fmt.Errorf("failed to remove dir '%s': %w", action.Path, err)
				}
			}
		case models.ActionRemoveFile:
			if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove file '%s': %w", action.Path, err)
			}
		case models.ActionRestoreFile:
			baselinePath := filepath.Join(baselineDir, action.Path)
			if err := utils.CopyFile(baselinePath, targetPath); err != nil {
				return fmt.Errorf("failed to restore file '%s' from baseline: %w", action.Path, err)
			}
		default:
			return fmt.Errorf("unknown undo action type '%s'", action.Type)
		}
	}

	return nil
}

// Validate executes the command and then the undo plan in an isolated validation sandbox,
// verifying that the state matches the pre-execution baseline.
func Validate(cmd *models.ParsedCommand, plan *models.UndoPlan, workspaceDir, tmpDir string) (*models.ValidationResult, string, error) {
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create tmp dir: %w", err)
	}

	valID := fmt.Sprintf("val-%s", time.Now().UTC().Format("2006-01-02-150405.000000"))
	valDir := filepath.Join(tmpDir, valID)
	baseDir := filepath.Join(tmpDir, valID+"-base")

	defer func() {
		_ = os.RemoveAll(valDir)
		_ = os.RemoveAll(baseDir)
	}()

	if err := os.MkdirAll(valDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create validation dir: %w", err)
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create baseline dir: %w", err)
	}

	// Copy baseline workspace into both sandbox directories
	if _, err := os.Stat(workspaceDir); err == nil {
		if err := utils.CopyDir(workspaceDir, valDir); err != nil {
			return nil, "", fmt.Errorf("failed to copy workspace to validation sandbox: %w", err)
		}
		if err := utils.CopyDir(workspaceDir, baseDir); err != nil {
			return nil, "", fmt.Errorf("failed to copy workspace to baseline sandbox: %w", err)
		}
	}

	// 1. Apply command in validation sandbox
	if err := simulation.ApplyAction(cmd, valDir); err != nil {
		return &models.ValidationResult{Validation: "failed"}, fmt.Sprintf("command execution failed in validation: %v", err), nil
	}

	// 2. Apply undo plan in validation sandbox
	if err := ApplyUndoPlan(plan, valDir, baseDir); err != nil {
		return &models.ValidationResult{Validation: "failed"}, fmt.Sprintf("undo plan execution failed in validation: %v", err), nil
	}

	// 3. Compare validation sandbox with pristine baseline
	matched, diffReason, err := utils.CompareDirs(baseDir, valDir)
	if err != nil {
		return nil, "", fmt.Errorf("state comparison failed during validation: %w", err)
	}

	if !matched {
		return &models.ValidationResult{Validation: "failed"}, fmt.Sprintf("state mismatch after undo: %s", diffReason), nil
	}

	return &models.ValidationResult{Validation: "passed"}, "", nil
}
