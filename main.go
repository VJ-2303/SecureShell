package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"safeshell/pkg/evidence"
	"safeshell/pkg/executor"
	"safeshell/pkg/models"
	"safeshell/pkg/policy"
	"safeshell/pkg/simulation"
	"safeshell/pkg/snapshot"
	"safeshell/pkg/undoplanner"
	"safeshell/pkg/validator"
)

// SafeShell holds path configurations for the framework.
type SafeShell struct {
	BaseDir      string
	WorkspaceDir string
	SnapshotDir  string
	TmpDir       string
	EvidenceDir  string
}

// NewSafeShell constructs a SafeShell instance rooted at baseDir.
func NewSafeShell(baseDir string) *SafeShell {
	if baseDir == "" {
		baseDir = "."
	}
	return &SafeShell{
		BaseDir:      baseDir,
		WorkspaceDir: filepath.Join(baseDir, "workspace"),
		SnapshotDir:  filepath.Join(baseDir, "snapshots"),
		TmpDir:       filepath.Join(baseDir, "tmp"),
		EvidenceDir:  filepath.Join(baseDir, "evidence"),
	}
}

func (s *SafeShell) Init() error {
	dirs := []string{s.WorkspaceDir, s.SnapshotDir, s.TmpDir, s.EvidenceDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	fmt.Println("SafeShell initialized successfully.")
	fmt.Printf("Workspace: %s\nSnapshots: %s\nTmp: %s\nEvidence: %s\n", s.WorkspaceDir, s.SnapshotDir, s.TmpDir, s.EvidenceDir)
	return nil
}

func (s *SafeShell) RunCommand(args []string) int {
	var simulateOnly, planOnly, validateOnly bool
	var filteredArgs []string

	for _, arg := range args {
		switch arg {
		case "--simulate-only":
			simulateOnly = true
		case "--plan-only":
			planOnly = true
		case "--validate-only":
			validateOnly = true
		default:
			filteredArgs = append(filteredArgs, arg)
		}
	}

	if len(filteredArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no command specified to run")
		return 1
	}

	parsedCmd, err := policy.ParseCommand(filteredArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to parse command: %v\n", err)
		return 1
	}

	// 1. Policy Gate Check
	polResult := policy.Evaluate(parsedCmd, s.WorkspaceDir)
	if !polResult.Approved {
		fmt.Println("Policy: rejected")
		fmt.Printf("Execution blocked: %s\n", polResult.Reason)

		// Save rejection evidence
		evRecord := &models.EvidenceRecord{
			Command:         parsedCmd.Raw,
			Policy:          "rejected",
			RejectionReason: polResult.Reason,
			Timestamp:       time.Now().UTC(),
		}
		_, _ = evidence.SaveExecutionEvidence(s.EvidenceDir, evRecord)
		fmt.Printf("Evidence: %s\n", filepath.Join("evidence", "latest.json"))
		return 1
	}

	// 2. Snapshot Creation (or temporary state copy for preview modes)
	snap, err := snapshot.CreateSnapshot(s.WorkspaceDir, s.SnapshotDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create snapshot: %v\n", err)
		return 1
	}

	// 3. Simulation Sandbox Execution
	effect, simDir, err := simulation.Simulate(parsedCmd, s.WorkspaceDir, s.TmpDir)
	defer func() {
		if simDir != "" {
			_ = os.RemoveAll(simDir)
		}
	}()

	if err != nil || effect.Simulation != "passed" {
		fmt.Println("Policy: approved")
		fmt.Printf("Simulation: failed (%v)\n", err)
		evRecord := &models.EvidenceRecord{
			Command:    parsedCmd.Raw,
			Policy:     "approved",
			SnapshotID: snap.SnapshotID,
			Simulation: "failed",
			Timestamp:  time.Now().UTC(),
		}
		_, _ = evidence.SaveExecutionEvidence(s.EvidenceDir, evRecord)
		return 1
	}

	if simulateOnly {
		effectJSON, _ := json.MarshalIndent(effect, "", "  ")
		fmt.Println(string(effectJSON))
		return 0
	}

	// 4. Structured Undo Plan Generation
	planner := undoplanner.NewTemplatePlanner()
	undoPlan, err := planner.GeneratePlan(parsedCmd, effect)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to generate undo plan: %v\n", err)
		return 1
	}

	if planOnly {
		planJSON, _ := json.MarshalIndent(undoPlan, "", "  ")
		fmt.Println(string(planJSON))
		return 0
	}

	// 5. Undo Validation
	valResult, valReason, err := validator.Validate(parsedCmd, undoPlan, s.WorkspaceDir, s.TmpDir)
	if err != nil || valResult.Validation != "passed" {
		fmt.Println("Policy: approved")
		fmt.Println("Snapshot: created")
		fmt.Println("Simulation: passed")
		fmt.Printf("Execution blocked: undo plan failed validation (%s)\n", valReason)
		evRecord := &models.EvidenceRecord{
			Command:    parsedCmd.Raw,
			Policy:     "approved",
			SnapshotID: snap.SnapshotID,
			Simulation: "passed",
			UndoPlan:   undoPlan,
			Validation: "failed",
			Timestamp:  time.Now().UTC(),
		}
		_, _ = evidence.SaveExecutionEvidence(s.EvidenceDir, evRecord)
		fmt.Printf("Evidence: %s\n", filepath.Join("evidence", "latest.json"))
		return 1
	}

	if validateOnly {
		fmt.Println("Simulation: passed")
		fmt.Println("Undo validation: passed")
		return 0
	}

	// 6. Real Workspace Execution & Verification
	execStatus, verifyStatus, err := executor.ExecuteAndVerify(parsedCmd, s.WorkspaceDir, effect)
	if err != nil {
		fmt.Printf("Execution error: %v\n", err)
	}

	// 7. Save Auditable Evidence
	evRecord := &models.EvidenceRecord{
		Command:      parsedCmd.Raw,
		Policy:       "approved",
		SnapshotID:   snap.SnapshotID,
		Simulation:   "passed",
		UndoPlan:     undoPlan,
		Validation:   "passed",
		Execution:    execStatus,
		Verification: verifyStatus,
		RollbackUsed: false,
		Timestamp:    time.Now().UTC(),
	}
	_, err = evidence.SaveExecutionEvidence(s.EvidenceDir, evRecord)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save evidence: %v\n", err)
	}

	// 8. Output Pipeline Results
	fmt.Println("Policy: approved")
	fmt.Println("Snapshot: created")
	fmt.Println("Simulation: passed")
	fmt.Println("Undo validation: passed")
	fmt.Printf("Execution: %s\n", execStatus)
	fmt.Printf("Verification: %s\n", verifyStatus)
	fmt.Printf("Evidence: %s\n", filepath.Join("evidence", "latest.json"))

	if execStatus != "success" || verifyStatus != "matched" {
		return 1
	}
	return 0
}

