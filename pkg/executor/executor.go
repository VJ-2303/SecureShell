package executor

import (
	"fmt"
	"os"
	"path/filepath"

	"safeshell/pkg/models"
	"safeshell/pkg/simulation"
)

// ExecuteAndVerify executes the command on the real workspace and verifies that the changes match the simulated effect report.
func ExecuteAndVerify(cmd *models.ParsedCommand, workspaceDir string, effect *models.EffectReport) (string, string, error) {
	if cmd == nil {
		return "failed", "mismatch", fmt.Errorf("command is nil")
	}

	// 1. Execute the command action on real workspace
	if err := simulation.ApplyAction(cmd, workspaceDir); err != nil {
		return "failed", "mismatch", fmt.Errorf("execution failed: %w", err)
	}

	// 2. Verify all expected created directories exist
	if effect != nil {
		for _, d := range effect.CreatedDirs {
			p := filepath.Join(workspaceDir, d)
			info, err := os.Stat(p)
			if err != nil || !info.IsDir() {
				return "success", "mismatch", fmt.Errorf("verification failed: expected directory '%s' not found", d)
			}
		}

		// Verify all expected created files exist
		for _, f := range effect.CreatedFiles {
			p := filepath.Join(workspaceDir, f)
			info, err := os.Stat(p)
			if err != nil || info.IsDir() {
				return "success", "mismatch", fmt.Errorf("verification failed: expected file '%s' not found", f)
			}
		}

		// Verify all expected deleted files are gone
		for _, f := range effect.DeletedFiles {
			p := filepath.Join(workspaceDir, f)
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				return "success", "mismatch", fmt.Errorf("verification failed: deleted file '%s' still exists", f)
			}
		}
	}

	return "success", "matched", nil
}
