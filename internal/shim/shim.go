package shim

import (
	"fmt"
	"os"
	"path/filepath"
)

func Install(home, lineageBinary string) error {
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create shim directory: %w", err)
	}

	for _, provider := range []string{"claude", "codex"} {
		if err := writeShim(filepath.Join(binDir, provider), lineageBinary, provider); err != nil {
			return err
		}
	}
	return nil
}

func writeShim(path, lineageBinary, provider string) error {
	content := fmt.Sprintf(`#!/bin/sh
exec %q run %s "$@"
`, lineageBinary, provider)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write %s shim: %w", provider, err)
	}
	return nil
}