func (s *SafeShell) RollbackLatest() int {
	latestSnap, err := snapshot.GetLatestSnapshot(s.SnapshotDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
		return 1
	}

	if err := snapshot.Rollback(latestSnap, s.SnapshotDir, s.WorkspaceDir); err != nil {
		fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
		evRecord := &models.RollbackEvidenceRecord{
			Command:    "rollback",
			SnapshotID: latestSnap.SnapshotID,
			Rollback:   "failed",
			Timestamp:  time.Now().UTC(),
		}
		_, _ = evidence.SaveRollbackEvidence(s.EvidenceDir, evRecord)
		return 1
	}

	evRecord := &models.RollbackEvidenceRecord{
		Command:    "rollback",
		SnapshotID: latestSnap.SnapshotID,
		Rollback:   "success",
		Timestamp:  time.Now().UTC(),
	}
	_, err = evidence.SaveRollbackEvidence(s.EvidenceDir, evRecord)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save rollback evidence: %v\n", err)
	}

	fmt.Println("Rollback: success")
	fmt.Printf("Restored snapshot: %s\n", latestSnap.SnapshotID)
	fmt.Printf("Evidence: %s\n", filepath.Join("evidence", "latest.json"))
	return 0
}

func (s *SafeShell) ShowEvidenceLatest() int {
	content, err := evidence.ReadLatestEvidence(s.EvidenceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Print(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Println()
	}
	return 0
}

func printUsage() {
	usage := `SafeShell — Trusted Transactional Command Execution Framework (MVP)

Usage:
  safeshell init
  safeshell run <command> [args...] [--simulate-only] [--plan-only] [--validate-only]
  safeshell rollback latest
  safeshell evidence latest

Commands:
  init              Initialize workspace, snapshot, tmp, and evidence directories
  run               Simulate, validate, and execute an allowed command (mkdir, touch, rm)
  rollback latest   Restore the workspace to the state of the most recent snapshot
  evidence latest   Display the most recent audit evidence JSON
`
	fmt.Print(usage)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	app := NewSafeShell(".")

	switch os.Args[1] {
	case "init":
		if err := app.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "Initialization failed: %v\n", err)
			os.Exit(1)
		}
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Error: missing command for 'run'")
			printUsage()
			os.Exit(1)
		}
		exitCode := app.RunCommand(os.Args[2:])
		os.Exit(exitCode)
	case "rollback":
		if len(os.Args) < 3 || os.Args[2] != "latest" {
			fmt.Fprintln(os.Stderr, "Error: currently only 'safeshell rollback latest' is supported")
			os.Exit(1)
		}
		exitCode := app.RollbackLatest()
		os.Exit(exitCode)
	case "evidence":
		if len(os.Args) < 3 || os.Args[2] != "latest" {
			fmt.Fprintln(os.Stderr, "Error: currently only 'safeshell evidence latest' is supported")
			os.Exit(1)
		}
		exitCode := app.ShowEvidenceLatest()
		os.Exit(exitCode)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}
