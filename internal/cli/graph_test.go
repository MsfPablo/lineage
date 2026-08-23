package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentic-lineage/lineage/internal/config"
	"github.com/agentic-lineage/lineage/internal/graph"
	"github.com/agentic-lineage/lineage/internal/snapshot"
)

func TestEnableRecordsGraphEntry(t *testing.T) {
	project, home := setUpEnabledProject(t)

	records, err := graph.Load(project)
	if err != nil {
		t.Fatalf("graph.Load() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("graph.Load() = %d records after one enable, want 1", len(records))
	}
	rec := records[0]
	if rec.Event != "enable" {
		t.Errorf("Event = %q, want %q", rec.Event, "enable")
	}
	if rec.Parent.Name != "agent-pack" || rec.Parent.Version != "0.1.0" {
		t.Errorf("Parent = %+v, want name=agent-pack version=0.1.0", rec.Parent)
	}
	if rec.Parent.Digest == "" {
		t.Error("Parent.Digest is empty, want a computed digest")
	}
	if rec.ID == "" {
		t.Error("ID is empty")
	}
	if rec.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	if rec.Parent.SnapshotID == "" {
		t.Fatal("Parent.SnapshotID is empty, want a snapshot created by enableRef")
	}
	if _, err := snapshot.LoadManifest(home, snapshot.ObjectID(rec.Parent.SnapshotID)); err != nil {
		t.Fatalf("snapshot.LoadManifest(%q) error = %v, want the snapshot enableRef created to be loadable", rec.Parent.SnapshotID, err)
	}
}

func TestEnableTwiceAppendsSecondRecord(t *testing.T) {
	project, _ := setUpEnabledProject(t)

	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"enable", "./agent-pack"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("second enable error = %v stderr=%s", err, stderr.String())
	}

	records, err := graph.Load(project)
	if err != nil {
		t.Fatalf("graph.Load() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("graph.Load() = %d records after enabling twice, want 2 (append-only log)", len(records))
	}
}

func TestGraphListHumanOutput(t *testing.T) {
	setUpEnabledProject(t)

	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"graph", "list"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("graph list error = %v stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "agent-pack@0.1.0") {
		t.Fatalf("graph list output = %s, want it to mention agent-pack@0.1.0", out)
	}
	if !strings.Contains(out, "enable") {
		t.Fatalf("graph list output = %s, want it to mention the enable event", out)
	}
}

func TestGraphListYAMLOutput(t *testing.T) {
	setUpEnabledProject(t)

	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"graph", "list", "--yaml"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("graph list --yaml error = %v stderr=%s", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "name: agent-pack") {
		t.Fatalf("graph list --yaml output = %s, want it to mention name: agent-pack", out)
	}
	if !strings.Contains(out, "event: enable") {
		t.Fatalf("graph list --yaml output = %s, want it to mention event: enable", out)
	}
	if !strings.Contains(out, "snapshot_id: sha256:") {
		t.Fatalf("graph list --yaml output = %s, want it to mention a snapshot_id", out)
	}
}

func TestGraphListWithNoRecordsSaysSo(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	project := filepath.Join(tmp, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveProjectConfig(config.ProjectConfigPath(project), config.DefaultProjectConfig()); err != nil {
		t.Fatal(err)
	}

	t.Setenv(config.HomeEnv, home)
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"graph", "list"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("graph list error = %v stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "no lineage graph records") {
		t.Fatalf("graph list output = %q, want a clear empty message", stdout.String())
	}
}
