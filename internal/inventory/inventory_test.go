package inventory

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// buildMixedWorkspace lays out a fixture covering the realistic messy-
// package shapes #102 targets: root CLAUDE.md/AGENTS.md, a skill with a
// colocated helper script it names, a workflow that names both that skill's
// script and a standalone script (the fan-in case — one script cited by two
// different instruction files), a reference file, a setup note, an
// unreferenced orphan script, and ignored-by-default directories.
func buildMixedWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "# Project instructions\n\nGeneral guidance for the assistant.\n")
	mustWrite(t, filepath.Join(root, "AGENTS.md"), "# Codex instructions\n\nSame general guidance, mirrored for Codex.\n")
	mustWrite(t, filepath.Join(root, "README.md"), "# Setup\n\nRun the install steps below.\n")

	mustWrite(t, filepath.Join(root, "skills", "foo", "SKILL.md"),
		"# Foo skill\n\nInvoke the helper by running `run.sh` after setup.\n")
	mustWrite(t, filepath.Join(root, "skills", "foo", "run.sh"), "#!/bin/sh\necho foo\n")

	mustWrite(t, filepath.Join(root, "workflows", "deploy", "WORKFLOW.md"),
		"# Deploy workflow\n\n1. Run skills/foo/run.sh to prepare.\n2. Run scripts/deploy.sh to ship.\n")
	mustWrite(t, filepath.Join(root, "scripts", "deploy.sh"), "#!/bin/sh\necho deploy\n")
	mustWrite(t, filepath.Join(root, "scripts", "orphan.sh"), "#!/bin/sh\necho orphan\n")

	mustWrite(t, filepath.Join(root, "references", "notes.md"), "# Notes\n\nBackground reference material.\n")

	mustWrite(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/main\n")
	mustWrite(t, filepath.Join(root, "node_modules", "pkg", "index.js"), "module.exports = {};\n")

	return root
}

