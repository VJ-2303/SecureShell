package undoplanner

import (
	"fmt"
	"sort"
	"strings"

	"safeshell/pkg/models"
)

// UndoPlanner defines the interface for generating undo plans.
type UndoPlanner interface {
	GeneratePlan(cmd *models.ParsedCommand, effect *models.EffectReport) (*models.UndoPlan, error)
}

// TemplatePlanner implements deterministic, template-based undo plan generation.
type TemplatePlanner struct{}

// NewTemplatePlanner creates a new instance of TemplatePlanner.
func NewTemplatePlanner() *TemplatePlanner {
	return &TemplatePlanner{}
}

// GeneratePlan generates a structured UndoPlan based on the parsed command and observed effects.
func (p *TemplatePlanner) GeneratePlan(cmd *models.ParsedCommand, effect *models.EffectReport) (*models.UndoPlan, error) {
	if cmd == nil {
		return nil, fmt.Errorf("command is nil")
	}

	plan := &models.UndoPlan{
		Strategy: "template",
		Actions:  []models.UndoAction{},
	}

	switch cmd.Name {
	case "mkdir":
		if effect != nil && len(effect.CreatedDirs) > 0 {
			// Sort descending so deepest subdirectories are removed first
			dirs := make([]string, len(effect.CreatedDirs))
			copy(dirs, effect.CreatedDirs)
			sort.Slice(dirs, func(i, j int) bool {
				return len(strings.Split(dirs[i], "/")) > len(strings.Split(dirs[j], "/"))
			})
			for _, d := range dirs {
				plan.Actions = append(plan.Actions, models.UndoAction{
					Type: models.ActionRemoveDir,
					Path: d,
				})
			}
		} else {
			plan.Actions = append(plan.Actions, models.UndoAction{
				Type: models.ActionRemoveDir,
				Path: cmd.Path,
			})
		}
		return plan, nil

	case "touch":
		if effect != nil && len(effect.CreatedFiles) > 0 {
			for _, f := range effect.CreatedFiles {
				plan.Actions = append(plan.Actions, models.UndoAction{
					Type: models.ActionRemoveFile,
					Path: f,
				})
			}
			// If parent directories were also created as part of touch
			if len(effect.CreatedDirs) > 0 {
				dirs := make([]string, len(effect.CreatedDirs))
				copy(dirs, effect.CreatedDirs)
				sort.Slice(dirs, func(i, j int) bool {
					return len(strings.Split(dirs[i], "/")) > len(strings.Split(dirs[j], "/"))
				})
				for _, d := range dirs {
					plan.Actions = append(plan.Actions, models.UndoAction{
						Type: models.ActionRemoveDir,
						Path: d,
					})
				}
			}
		} else {
			plan.Actions = append(plan.Actions, models.UndoAction{
				Type: models.ActionRemoveFile,
				Path: cmd.Path,
			})
		}
		return plan, nil

	case "rm":
		if effect != nil && len(effect.DeletedFiles) > 0 {
			for _, f := range effect.DeletedFiles {
				plan.Actions = append(plan.Actions, models.UndoAction{
					Type: models.ActionRestoreFile,
					Path: f,
				})
			}
		} else {
			plan.Actions = append(plan.Actions, models.UndoAction{
				Type: models.ActionRestoreFile,
				Path: cmd.Path,
			})
		}
		return plan, nil

	default:
		return nil, fmt.Errorf("no undo template available for command '%s'", cmd.Name)
	}
}
