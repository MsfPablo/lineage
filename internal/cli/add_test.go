package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentic-lineage/lineage/internal/config"
	"github.com/agentic-lineage/lineage/internal/packages"
)

func TestAddUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"add"}, nil, &stdout, &stderr); err == nil {
		t.Fatal("add with no ref: error = nil, want error")
	} else if !strings.Contains(stderr.String(), "usage: lineage add <package-ref>") {
		t.Fatalf("stderr = %q, want usage message", stderr.String())
	}
}

func TestAddHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"add", "-h"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("add -h error = %v", err)
	}
	if !strings.HasPrefix(stdout.String(), "usage: lineage add <package-ref>") {
		t.Fatalf("stdout = %q, want the add usage line", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

// addTestServer serves the same contract as the real registry API, backed
// by an in-memory package built from srcDir.
func addTestServer(t *testing.T, ref, srcDir string) *httptest.Server {
	t.Helper()
	report, err := packages.Validate(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	if err := packages.Export(srcDir, &archive); err != nil {
		t.Fatal(err)
	}
	archiveBytes := archive.Bytes()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/packages/"+ref, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": report.Manifest.Name, "version": report.Manifest.Version, "digest": report.Digest,
			"downloadPath": "/api/packages/" + ref + "/download",
		})
	})
	mux.HandleFunc("/api/packages/"+ref+"/download", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archiveBytes)
	})
	return httptest.NewServer(mux)
}

func TestAddFetchesInspectsAndEnablesWithYes(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	project := filepath.Join(tmp, "project")
	srcDir := filepath.Join(tmp, "commit-helper")
	if err := packages.InitPackage(srcDir, "commit-helper"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "skills", "commit-messages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "skills", "commit-messages", "SKILL.md"), []byte("# Commit messages"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	ref := "commit-helper@0.1.0"
	srv := addTestServer(t, ref, srcDir)
	defer srv.Close()

	oldHome := os.Getenv(config.HomeEnv)
	t.Setenv(config.HomeEnv, home)
	defer t.Setenv(config.HomeEnv, oldHome)
	t.Setenv("LINEAGE_REGISTRY_URL", srv.URL)

	oldWd, _ := os.Getwd()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"add", ref, "--yes"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("add error = %v stderr=%s", err, stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{"fetched commit-helper@0.1.0", "skills: commit-messages", "enabled package commit-helper", "Ready. Run `lineage run"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout = %q, want it to contain %q", out, want)
		}
	}

	found, err := config.FindProjectConfig(project)
	if err != nil {
		t.Fatalf("expected a project config after add --yes: %v", err)
	}
	if !contains(found.Config.EnabledPackages, "commit-helper") {
		t.Errorf("enabled packages = %v, want commit-helper", found.Config.EnabledPackages)
	}
}

func TestAddDeclinedDoesNotEnable(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	project := filepath.Join(tmp, "project")
	srcDir := filepath.Join(tmp, "resume-like")
	if err := packages.InitPackage(srcDir, "resume-like"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	ref := "resume-like@0.1.0"
	srv := addTestServer(t, ref, srcDir)
	defer srv.Close()

	oldHome := os.Getenv(config.HomeEnv)
	t.Setenv(config.HomeEnv, home)
	defer t.Setenv(config.HomeEnv, oldHome)
	t.Setenv("LINEAGE_REGISTRY_URL", srv.URL)

	oldWd, _ := os.Getwd()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("n\n")
	if err := Execute(nil, []string{"add", ref}, stdin, &stdout, &stderr); err != nil {
		t.Fatalf("add error = %v stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "not enabled") {
		t.Fatalf("stdout = %q, want a message confirming it was not enabled", stdout.String())
	}

	if _, err := config.FindProjectConfig(project); err == nil {
		t.Fatal("expected no project config to exist after declining - add must not enable without permission")
	}

	// The package is still fetched locally even though it wasn't enabled -
	// declining should stop at enable, not undo the fetch.
	if _, err := os.Stat(filepath.Join(config.UserPackagesDir(home), "resume-like", packages.ManifestFileName)); err != nil {
		t.Errorf("expected the package to still be pulled locally after declining: %v", err)
	}
}
