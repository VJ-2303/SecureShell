package models

import "time"

// ActionType defines the type of undo action.
type ActionType string

const (
	ActionRemoveDir   ActionType = "remove_dir"
	ActionRemoveFile  ActionType = "remove_file"
	ActionRestoreFile ActionType = "restore_file"
)

// UndoAction represents a single reversible operation.
type UndoAction struct {
	Type ActionType `json:"type"`
	Path string     `json:"path"`
}

// UndoPlan holds a sequence of undo actions and the strategy used to generate them.
type UndoPlan struct {
	Strategy string       `json:"strategy"`
	Actions  []UndoAction `json:"actions"`
}

// Snapshot represents metadata for a captured workspace snapshot.
type Snapshot struct {
	SnapshotID string    `json:"snapshot_id"`
	CreatedAt  time.Time `json:"created_at"`
	Workspace  string    `json:"workspace"`
}

// EffectReport documents the simulated or actual changes made by a command.
type EffectReport struct {
	Command      string   `json:"command"`
	CreatedDirs  []string `json:"created_dirs"`
	CreatedFiles []string `json:"created_files"`
	DeletedFiles []string `json:"deleted_files"`
	Simulation   string   `json:"simulation"` // "passed" or "failed"
}

// ValidationResult indicates whether an undo plan passed isolated validation.
type ValidationResult struct {
	Validation string `json:"validation"` // "passed" or "failed"
}

// EvidenceRecord represents the complete audit trail for a command execution.
type EvidenceRecord struct {
	Command         string            `json:"command"`
	Policy          string            `json:"policy"` // "approved" or "rejected"
	RejectionReason string            `json:"rejection_reason,omitempty"`
	SnapshotID      string            `json:"snapshot_id,omitempty"`
	Simulation      string            `json:"simulation,omitempty"`
	UndoPlan        *UndoPlan         `json:"undo_plan,omitempty"`
	Validation      string            `json:"validation,omitempty"`
	Execution       string            `json:"execution,omitempty"`
	Verification    string            `json:"verification,omitempty"`
	RollbackUsed    bool              `json:"rollback_used,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
}

// RollbackEvidenceRecord represents the audit trail for a rollback operation.
type RollbackEvidenceRecord struct {
	Command    string    `json:"command"`
	SnapshotID string    `json:"snapshot_id"`
	Rollback   string    `json:"rollback"` // "success" or "failed"
	Timestamp  time.Time `json:"timestamp"`
}

// ParsedCommand represents a normalized shell command for SafeShell.
type ParsedCommand struct {
	Raw  string   `json:"raw"`
	Name string   `json:"name"`
	Args []string `json:"args"`
	Path string   `json:"path"`
}

// PolicyResult contains the decision made by the Policy Gate.
type PolicyResult struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}
