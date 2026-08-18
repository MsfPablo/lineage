package provider

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lineage-dev/lineage/internal/config"
)

func TestIsShimPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path prefix semantics differ on Windows")
	}

	home := t.TempDir()
	shimPath := filepath.Join(home, "bin", "claude")
	if !IsShimPath(shimPath, home) {
		t.Fatalf("IsShimPath(%q) = false, want true", shimPath)
	}
	realPath := filepath.Join(home, "real", "claude")
	if IsShimPath(realPath, home) {
		t.Fatalf("IsShimPath(%q) = true, want false", realPath)
	}
}

func TestResolveWithConfiguredBinary(t *testing.T) {
	project := config.ProjectConfig{
		Providers: map[string]config.Provider{
			"codex": {Binary: "/bin/echo"},
		},
	}
	plan, err := Resolve("codex", t.TempDir(), project, []string{"hello"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if plan.Binary != "/bin/echo" {
		t.Fatalf("Binary = %q", plan.Binary)
	}
	if len(plan.Args) != 1 || plan.Args[0] != "hello" {
		t.Fatalf("Args = %#v", plan.Args)
	}
}
