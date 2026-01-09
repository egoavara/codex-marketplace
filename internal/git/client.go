package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Client is the interface for git operations
type Client interface {
	Clone(url, destPath string) error
	Pull(repoPath string) error
	Fetch(repoPath string) error
	GetCurrentCommit(repoPath string) (string, error)
	GetRemoteCommit(repoPath, branch string) (string, error)
	HasUpdates(repoPath string) (bool, error)
	IsGitRepository(path string) bool
}

// DefaultClient is the default git client implementation
type DefaultClient struct {
	Timeout time.Duration
}

// NewClient creates a new git client
func NewClient() *DefaultClient {
	return &DefaultClient{
		Timeout: 5 * time.Minute,
	}
}

// Clone clones a git repository to the specified path
func (c *DefaultClient) Clone(url, destPath string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", url, destPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if isAuthError(errMsg) {
			return &AuthError{URL: url, Message: errMsg}
		}
		return fmt.Errorf("git clone failed: %s", errMsg)
	}

	return nil
}

// Pull pulls the latest changes in a git repository
func (c *DefaultClient) Pull(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "pull", "--ff-only")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if isAuthError(errMsg) {
			return &AuthError{URL: repoPath, Message: errMsg}
		}
		return fmt.Errorf("git pull failed: %s", errMsg)
	}

	return nil
}

// GetCurrentCommit returns the current commit SHA
func (c *DefaultClient) GetCurrentCommit(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsGitRepository checks if the given path is a git repository
func (c *DefaultClient) IsGitRepository(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// Fetch fetches changes from remote without merging
func (c *DefaultClient) Fetch(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "fetch", "--quiet")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if isAuthError(errMsg) {
			return &AuthError{URL: repoPath, Message: errMsg}
		}
		return fmt.Errorf("git fetch failed: %s", errMsg)
	}

	return nil
}

// GetRemoteCommit returns the latest commit SHA of a remote branch
func (c *DefaultClient) GetRemoteCommit(repoPath, branch string) (string, error) {
	if branch == "" {
		branch = "origin/HEAD"
	} else {
		branch = "origin/" + branch
	}

	cmd := exec.Command("git", "-C", repoPath, "rev-parse", branch)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get remote commit: %s", stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// HasUpdates checks if the local repository is behind the remote
func (c *DefaultClient) HasUpdates(repoPath string) (bool, error) {
	// Fetch first to get latest remote state
	if err := c.Fetch(repoPath); err != nil {
		return false, err
	}

	// Get current branch name
	branchCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	var branchOut bytes.Buffer
	branchCmd.Stdout = &branchOut
	if err := branchCmd.Run(); err != nil {
		return false, fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(branchOut.String())

	// Get local commit
	localCommit, err := c.GetCurrentCommit(repoPath)
	if err != nil {
		return false, err
	}

	// Get remote commit
	remoteCommit, err := c.GetRemoteCommit(repoPath, branch)
	if err != nil {
		return false, err
	}

	return localCommit != remoteCommit, nil
}

// AuthError represents a git authentication error
type AuthError struct {
	URL     string
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication failed for '%s': %s", e.URL, e.Message)
}

// isAuthError checks if the error message indicates an authentication failure
func isAuthError(msg string) bool {
	authPatterns := []string{
		"Authentication failed",
		"Permission denied",
		"could not read Username",
		"fatal: repository",
		"not found",
		"403",
		"401",
	}

	for _, pattern := range authPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}
