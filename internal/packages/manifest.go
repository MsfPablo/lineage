package packages

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ManifestFileName = "lineage.yaml"

type Manifest struct {
	Name        string      `yaml:"name"`
	Version     string      `yaml:"version"`
	Description string      `yaml:"description"`
	Exports     Exports     `yaml:"exports"`
	Requires    Requires    `yaml:"requires"`
	Entrypoints Entrypoints `yaml:"entrypoints"`
}

type Exports struct {
	Agents    []string `yaml:"agents"`
	Workflows []string `yaml:"workflows"`
}

type Requires struct {
	Skills []string `yaml:"skills"`
}

type Entrypoints struct {
	Claude string `yaml:"claude"`
	Codex  string `yaml:"codex"`
}

func DefaultManifest(name string) Manifest {
	return Manifest{
		Name:        name,
		Version:     "0.1.0",
		Description: "A shareable Lineage agent package.",
		Exports: Exports{
			Agents:    []string{},
			Workflows: []string{},
		},
		Requires: Requires{
			Skills: []string{},
		},
		Entrypoints: Entrypoints{},
	}
}

func LoadManifest(dir string) (Manifest, error) {
	path := filepath.Join(dir, ManifestFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	if manifest.Name == "" {
		return Manifest{}, fmt.Errorf("manifest %s missing name", path)
	}
	if manifest.Version == "" {
		manifest.Version = "0.1.0"
	}
	return manifest, nil
}

func SaveManifest(dir string, manifest Manifest) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create package directory: %w", err)
	}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}
