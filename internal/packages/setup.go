package packages

import (
	"fmt"
	"os"
	"path/filepath"
)

// SetupPlan is what applying a package's Setup declarations against a
// project root would do: every declared file/directory, and whether it
// already exists (a no-op) or would be created. Computing this separately
// from ApplySetup is what lets a caller show "here's what will happen"
// and get permission before anything is written (#72).
type SetupPlan struct {
	Files       []SetupFileAction
	Directories []SetupDirectoryAction
}

// NeedsAction reports whether applying plan would actually create
// anything. A plan where every declared file and directory already exists
// doesn't need a permission prompt - there's nothing new to approve.
func (p SetupPlan) NeedsAction() bool {
	for _, f := range p.Files {
		if !f.Exists {
			return true
		}
	}
	for _, d := range p.Directories {
		if !d.Exists {
			return true
		}
	}
	return false
}

type SetupFileAction struct {
	Path        string // relative to the project root
	Description string
	Template    string
	Exists      bool
}

type SetupDirectoryAction struct {
	Path        string // relative to the project root
	Description string
	Exists      bool
}

// PlanSetup resolves every path setup declares against projectRoot and
// checks what's already there, without creating anything. A setup path is
// publisher-declared, untrusted input like every other package-controlled
// path in this project, so it goes through SafeJoin the same way archive
// entries and manifest-declared entrypoints do.
func PlanSetup(projectRoot string, setup Setup) (SetupPlan, error) {
	var plan SetupPlan
	for _, f := range setup.Files {
		abs, err := SafeJoin(projectRoot, f.Path)
		if err != nil {
			return SetupPlan{}, fmt.Errorf("setup file %q: %w", f.Path, err)
		}
		_, statErr := os.Stat(abs)
		if statErr != nil && !os.IsNotExist(statErr) {
			return SetupPlan{}, fmt.Errorf("check setup file %q: %w", f.Path, statErr)
		}
		plan.Files = append(plan.Files, SetupFileAction{
			Path:        f.Path,
			Description: f.Description,
			Template:    f.Template,
			Exists:      statErr == nil,
		})
	}
	for _, d := range setup.Directories {
		abs, err := SafeJoin(projectRoot, d.Path)
		if err != nil {
			return SetupPlan{}, fmt.Errorf("setup directory %q: %w", d.Path, err)
		}
		_, statErr := os.Stat(abs)
		if statErr != nil && !os.IsNotExist(statErr) {
			return SetupPlan{}, fmt.Errorf("check setup directory %q: %w", d.Path, statErr)
		}
		plan.Directories = append(plan.Directories, SetupDirectoryAction{
			Path:        d.Path,
			Description: d.Description,
			Exists:      statErr == nil,
		})
	}
	return plan, nil
}

// ApplySetup creates whatever plan says doesn't already exist: missing
// files get their declared template content, missing directories get
// created empty. Anything already present is left completely untouched -
// this is what makes ApplySetup safe to call more than once (#72's
// idempotent/resumable requirement): a second call after a first partial
// success only creates what's still missing, and a call against a project
// that already has everything does nothing at all.
func ApplySetup(projectRoot string, plan SetupPlan) error {
	for _, f := range plan.Files {
		if f.Exists {
			continue
		}
		abs, err := SafeJoin(projectRoot, f.Path)
		if err != nil {
			return fmt.Errorf("setup file %q: %w", f.Path, err)
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return fmt.Errorf("create directory for %q: %w", f.Path, err)
		}
		if err := os.WriteFile(abs, []byte(f.Template), 0o644); err != nil {
			return fmt.Errorf("create %q: %w", f.Path, err)
		}
	}
	for _, d := range plan.Directories {
		if d.Exists {
			continue
		}
		abs, err := SafeJoin(projectRoot, d.Path)
		if err != nil {
			return fmt.Errorf("setup directory %q: %w", d.Path, err)
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("create directory %q: %w", d.Path, err)
		}
	}
	return nil
}
