package shell

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	// AliasLine is the alias command to add to shell config
	AliasLine = `alias codex="codex-market run"`
	// AliasMarker is a comment marker to identify our alias
	AliasMarker = "# codex-market auto-updater"
)

// HasCodexAlias checks if the codex alias is already set in the config file
func HasCodexAlias(configPath string) (bool, error) {
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `alias codex=`) && strings.Contains(line, "codex-market run") {
			return true, nil
		}
		if strings.Contains(line, AliasMarker) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to read config file: %w", err)
	}

	return false, nil
}

// AddCodexAlias adds the codex alias to the shell config file
func AddCodexAlias(configPath string) error {
	// Open file in append mode, create if not exists
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Add newline, marker, and alias
	content := fmt.Sprintf("\n%s\n%s\n", AliasMarker, AliasLine)

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write alias: %w", err)
	}

	return nil
}

// RemoveCodexAlias removes the codex alias from the shell config file
func RemoveCodexAlias(configPath string) error {
	// Read the entire file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	skipNext := false

	for _, line := range lines {
		// Skip marker and the alias line that follows
		if strings.Contains(line, AliasMarker) {
			skipNext = true
			continue
		}
		if skipNext && strings.Contains(line, `alias codex=`) {
			skipNext = false
			continue
		}
		skipNext = false
		newLines = append(newLines, line)
	}

	// Write back
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
