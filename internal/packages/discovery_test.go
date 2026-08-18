package packages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestRoundTripAndDiscovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), "agent-pack")
	if err := InitPackage(root, "agent-pack"); err != nil {
		t.Fatalf("InitPackage() error = %v", err)
	}
	mustWrite(t, filepath.Join(root, "skills", "review", "SKILL.md"), "# Review")
	mustWrite(t, filepath.Join(root, "skills", "ignored", "README.md"), "# Not a skill")
	mustWrite(t, filepath.Join(root, "workflows", "ship", "WORKFLOW.md"), "# Ship")
	mustWrite(t, filepath.Join(root, "agents", "reviewer.md"), "# Reviewer")
	mustWrite(t, filepath.Join(root, "policies", "safe.md"), "# Safe")

	pkg, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if pkg.Manifest.Name != "agent-pack" {
		t.Fatalf("Manifest.Name = %q", pkg.Manifest.Name)
	}
	assertEqualSlices(t, pkg.Skills, []string{"review"})
	assertEqualSlices(t, pkg.Workflows, []string{"ship"})
	assertEqualSlices(t, pkg.Agents, []string{"reviewer.md"})
	assertEqualSlices(t, pkg.Policies, []string{"safe.md"})
}

func TestResolveReferenceOrder(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	project := filepath.Join(tmp, "project")
	userPkg := filepath.Join(home, "user", "packages", "shared")
	workspacePkg := filepath.Join(home, "workspaces", "team", "packages", "shared")
	projectPkg := filepath.Join(project, ".lineage", "packages", "shared")

	for _, dir := range []string{userPkg, workspacePkg, projectPkg} {
		if err := InitPackage(dir, filepath.Base(dir)); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ResolveReference(home, "team", project, "shared")
	if err != nil {
		t.Fatalf("ResolveReference() error = %v", err)
	}
	if got != projectPkg {
		t.Fatalf("ResolveReference() = %q, want project package %q", got, projectPkg)
	}
}

func TestResolveReferenceMissing(t *testing.T) {
	_, err := ResolveReference(t.TempDir(), "", t.TempDir(), "missing")
	if err == nil {
		t.Fatal("ResolveReference() error = nil, want error")
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertEqualSlices(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	}
}
