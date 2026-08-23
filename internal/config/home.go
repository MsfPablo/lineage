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

// ObjectsDir is where internal/snapshot stores individual content-addressed
// file objects, kept separate from SnapshotsDir so a snapshot manifest and
// the objects it references live in distinct namespaces on disk.
func ObjectsDir(home string) string {
	return filepath.Join(home, "objects")
}

// SnapshotsDir is where internal/snapshot stores content-addressed snapshot
// manifests, addressed the same way as ObjectsDir's file objects but never
// commingled with them.
func SnapshotsDir(home string) string {
	return filepath.Join(home, "snapshots")
}
