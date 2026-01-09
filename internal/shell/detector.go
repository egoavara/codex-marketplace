package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ShellType represents the type of shell
type ShellType string

const (
	ShellZsh     ShellType = "zsh"
	ShellBash    ShellType = "bash"
	ShellUnknown ShellType = "unknown"
)

// ErrUnsupportedShell is returned when the shell is not zsh or bash
var ErrUnsupportedShell = errors.New("unsupported shell")

// DetectShell detects the current shell type from SHELL environment variable
// Returns ErrUnsupportedShell if the shell is not zsh or bash
func DetectShell() (ShellType, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return ShellUnknown, ErrUnsupportedShell
	}

	shellName := filepath.Base(shell)

	switch {
	case strings.Contains(shellName, "zsh"):
		return ShellZsh, nil
	case strings.Contains(shellName, "bash"):
		return ShellBash, nil
	default:
		return ShellUnknown, ErrUnsupportedShell
	}
}

// GetShellConfigPath returns the path to the shell configuration file
// Only supports zsh and bash
func GetShellConfigPath(shellType ShellType) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch shellType {
	case ShellZsh:
		return filepath.Join(home, ".zshrc"), nil
	case ShellBash:
		// Check .bash_profile first (macOS), then .bashrc (Linux)
		bashProfile := filepath.Join(home, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			return bashProfile, nil
		}
		return filepath.Join(home, ".bashrc"), nil
	default:
		return "", ErrUnsupportedShell
	}
}
