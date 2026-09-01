package packages

import (
	"strings"
	"testing"
)

func TestValidateDependenciesSatisfiedByOwnSkill(t *testing.T) {
	pkg := Package{
		Manifest: Manifest{Name: "self-sufficient", Requires: Requires{Skills: []string{"review"}}},
		Skills:   []string{"review"},
	}
	if err := ValidateDependencies([]Package{pkg}); err != nil {
		t.Fatalf("ValidateDependencies() error = %v, want nil", err)
	}
}

func TestValidateDependenciesSatisfiedByAnotherPackage(t *testing.T) {
	dependent := Package{
		Manifest: Manifest{Name: "dependent", Requires: Requires{Skills: []string{"security-basics"}}},
	}
	provider := Package{
		Manifest: Manifest{Name: "provider"},
		Skills:   []string{"security-basics"},
	}
	if err := ValidateDependencies([]Package{dependent, provider}); err != nil {
		t.Fatalf("ValidateDependencies() error = %v, want nil", err)
	}
}

func TestValidateDependenciesMissingSkillFails(t *testing.T) {
	dependent := Package{
		Manifest: Manifest{Name: "dependent", Requires: Requires{Skills: []string{"nonexistent"}}},
	}
	err := ValidateDependencies([]Package{dependent})
	if err == nil {
		t.Fatal("ValidateDependencies() error = nil, want error for missing required skill")
	}
}

func TestValidateDependenciesRejectsDuplicateSkillExport(t *testing.T) {
	foo := Package{Manifest: Manifest{Name: "foo"}, Skills: []string{"research"}}
	bar := Package{Manifest: Manifest{Name: "bar"}, Skills: []string{"research"}}

	err := ValidateDependencies([]Package{foo, bar})
	if err == nil {
		t.Fatal("ValidateDependencies() error = nil, want error for duplicate skill export")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"research"`) {
		t.Errorf("error %q does not name the conflicting skill", msg)
	}
	if !strings.Contains(msg, `"foo"`) || !strings.Contains(msg, `"bar"`) {
		t.Errorf("error %q does not name every conflicting package", msg)
	}
}

// The conflict must be reported regardless of which package is supplied first,
// so a caller cannot dodge the error by reordering enabled packages.
func TestValidateDependenciesDuplicateExportIsOrderIndependent(t *testing.T) {
	foo := Package{Manifest: Manifest{Name: "foo"}, Skills: []string{"research"}}
	bar := Package{Manifest: Manifest{Name: "bar"}, Skills: []string{"research"}}

	first := ValidateDependencies([]Package{foo, bar})
	second := ValidateDependencies([]Package{bar, foo})
	if first == nil || second == nil {
		t.Fatal("expected both orderings to fail")
	}
	if first.Error() != second.Error() {
		t.Errorf("error differs by input order: first=%q second=%q", first, second)
	}
}

// Three or more providers of the same skill must all be named.
func TestValidateDependenciesRejectsThreeOrMoreProviders(t *testing.T) {
	pkgs := []Package{
		{Manifest: Manifest{Name: "alpha"}, Skills: []string{"research"}},
		{Manifest: Manifest{Name: "beta"}, Skills: []string{"research"}},
		{Manifest: Manifest{Name: "gamma"}, Skills: []string{"research"}},
	}
	err := ValidateDependencies(pkgs)
	if err == nil {
		t.Fatal("ValidateDependencies() error = nil, want error for 3+ providers")
	}
	msg := err.Error()
	for _, name := range []string{`"alpha"`, `"beta"`, `"gamma"`} {
		if !strings.Contains(msg, name) {
			t.Errorf("error %q does not name provider %s", msg, name)
		}
	}
}

// A package requiring a skill it also exports is the satisfied-by-own-skill
// case, not a duplicate — there is only one provider.
func TestValidateDependenciesSelfProvidesAndRequiresIsNotADuplicate(t *testing.T) {
	pkg := Package{
		Manifest: Manifest{Name: "self", Requires: Requires{Skills: []string{"research"}}},
		Skills:   []string{"research"},
	}
	if err := ValidateDependencies([]Package{pkg}); err != nil {
		t.Fatalf("ValidateDependencies() error = %v, want nil (self-provided skill is not a duplicate)", err)
	}
}

// A unique provider plus a satisfied cross-package requirement must still pass.
func TestValidateDependenciesUniqueProvidersAndResolvedRequirementsPass(t *testing.T) {
	pkgs := []Package{
		{Manifest: Manifest{Name: "core", Requires: Requires{Skills: []string{"review"}}}, Skills: []string{"planning"}},
		{Manifest: Manifest{Name: "reviewer"}, Skills: []string{"review"}},
	}
	if err := ValidateDependencies([]Package{pkgs[0], pkgs[1]}); err != nil {
		t.Fatalf("ValidateDependencies() error = %v, want nil", err)
	}
}
