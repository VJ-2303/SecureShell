package policy

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"safeshell/pkg/models"
)

// AllowedCommands is the set of commands allowed in the MVP.
var AllowedCommands = map[string]bool{
	"mkdir": true,
	"touch": true,
	"rm":    true,
}

// ParseCommand parses raw command arguments into a ParsedCommand struct.
func ParseCommand(args []string) (*models.ParsedCommand, error) {
	if len(args) == 0 {
		return nil, errors.New("empty command")
	}

	raw := strings.Join(args, " ")
	cmdName := args[0]
	cmdArgs := args[1:]

	parsed := &models.ParsedCommand{
		Raw:  raw,
		Name: cmdName,
		Args: cmdArgs,
	}

	// Extract primary target path for supported commands
	for _, arg := range cmdArgs {
		if !strings.HasPrefix(arg, "-") {
			parsed.Path = arg
			break
		}
	}

	return parsed, nil
}

// Evaluate evaluates a parsed command against SafeShell security policy rules.
func Evaluate(cmd *models.ParsedCommand, workspaceDir string) *models.PolicyResult {
	if cmd == nil {
		return &models.PolicyResult{Approved: false, Reason: "command is nil"}
	}

	// 1. Check command allowlist
	if !AllowedCommands[cmd.Name] {
		return &models.PolicyResult{
			Approved: false,
			Reason:   fmt.Sprintf("command '%s' is not allowed in safe policy", cmd.Name),
		}
	}

	// 2. Disallow recursive delete flags for rm
	if cmd.Name == "rm" {
		for _, arg := range cmd.Args {
			if arg == "-r" || arg == "-rf" || arg == "-fr" || arg == "-R" || strings.Contains(arg, "r") && strings.HasPrefix(arg, "-") || arg == "--recursive" {
				return &models.PolicyResult{
					Approved: false,
					Reason:   "recursive delete is strictly prohibited",
				}
			}
		}
	}

	// 3. Path must not be empty
	if cmd.Path == "" {
		return &models.PolicyResult{
			Approved: false,
			Reason:   "target path is empty",
		}
	}

	// 4. Disallow absolute paths
	if filepath.IsAbs(cmd.Path) || strings.HasPrefix(cmd.Path, "/") || strings.HasPrefix(cmd.Path, "\\") {
		return &models.PolicyResult{
			Approved: false,
			Reason:   "absolute paths are prohibited",
		}
	}

	// 5. Disallow path traversal (contains "..")
	cleaned := filepath.Clean(cmd.Path)
	parts := strings.Split(filepath.ToSlash(cleaned), "/")
	for _, part := range parts {
		if part == ".." {
			return &models.PolicyResult{
				Approved: false,
				Reason:   "path traversal ('..') is prohibited",
			}
		}
	}

	// Also check raw path for ".." in case of tricks
	if strings.Contains(cmd.Path, "..") {
		return &models.PolicyResult{
			Approved: false,
			Reason:   "path traversal ('..') is prohibited",
		}
	}

	// 6. Ensure the target path resolves inside workspaceDir
	if workspaceDir != "" {
		absWorkspace, err := filepath.Abs(workspaceDir)
		if err == nil {
			targetAbs := filepath.Join(absWorkspace, cleaned)
			rel, err := filepath.Rel(absWorkspace, targetAbs)
			if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
				return &models.PolicyResult{
					Approved: false,
					Reason:   "path escapes workspace boundary",
				}
			}
		}
	}

	return &models.PolicyResult{
		Approved: true,
	}
}
