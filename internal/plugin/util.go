package plugin

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
)

// GenerateRandomSuffix generates a random alphanumeric suffix of the specified length.
// Uses crypto/rand for secure random generation.
func GenerateRandomSuffix(length int) (string, error) {
	bytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// ResolveUniqueSkillPath returns a unique skill destination path.
// If the base path already exists, appends -{8-char-random} suffix.
// Returns (resolvedPath, actualFolderName, error)
func ResolveUniqueSkillPath(skillsDir, skillName string) (string, string, error) {
	basePath := filepath.Join(skillsDir, skillName)

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return basePath, skillName, nil
	}

	suffix, err := GenerateRandomSuffix(8)
	if err != nil {
		return "", "", err
	}

	uniqueName := skillName + "-" + suffix
	uniquePath := filepath.Join(skillsDir, uniqueName)

	return uniquePath, uniqueName, nil
}