func TestDiscoverDeterministic(t *testing.T) {
	root := buildMixedWorkspace(t)

	first, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	second, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() second run error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("Discover() not deterministic:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

func TestDiscoverSkipsIgnoredDirs(t *testing.T) {
	root := buildMixedWorkspace(t)

	inv, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	for _, e := range inv.Entries {
		if hasIgnoredSegment(e.Path) {
			t.Fatalf("entry %q should have been skipped as ignored", e.Path)
		}
	}
}

func hasIgnoredSegment(rel string) bool {
	for _, seg := range strings.Split(rel, "/") {
		if isIgnoredDir(seg) {
			return true
		}
	}
	return false
}

func TestDiscoverClassifiesRealisticShapes(t *testing.T) {
	root := buildMixedWorkspace(t)

	inv, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	byPath := entryMap(inv)
	cases := []struct {
		path string
		kind ArtifactKind
	}{
		{"CLAUDE.md", KindInstruction},
		{"AGENTS.md", KindInstruction},
		{"README.md", KindSetupMaterial},
		{"skills/foo/SKILL.md", KindInstruction},
		{"skills/foo/run.sh", KindExecutableHelper},
		{"workflows/deploy/WORKFLOW.md", KindInstruction},
		{"scripts/deploy.sh", KindExecutableHelper},
		{"scripts/orphan.sh", KindExecutableHelper},
		{"references/notes.md", KindReference},
	}
	for _, c := range cases {
		e, ok := byPath[c.path]
		if !ok {
			t.Fatalf("missing expected entry %q", c.path)
		}
		if e.Kind != c.kind {
			t.Errorf("%s: Kind = %q, want %q (reason: %s)", c.path, e.Kind, c.kind, e.Reason)
		}
		if e.Digest == "" {
			t.Errorf("%s: Digest is empty", c.path)
		}
		if e.Reason == "" {
			t.Errorf("%s: Reason is empty", c.path)
		}
	}
}

func TestDiscoverCitationSingleReference(t *testing.T) {
	root := buildMixedWorkspace(t)

	inv, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	byPath := entryMap(inv)

	wf := byPath["workflows/deploy/WORKFLOW.md"]
	deploy := byPath["scripts/deploy.sh"]

	if len(deploy.ReferencedBy) != 1 {
		t.Fatalf("scripts/deploy.sh ReferencedBy = %+v, want exactly 1 citation", deploy.ReferencedBy)
	}
	want := Citation{FromPath: "workflows/deploy/WORKFLOW.md", Line: 4, Snippet: "2. Run scripts/deploy.sh to ship."}
	if deploy.ReferencedBy[0] != want {
		t.Errorf("scripts/deploy.sh ReferencedBy[0] = %+v, want %+v", deploy.ReferencedBy[0], want)
	}
	if !containsCitation(wf.Mentions, want) {
		t.Errorf("workflows/deploy/WORKFLOW.md Mentions = %+v, want to contain %+v", wf.Mentions, want)
	}

	// Reason must never be overwritten by the citation pass.
	if deploy.Reason == "" || deploy.Reason == want.Snippet {
		t.Errorf("scripts/deploy.sh Reason was corrupted by citation pass: %q", deploy.Reason)
	}
}

func TestDiscoverCitationFanIn(t *testing.T) {
	root := buildMixedWorkspace(t)

	inv, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	byPath := entryMap(inv)

	runSh := byPath["skills/foo/run.sh"]
	if len(runSh.ReferencedBy) != 2 {
		t.Fatalf("skills/foo/run.sh ReferencedBy = %+v, want exactly 2 citations (SKILL.md and WORKFLOW.md)", runSh.ReferencedBy)
	}

	froms := make([]string, len(runSh.ReferencedBy))
	for i, c := range runSh.ReferencedBy {
		froms[i] = c.FromPath
	}
	sort.Strings(froms)
	want := []string{"skills/foo/SKILL.md", "workflows/deploy/WORKFLOW.md"}
	if !reflect.DeepEqual(froms, want) {
		t.Errorf("skills/foo/run.sh ReferencedBy froms = %v, want %v", froms, want)
	}
}

func TestDiscoverOrphanScriptUnreferenced(t *testing.T) {
	root := buildMixedWorkspace(t)

	inv, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	byPath := entryMap(inv)

	orphan := byPath["scripts/orphan.sh"]
	if len(orphan.ReferencedBy) != 0 {
		t.Fatalf("scripts/orphan.sh ReferencedBy = %+v, want empty (truly unreferenced)", orphan.ReferencedBy)
	}
	if orphan.Reason == "" {
		t.Error("scripts/orphan.sh Reason must still be set even though unreferenced")
	}
}

// TestDiscoverAmbiguousBasenameOnlyMatchesFullPath reproduces a real bug
// found by running Discover against an actual messy repo: a workflow
// referencing a dated output file by its bare name (e.g. "report.json")
// must not fan out to every unrelated file sharing that basename elsewhere
// in the tree. Only an unambiguous full-path mention should cite.
func TestDiscoverAmbiguousBasenameOnlyMatchesFullPath(t *testing.T) {
	root := t.TempDir()

	mustWrite(t, filepath.Join(root, "workspace", "2024-01-01", "report.json"), "{}\n")
	mustWrite(t, filepath.Join(root, "workspace", "2024-01-02", "report.json"), "{}\n")
	mustWrite(t, filepath.Join(root, "workspace", "2024-01-03", "report.json"), "{}\n")
	mustWrite(t, filepath.Join(root, "CLAUDE.md"),
		"# Debugging\n\nIf `report.json` is missing, check the fetch step.\n\n"+
			"Full path: workspace/2024-01-02/report.json is the one that matters here.\n")

	inv, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	byPath := entryMap(inv)

	for _, day := range []string{"2024-01-01", "2024-01-03"} {
		p := "workspace/" + day + "/report.json"
		e := byPath[p]
		if len(e.ReferencedBy) != 0 {
			t.Errorf("%s ReferencedBy = %+v, want empty (bare-name match is ambiguous, must not fan out)", p, e.ReferencedBy)
		}
	}

	cited := byPath["workspace/2024-01-02/report.json"]
	if len(cited.ReferencedBy) != 1 {
		t.Fatalf("workspace/2024-01-02/report.json ReferencedBy = %+v, want exactly 1 (unambiguous full-path mention)", cited.ReferencedBy)
	}
	if cited.ReferencedBy[0].Line != 5 {
		t.Errorf("ReferencedBy[0].Line = %d, want 5", cited.ReferencedBy[0].Line)
	}
}

// TestDiscoverRecognizesProviderPrefixedAgentFiles reproduces a real gap
// found by running Discover against an actual public Claude Code project
// (ChrisWiles/claude-code-showcase): .claude/agents/code-reviewer.md is
// genuine instruction content (YAML frontmatter + prose, structurally
// identical to a SKILL.md) but was classified unknown, because the
// directory-based instruction rule only recognized bare top-level
// skills/workflows/, not the .claude/-prefixed convention real Claude Code
// repos actually use, and had no notion of an "agents" directory at all.
func TestDiscoverRecognizesProviderPrefixedAgentFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".claude", "agents", "code-reviewer.md"),
		"---\nname: code-reviewer\n---\n\nSenior code reviewer. Run `git diff` and review changes.\n")
	// Exercises the ".agents" provider-root branch symmetrically — not
	// itself an observed real-world path, just coverage for the second
	// known prefix alongside the ".claude" one found in the wild above.
	mustWrite(t, filepath.Join(root, ".agents", "agents", "deployer.md"),
		"Deploys the app after tests pass.\n")
	// A flat file directly in "agents/" at the bare top level (no provider
	// prefix) must also match, mirroring how Lineage's own agents/ content
	// dir works (flat files, per internal/packages).
	mustWrite(t, filepath.Join(root, "agents", "reviewer.md"), "Reviews things.\n")
	// One level deeper than a provider root's agents/ dir is auxiliary
	// material, not the agent's own file, and must NOT match.
	mustWrite(t, filepath.Join(root, ".claude", "agents", "notes", "scratch.md"), "unrelated notes\n")

	inv, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	byPath := entryMap(inv)

	for _, p := range []string{
		".claude/agents/code-reviewer.md",
		".agents/agents/deployer.md",
		"agents/reviewer.md",
	} {
		if e, ok := byPath[p]; !ok || e.Kind != KindInstruction {
			t.Errorf("%s Kind = %v, want %q", p, e.Kind, KindInstruction)
		}
	}
	if e := byPath[".claude/agents/notes/scratch.md"]; e.Kind == KindInstruction {
		t.Errorf(".claude/agents/notes/scratch.md Kind = %q, want anything but %q (one level too deep)", e.Kind, KindInstruction)
	}
}

func TestDiscoverUnknownIsHonestNotGuessed(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "mystery.xyz"), "no rule matches this\n")

	inv, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	byPath := entryMap(inv)
	e, ok := byPath["mystery.xyz"]
	if !ok {
		t.Fatal("missing mystery.xyz entry")
	}
	if e.Kind != KindUnknown {
		t.Errorf("Kind = %q, want %q", e.Kind, KindUnknown)
	}
	if e.Reason != "no classification rule matched" {
		t.Errorf("Reason = %q, want honest no-match reason", e.Reason)
	}
}

func TestDiscoverRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "real.txt"), "content\n")

	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("outside content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(root, "link.txt")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	if _, err := Discover(root); err == nil {
		t.Fatal("Discover() error = nil, want error for symlinked content")
	}
}

func entryMap(inv Inventory) map[string]Entry {
	m := make(map[string]Entry, len(inv.Entries))
	for _, e := range inv.Entries {
		m[e.Path] = e
	}
	return m
}

func containsCitation(cs []Citation, want Citation) bool {
	for _, c := range cs {
		if c == want {
			return true
		}
	}
	return false
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
