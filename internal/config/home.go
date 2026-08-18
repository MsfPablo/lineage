package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const HomeEnv = "LINEAGE_HOME"

func HomeDir() (string, error) {
	if value := os.Getenv(HomeEnv); value != "" {
		return filepath.Abs(value)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(home, ".lineage"), nil
}

func UserPackagesDir(home string) string {
	return filepath.Join(home, "user", "packages")
}

func WorkspacesDir(home string) string {
	return filepath.Join(home, "workspaces")
}

func WorkspacePackagesDir(home, name string) string {
	return filepath.Join(WorkspacesDir(home), name, "packages")
}

func ShimsDir(home string) string {
	return filepath.Join(home, "bin")
}
