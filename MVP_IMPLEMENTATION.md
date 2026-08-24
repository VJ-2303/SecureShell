# 4-Hour MVP Implementation Plan

## Objective

Build the smallest working SafeShell demo that proves:

1. Unsafe commands are rejected.
2. A snapshot is taken before execution.
3. The command is simulated.
4. An undo plan is generated.
5. The undo plan is validated.
6. The command is executed.
7. Rollback restores the original state.
8. Evidence is saved.

---

## MVP Scope

Build only this:

| Feature | Include |
|---|---|
| CLI | Yes |
| Workspace directory | Yes |
| Supported commands | `mkdir`, `touch` |
| Optional command if fast | `rm` for a single file |
| Snapshot | Copy workspace directory |
| Simulation | Run command in temporary copy |
| Undo plan | Structured JSON |
| Undo validation | Apply undo in temporary copy |
| Rollback | Restore snapshot copy |
| Evidence | JSON file |
| Policy gate | Path safety and command allowlist |

---

## Do Not Build

Do not spend time on:

- Secure Boot
- TPM
- attestation
- OverlayFS
- Btrfs
- systemd
- network commands
- package manager commands
- kernel modules
- sysctl
- daemon architecture
- database
- web UI
- full LLM integration
- production deployment hardening

If asked, say these are part of the full SafeShell architecture but are not included in the 4-hour MVP.

---

## Command Set

Support only:

```bash
safeshell init
safeshell run mkdir <path>
safeshell run touch <path>
safeshell rollback latest
safeshell evidence latest
```

Optional if time remains:

```bash
safeshell run rm <path>
```

Reject:

```bash
safeshell run rm -rf /
safeshell run mkdir ../../evil
safeshell run curl example.com
```

---

## Core Flow

For every command:

```text
Parse
  ↓
Policy check
  ↓
Snapshot workspace
  ↓
Simulate in temporary copy
  ↓
Generate undo plan
  ↓
Validate undo plan
  ↓
Execute on real workspace
  ↓
Verify result
  ↓
Save evidence
```

For rollback:

```text
Load latest snapshot
  ↓
Restore workspace
  ↓
Save recovery evidence
```

---

# 4-Hour Timeline

## Hour 0:00 to 0:20 — Project Setup

Build:

- Project folder.
- Single CLI entrypoint.
- `safeshell init`.
- Workspace directory.
- Evidence directory.
- Snapshot directory.

Directory structure:

```text
safeshell/
├── main.go
├── workspace/
├── snapshots/
├── tmp/
└── evidence/
```

Definition of done:

```bash
safeshell init
```

Creates required directories.

---

## Hour 0:20 to 0:50 — Command Parser and Policy Gate

Build:

- Parse CLI arguments.
- Allow only:
  - `mkdir`
  - `touch`
  - optional `rm`
- Reject unsupported commands.
- Reject paths outside workspace.
- Reject `..`
- Reject absolute paths.
- Reject recursive delete.

Policy rules:

| Rule | Result |
|---|---|
| Command not allowed | Reject |
| Path is empty | Reject |
| Path contains `..` | Reject |
| Path is absolute | Reject |
| Path outside workspace | Reject |
| Recursive delete | Reject |

Definition of done:

```bash
safeshell run mkdir docs
```

Outputs:

```text
Policy: approved
```

```bash
safeshell run mkdir ../../evil
```

Outputs:

```text
Policy: rejected
```

---

## Hour 0:50 to 1:20 — Snapshot and Rollback

Build:

- Create snapshot before execution.
- Copy workspace into:

```text
snapshots/<timestamp>
```

- Save snapshot metadata:

```json
{
  "snapshot_id": "2026-01-01-120000",
  "created_at": "2026-01-01T12:00:00Z",
  "workspace": "workspace"
}
```

Build rollback:

```bash
safeshell rollback latest
```

Rollback must:

1. Find latest snapshot.
2. Clear workspace.
3. Copy snapshot back.
4. Save recovery evidence.

Definition of done:

- Manual file creation in workspace.
- Run snapshot.
- Modify workspace.
- Run rollback.
- Workspace is restored.

---

## Hour 1:20 to 1:50 — Simulation

Build:

- Copy workspace to temporary directory:

```text
tmp/sim-<timestamp>
```

- Execute command inside temporary copy.
- Record effect.

Effect report:

```json
{
  "command": "mkdir docs",
  "created_dirs": ["docs"],
  "created_files": [],
  "deleted_files": [],
  "simulation": "passed"
}
```

For MVP:

| Command | Simulation Action |
|---|---|
| `mkdir` | create directory in temp copy |
| `touch` | create empty file in temp copy |
| `rm` | delete file in temp copy |

Definition of done:

```bash
safeshell run mkdir docs --simulate-only
```

Shows effect report.

---

## Hour 1:50 to 2:20 — Undo Plan Generation

Build structured undo planner.

Do not use raw shell commands.

Undo plan format:

```json
{
  "strategy": "template",
  "actions": [
    {
      "type": "remove_dir",
      "path": "docs"
    }
  ]
}
```

Undo rules:

| Command | Undo Action |
|---|---|
| `mkdir docs` | remove `docs` |
| `touch file.txt` | remove `file.txt` |
| `rm file.txt` | restore `file.txt` from snapshot |

If time permits, create an interface:

```text
UndoPlanner
├── TemplatePlanner
└── LLMPlanner
```

But implement only `TemplatePlanner`.

Definition of done:

```bash
safeshell run mkdir docs --plan-only
```

Shows undo plan.

---

## Hour 2:20 to 2:50 — Undo Validation

Build validator:

1. Copy workspace to temporary validation directory.
2. Apply command.
3. Apply undo plan.
4. Compare result with baseline.

For MVP comparison:

- Compare file list.
- Compare directory list.
- Ignore SafeShell metadata directories.

Validation result:

```json
{
  "validation": "passed"
}
```

If validation fails:

```text
Execution blocked: undo plan failed validation
```

Definition of done:

```bash
safeshell run mkdir docs --validate-only
```

Outputs:

```text
Simulation: passed
Undo validation: passed
```

---

## Hour 2:50 to 3:20 — Real Execution and Verification

Build executor:

- Execute only if:
  - policy passed
  - snapshot created
  - simulation passed
  - undo validation passed

Execute command in real workspace.

Verify:

- Expected created directory exists.
- Expected created file exists.
- Expected deleted file is removed.

Save evidence.

Definition of done:

```bash
safeshell run mkdir docs
```

Outputs:

```text
Policy: approved
Snapshot: created
Simulation: passed
Undo validation: passed
Execution: success
Verification: matched
Evidence: evidence/latest.json
```

---

## Hour 3:20 to 3:40 — Evidence Reporter

Every run must save:

```json
{
  "command": "mkdir docs",
  "policy": "approved",
  "snapshot_id": "2026-01-01-120000",
  "simulation": "passed",
  "undo_plan": {
    "strategy": "template",
    "actions": [
      {
        "type": "remove_dir",
        "path": "docs"
      }
    ]
  },
  "validation": "passed",
  "execution": "success",
  "verification": "matched",
  "rollback_used": false,
  "timestamp": "2026-01-01T12:00:00Z"
}
```

For rollback:

```json
{
  "command": "rollback",
  "snapshot_id": "2026-01-01-120000",
  "rollback": "success",
  "timestamp": "2026-01-01T12:05:00Z"
}
```

Definition of done:

```bash
safeshell evidence latest
```

Shows latest evidence file.

---

## Hour 3:40 to 4:00 — Demo Script and Final Checks

Create `demo.sh`.

Demo script:

```bash
rm -rf workspace snapshots tmp evidence

safeshell init

safeshell run mkdir docs
safeshell run touch notes.txt

find workspace

safeshell rollback latest

find workspace

safeshell evidence latest
```

Expected demo result:

1. `docs` and `notes.txt` are created.
2. Rollback removes them.
3. Evidence shows the full record.

Also test:

```bash
safeshell run rm -rf /
safeshell run mkdir ../../evil
```

Both must be rejected.

---

# Minimum Acceptance Tests

## Test 1: Safe mkdir

```bash
safeshell run mkdir docs
```

Expected:

- Command executes.
- Undo plan generated.
- Evidence saved.

---

## Test 2: Safe touch

```bash
safeshell run touch notes.txt
```

Expected:

- File created.
- Undo plan removes file.
- Evidence saved.

---

## Test 3: Rollback

```bash
safeshell rollback latest
```

Expected:

- Workspace restored.
- Recovery evidence saved.

---

## Test 4: Dangerous command

```bash
safeshell run rm -rf /
```

Expected:

- Rejected.
- No execution.
- Evidence saved.

---

## Test 5: Path escape

```bash
safeshell run mkdir ../../evil
```

Expected:

- Rejected.
- No execution.
- Evidence saved.

---

# What to Say During Submission

Use this explanation:

> This MVP demonstrates SafeShell’s core transactional safety model: policy-controlled command intake, snapshot-backed recovery, isolated simulation, structured undo planning, undo validation, and auditable evidence. The full architecture includes TPM-based measured execution, Secure Boot alignment, hardened toolchains, broader Linux state modeling, and layered sandboxing, but those are not implemented in this 4-hour MVP.

---

# Absolute Priority Order

If you run out of time, keep only this:

1. `mkdir`
2. `touch`
3. Snapshot
4. Rollback
5. Evidence
6. Policy rejection
7. Simulation
8. Undo validation
9. CLI polish
10. Optional `rm`

If you must cut something, cut in this order:

1. Optional `rm`
2. Undo validation
3. Simulation
4. Evidence polish
5. CLI polish

Do not cut:

- Snapshot
- Rollback
- Policy rejection
- Evidence file
- Working demo
