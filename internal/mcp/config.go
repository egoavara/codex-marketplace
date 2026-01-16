package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	// MarkerStartPrefix is the prefix for start markers
	MarkerStartPrefix = "# [codex-market:start]"
	// MarkerEndPrefix is the prefix for end markers
	MarkerEndPrefix = "# [codex-market:end]"
)

// MCPServerConfig represents a single MCP server configuration from .mcp.json
type MCPServerConfig struct {
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// mcpJSONWrapped represents the wrapped format: { "mcpServers": { ... } }
type mcpJSONWrapped struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// ParseMCPJSON parses a .mcp.json file and returns the server configurations.
// Supports two formats:
// 1. Direct format: { "serverName": { "command": "...", ... } }
// 2. Wrapped format: { "mcpServers": { "serverName": { "command": "...", ... } } }
func ParseMCPJSON(data []byte) (map[string]MCPServerConfig, error) {
	// First try the wrapped format (mcpServers)
	var wrapped mcpJSONWrapped
	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.MCPServers) > 0 {
		return wrapped.MCPServers, nil
	}

	// Fall back to direct format
	var servers map[string]MCPServerConfig
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("failed to parse .mcp.json: %w", err)
	}

	// Filter out any non-server entries (like "mcpServers" key with wrong structure)
	result := make(map[string]MCPServerConfig)
	for name, config := range servers {
		if name != "mcpServers" && (config.Command != "" || config.URL != "") {
			result[name] = config
		}
	}

	return result, nil
}

