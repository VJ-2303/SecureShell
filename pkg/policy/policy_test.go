package policy

import (
	"testing"
)

func TestPolicyEvaluate(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		workspaceDir string
		wantApproved bool
	}{
		{
			name:         "valid mkdir",
			args:         []string{"mkdir", "docs"},
			workspaceDir: "/tmp/workspace",
			wantApproved: true,
		},
		{
			name:         "valid touch",
			args:         []string{"touch", "notes.txt"},
			workspaceDir: "/tmp/workspace",
			wantApproved: true,
		},
		{
			name:         "valid touch nested",
			args:         []string{"touch", "docs/sub/notes.txt"},
			workspaceDir: "/tmp/workspace",
			wantApproved: true,
		},
		{
			name:         "valid single file rm",
			args:         []string{"rm", "file.txt"},
			workspaceDir: "/tmp/workspace",
			wantApproved: true,
		},
		{
			name:         "rejected curl",
			args:         []string{"curl", "example.com"},
			workspaceDir: "/tmp/workspace",
			wantApproved: false,
		},
		{
			name:         "rejected rm -rf /",
			args:         []string{"rm", "-rf", "/"},
			workspaceDir: "/tmp/workspace",
			wantApproved: false,
		},
		{
			name:         "rejected rm -r dir",
			args:         []string{"rm", "-r", "dir"},
			workspaceDir: "/tmp/workspace",
			wantApproved: false,
		},
		{
			name:         "rejected path escape with ..",
			args:         []string{"mkdir", "../../evil"},
			workspaceDir: "/tmp/workspace",
			wantApproved: false,
		},
		{
			name:         "rejected embedded traversal",
			args:         []string{"touch", "docs/../evil.txt"},
			workspaceDir: "/tmp/workspace",
			wantApproved: false,
		},
		{
			name:         "rejected absolute path",
			args:         []string{"mkdir", "/etc/evil"},
			workspaceDir: "/tmp/workspace",
			wantApproved: false,
		},
		{
			name:         "rejected empty path",
			args:         []string{"mkdir"},
			workspaceDir: "/tmp/workspace",
			wantApproved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCommand(tt.args)
			if tt.name == "rejected empty path" {
				if cmd == nil || !Evaluate(cmd, tt.workspaceDir).Approved {
					// Expected
					return
				}
				t.Fatalf("expected rejected empty path")
			}
			if err != nil {
				t.Fatalf("ParseCommand failed: %v", err)
			}
			res := Evaluate(cmd, tt.workspaceDir)
			if res.Approved != tt.wantApproved {
				t.Errorf("Evaluate() approved = %v, want %v (reason: %s)", res.Approved, tt.wantApproved, res.Reason)
			}
		})
	}
}
