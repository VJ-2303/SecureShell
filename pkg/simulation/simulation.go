package simulation

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"safeshell/pkg/models"
	"safeshell/pkg/utils"
)

// ApplyAction applies a command to a target directory without executing raw shell.
func ApplyAction(cmd *models.ParsedCommand, targetDir string) error {
	targetPath := filepath.Join(targetDir, cmd.Path)

	switch cmd.Name {
	case "mkdir":
		// Handle mkdir (supports nested creation if requested)
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		return nil

	case "touch":
		parentDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directories: %w", err)
		}
		// Touch or create empty file
		f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to touch file: %w", err)
		}
		f.Close()
		now := time.Now()
		_ = os.Chtimes(targetPath, now, now)
		return nil

	case "rm":
		info, err := os.Stat(targetPath)
		if err != nil {
			return fmt.Errorf("cannot remove '%s': no such file or directory", cmd.Path)
		}
		if info.IsDir() {
			return fmt.Errorf("cannot remove '%s': is a directory", cmd.Path)
		}
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported command '%s'", cmd.Name)
	}
}

// Simulate runs a command in an isolated temporary copy of the workspace.
func Simulate(cmd *models.ParsedCommand, workspaceDir, tmpDir string) (*models.EffectReport, string, error) {
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create tmp dir: %w", err)
	}

	simID := fmt.Sprintf("sim-%s", time.Now().UTC().Format("2006-01-02-150405.000000"))
	simDir := filepath.Join(tmpDir, simID)

	if err := os.MkdirAll(simDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create sim sandbox: %w", err)
	}

	// Copy workspace to sim sandbox
	if _, err := os.Stat(workspaceDir); err == nil {
		if err := utils.CopyDir(workspaceDir, simDir); err != nil {
			return nil, simDir, fmt.Errorf("failed to copy workspace to simulation: %w", err)
		}
	}

	// Baseline scan
	beforeDirs, beforeFiles, err := utils.ScanDir(simDir)
	if err != nil {
		return nil, simDir, fmt.Errorf("failed to scan pre-simulation state: %w", err)
	}

	beforeDirMap := make(map[string]bool)
	for _, d := range beforeDirs {
		beforeDirMap[d] = true
	}
	beforeFileMap := make(map[string]bool)
	for _, f := range beforeFiles {
		beforeFileMap[f] = true
	}

	// Apply command inside simulation
	applyErr := ApplyAction(cmd, simDir)
	if applyErr != nil {
		report := &models.EffectReport{
			Command:      cmd.Raw,
			CreatedDirs:  []string{},
			CreatedFiles: []string{},
			DeletedFiles: []string{},
			Simulation:   "failed",
		}
		return report, simDir, applyErr
	}

	// Post-simulation scan
	afterDirs, afterFiles, err := utils.ScanDir(simDir)
	if err != nil {
		return nil, simDir, fmt.Errorf("failed to scan post-simulation state: %w", err)
	}

	var createdDirs []string
	for _, d := range afterDirs {
		if !beforeDirMap[d] {
			createdDirs = append(createdDirs, d)
		}
	}

	var createdFiles []string
	for _, f := range afterFiles {
		if !beforeFileMap[f] {
			createdFiles = append(createdFiles, f)
		}
	}

	var deletedFiles []string
	afterFileMap := make(map[string]bool)
	for _, f := range afterFiles {
		afterFileMap[f] = true
	}
	for _, f := range beforeFiles {
		if !afterFileMap[f] {
			deletedFiles = append(deletedFiles, f)
		}
	}

	sort.Strings(createdDirs)
	sort.Strings(createdFiles)
	sort.Strings(deletedFiles)

	if createdDirs == nil {
		createdDirs = []string{}
	}
	if createdFiles == nil {
		createdFiles = []string{}
	}
	if deletedFiles == nil {
		deletedFiles = []string{}
	}

	report := &models.EffectReport{
		Command:      cmd.Raw,
		CreatedDirs:  createdDirs,
		CreatedFiles: createdFiles,
		DeletedFiles: deletedFiles,
		Simulation:   "passed",
	}

	return report, simDir, nil
}