// AddMCPServers adds MCP server configurations to config.toml with marker comments
// Returns any env var mismatches found (where key name differs from referenced variable)
func AddMCPServers(configPath string, pluginName string, marketplace string, servers map[string]MCPServerConfig) ([]EnvVarMismatch, error) {
	// Read existing config
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			content = []byte{}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Remove existing marker block for this plugin if present
	contentStr := RemoveMarkedBlock(string(content), pluginName)

	// Generate new TOML content for MCP servers
	tomlContent, mismatches := GenerateMCPServerTOML(pluginName, marketplace, servers)

	// Append new content
	newContent := strings.TrimRight(contentStr, "\n") + "\n" + tomlContent

	// Ensure directory exists
	if err := os.MkdirAll(strings.TrimSuffix(configPath, "/config.toml"), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	return mismatches, nil
}

// RemoveMCPServers removes MCP server configurations by plugin marker
func RemoveMCPServers(configPath string, pluginName string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to remove
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	newContent := RemoveMarkedBlock(string(content), pluginName)

	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// HasMCPServerMarker checks if a plugin's MCP servers are already installed
func HasMCPServerMarker(configPath string, pluginName string) bool {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	marker := fmt.Sprintf("%s plugin=%s", MarkerStartPrefix, pluginName)
	return strings.Contains(string(content), marker)
}

// GenerateMCPServerTOML generates TOML content for MCP servers with markers
// Returns the TOML content and any env var mismatches found
func GenerateMCPServerTOML(pluginName, marketplace string, servers map[string]MCPServerConfig) (string, []EnvVarMismatch) {
	var sb strings.Builder
	var allMismatches []EnvVarMismatch

	sb.WriteString(fmt.Sprintf("\n%s plugin=%s marketplace=%s\n", MarkerStartPrefix, pluginName, marketplace))

	// Sort server names for consistent output
	serverNames := make([]string, 0, len(servers))
	for name := range servers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	for _, name := range serverNames {
		config := servers[name]
		sb.WriteString(fmt.Sprintf("[mcp_servers.%q]\n", name))
		mismatches := writeMCPConfigToTOML(&sb, name, config)
		allMismatches = append(allMismatches, mismatches...)
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("%s plugin=%s\n", MarkerEndPrefix, pluginName))

	return sb.String(), allMismatches
}

// RemoveMarkedBlock removes a marked block from TOML content
func RemoveMarkedBlock(content string, pluginName string) string {
	// Build regex pattern to match start marker to end marker (inclusive)
	startPattern := regexp.QuoteMeta(fmt.Sprintf("%s plugin=%s", MarkerStartPrefix, pluginName))
	endPattern := regexp.QuoteMeta(fmt.Sprintf("%s plugin=%s", MarkerEndPrefix, pluginName))

	// Match from start marker line to end marker line (including newlines between)
	fullPattern := fmt.Sprintf(`(?m)\n?%s[^\n]*\n(?:.*\n)*?%s\n?`, startPattern, endPattern)

	re := regexp.MustCompile(fullPattern)
	return re.ReplaceAllString(content, "")
}

// EnvVarMismatch represents a case where env key differs from referenced variable
type EnvVarMismatch struct {
	Key     string // The key name (e.g., "TEST")
	VarName string // The referenced variable name (e.g., "TEST_NOT")
}

// writeMCPConfigToTOML writes an MCPServerConfig to TOML format
// Converts env values with ${VAR} pattern to env_vars array for Codex compatibility
// Returns a list of mismatches where key name differs from referenced variable name
func writeMCPConfigToTOML(sb *strings.Builder, name string, config MCPServerConfig) []EnvVarMismatch {
	if config.Type != "" {
		sb.WriteString(fmt.Sprintf("type = %q\n", config.Type))
	}
	if config.Command != "" {
		sb.WriteString(fmt.Sprintf("command = %q\n", config.Command))
	}
	if config.URL != "" {
		sb.WriteString(fmt.Sprintf("url = %q\n", config.URL))
	}
	// Note: cwd is not supported by Codex, so we skip it
	if len(config.Args) > 0 {
		sb.WriteString("args = [\n")
		for _, arg := range config.Args {
			sb.WriteString(fmt.Sprintf("  %q,\n", arg))
		}
		sb.WriteString("]\n")
	}

	var mismatches []EnvVarMismatch

	if len(config.Env) > 0 {
		// Separate env vars into two categories:
		// 1. env_vars: shell environment variable references (${VAR} pattern)
		// 2. env: literal values
		var envVars []string
		literalEnv := make(map[string]string)

		// Pattern to match ${VAR_NAME} or $VAR_NAME
		envRefPattern := regexp.MustCompile(`^\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?$`)

		for k, v := range config.Env {
			if matches := envRefPattern.FindStringSubmatch(v); len(matches) > 1 {
				// This is an environment variable reference
				// Use the key name for env_vars (Codex forwards shell env var with this name)
				envVars = append(envVars, k)

				// Track if key name differs from referenced variable name
				if k != matches[1] {
					mismatches = append(mismatches, EnvVarMismatch{
						Key:     k,
						VarName: matches[1],
					})
				}
			} else {
				// This is a literal value
				literalEnv[k] = v
			}
		}

		// Write env_vars array (for shell environment variable forwarding)
		if len(envVars) > 0 {
			sort.Strings(envVars)
			sb.WriteString("env_vars = [\n")
			for _, varName := range envVars {
				sb.WriteString(fmt.Sprintf("  %q,\n", varName))
			}
			sb.WriteString("]\n")
		}

		// Write literal env values
		if len(literalEnv) > 0 {
			envKeys := make([]string, 0, len(literalEnv))
			for k := range literalEnv {
				envKeys = append(envKeys, k)
			}
			sort.Strings(envKeys)

			sb.WriteString(fmt.Sprintf("\n[mcp_servers.%q.env]\n", name))
			for _, k := range envKeys {
				sb.WriteString(fmt.Sprintf("%s = %q\n", k, literalEnv[k]))
			}
		}
	}

	return mismatches
}

// GetExistingMCPServerNames returns the names of existing MCP servers from config.toml
// that are NOT managed by codex-market (no markers)
func GetExistingMCPServerNames(configPath string) ([]string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Find all [mcp_servers.XXX] or [mcp_servers."XXX"] sections
	// Pattern 1: unquoted names like [mcp_servers.atlassian]
	// Pattern 2: quoted names like [mcp_servers."my-server"]
	reUnquoted := regexp.MustCompile(`(?m)^\[mcp_servers\.([a-zA-Z_][a-zA-Z0-9_-]*)\]`)
	reQuoted := regexp.MustCompile(`(?m)^\[mcp_servers\."([^"]+)"\]`)

	var names []string

	matchesUnquoted := reUnquoted.FindAllStringSubmatch(string(content), -1)
	for _, match := range matchesUnquoted {
		if len(match) >= 2 {
			names = append(names, match[1])
		}
	}

	matchesQuoted := reQuoted.FindAllStringSubmatch(string(content), -1)
	for _, match := range matchesQuoted {
		if len(match) >= 2 {
			names = append(names, match[1])
		}
	}

	return names, nil
}

// CheckServerNameConflicts checks if any server names conflict with existing unmanaged servers
func CheckServerNameConflicts(configPath string, newServers map[string]MCPServerConfig) ([]string, error) {
	existing, err := GetExistingMCPServerNames(configPath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var conflicts []string
	for name := range newServers {
		for _, existingName := range existing {
			if name == existingName {
				// Check if it's managed by codex-market
				// If not (no marker contains this server), it's a user-managed server
				markerPattern := regexp.MustCompile(fmt.Sprintf(`%s plugin=.*\n(?:.*\n)*?\[mcp_servers\.%q\]`, regexp.QuoteMeta(MarkerStartPrefix), name))
				if !markerPattern.MatchString(string(content)) {
					conflicts = append(conflicts, name)
				}
			}
		}
	}

	return conflicts, nil
}
