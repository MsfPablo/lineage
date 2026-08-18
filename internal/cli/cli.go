package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lineage-dev/lineage/internal/config"
	"github.com/lineage-dev/lineage/internal/packages"
	"github.com/lineage-dev/lineage/internal/provider"
	"github.com/lineage-dev/lineage/internal/runtime"
	"github.com/lineage-dev/lineage/internal/shim"
)

func Execute(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	home, err := config.HomeDir()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], home, stdout, stderr)
	case "package":
		return runPackage(args[1:], stdout, stderr)
	case "enable":
		return runEnable(args[1:], home, stdout, stderr)
	case "run":
		return runProvider(ctx, args[1:], home, stdout, stderr)
	case "install-shims":
		return runInstallShims(home, stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		err := fmt.Errorf("unknown command %q", args[0])
		fmt.Fprintln(stderr, err)
		printUsage(stderr)
		return err
	}
}

func runInit(args []string, home string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		err := fmt.Errorf("usage: lineage init user | lineage init workspace <name>")
		fmt.Fprintln(stderr, err)
		return err
	}

	switch args[0] {
	case "user":
		if err := os.MkdirAll(config.UserPackagesDir(home), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return err
		}
		fmt.Fprintf(stdout, "initialized user packages at %s\n", config.UserPackagesDir(home))
		return nil
	case "workspace":
		if len(args) != 2 {
			err := fmt.Errorf("usage: lineage init workspace <name>")
			fmt.Fprintln(stderr, err)
			return err
		}
		dir := config.WorkspacePackagesDir(home, args[1])
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return err
		}
		fmt.Fprintf(stdout, "initialized workspace %s at %s\n", args[1], dir)
		return nil
	default:
		err := fmt.Errorf("unknown init target %q", args[0])
		fmt.Fprintln(stderr, err)
		return err
	}
}

func runPackage(args []string, stdout, stderr io.Writer) error {
	if len(args) != 2 || args[0] != "init" {
		err := fmt.Errorf("usage: lineage package init <name>")
		fmt.Fprintln(stderr, err)
		return err
	}

	name := args[1]
	dir := filepath.Clean(name)
	if err := packages.InitPackage(dir, filepath.Base(name)); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintf(stdout, "initialized package %s\n", dir)
	return nil
}

func runEnable(args []string, home string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		err := fmt.Errorf("usage: lineage enable <package-path-or-id>")
		fmt.Fprintln(stderr, err)
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	configPath := config.ProjectConfigPath(cwd)
	cfg := config.DefaultProjectConfig()
	if loaded, err := config.LoadProjectConfig(configPath); err == nil {
		cfg = loaded
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(stderr, err)
		return err
	}

	ref := args[0]
	if _, err := packages.ResolveReference(home, cfg.Workspace, cwd, ref); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if !contains(cfg.EnabledPackages, ref) {
		cfg.EnabledPackages = append(cfg.EnabledPackages, ref)
	}
	if err := config.SaveProjectConfig(configPath, cfg); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintf(stdout, "enabled package %s in %s\n", ref, configPath)
	return nil
}

func runProvider(ctx context.Context, args []string, home string, stdout, stderr io.Writer) error {
	_ = ctx
	if len(args) == 0 {
		err := fmt.Errorf("usage: lineage run <claude|codex> [--dry-run] [-- provider args...]")
		fmt.Fprintln(stderr, err)
		return err
	}

	providerName := args[0]
	dryRun := false
	providerArgs := []string{}
	for _, arg := range args[1:] {
		if arg == "--dry-run" {
			dryRun = true
			continue
		}
		if arg == "--" {
			continue
		}
		providerArgs = append(providerArgs, arg)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	plan, err := runtime.BuildPlan(providerName, cwd, home, providerArgs)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}

	if dryRun {
		fmt.Fprint(stdout, plan.DryRunString())
		return nil
	}
	if err := provider.Launch(plan.ProviderPlan); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	return nil
}

func runInstallShims(home string, stdout, stderr io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	if err := shim.Install(home, exe); err != nil {
		fmt.Fprintln(stderr, err)
		return err
	}
	fmt.Fprintf(stdout, "installed shims in %s\n", config.ShimsDir(home))
	fmt.Fprintf(stdout, "add this directory before existing agent binaries in PATH to enable transparent Lineage runtime\n")
	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, strings.TrimSpace(`
Lineage shareable agent runtime

Usage:
  lineage init user
  lineage init workspace <name>
  lineage package init <name>
  lineage enable <package-path-or-id>
  lineage run <claude|codex> [--dry-run] [-- provider args...]
  lineage install-shims
`))
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
