package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestTopLevelHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"--help"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("--help error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Package authoring:") {
		t.Fatalf("stdout = %q, want grouped command sections", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for --help", stderr.String())
	}
}

func TestVersionFlag(t *testing.T) {
	for _, flag := range []string{"version", "-v", "--version"} {
		var stdout, stderr bytes.Buffer
		if err := Execute(nil, []string{flag}, nil, &stdout, &stderr); err != nil {
			t.Fatalf("%s error = %v", flag, err)
		}
		if !strings.HasPrefix(stdout.String(), "lineage ") {
			t.Fatalf("%s stdout = %q, want it to start with \"lineage \"", flag, stdout.String())
		}
	}
}

func TestPackageHelpShowsSubcommandList(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute(nil, []string{"package", "--help"}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("package --help error = %v", err)
	}
	if !strings.Contains(stdout.String(), "publish <path>") {
		t.Fatalf("stdout = %q, want the subcommand list", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for package --help", stderr.String())
	}
}

// A bare "-h" as the argument to a subcommand must show that subcommand's
// own usage instead of being treated as a path/name argument - previously
// `lineage package export -h` would try to export a package literally
// named "-h" and fail with a confusing filesystem error.
func TestPackageSubcommandHelpDoesNotTreatFlagAsArgument(t *testing.T) {
	cases := [][]string{
		{"package", "init", "-h"},
		{"package", "validate", "--help"},
		{"package", "export", "-h"},
		{"package", "import", "--help"},
		{"package", "publish", "-h"},
		{"package", "pull", "--help"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		if err := Execute(nil, args, nil, &stdout, &stderr); err != nil {
			t.Fatalf("%v error = %v stderr=%s", args, err, stderr.String())
		}
		if !strings.HasPrefix(stdout.String(), "usage: lineage package "+args[1]) {
			t.Fatalf("%v stdout = %q, want it to start with that subcommand's usage line", args, stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("%v stderr = %q, want empty", args, stderr.String())
		}
	}
}
