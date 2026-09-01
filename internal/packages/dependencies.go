package packages

import (
	"fmt"
	"sort"
	"strings"
)

// ValidateDependencies checks two invariants over the set of packages that will
// be enabled together:
//
//  1. No two enabled packages export the same skill name. A skill name must
//     resolve to exactly one providing package, otherwise the launcher cannot
//     decide which implementation to wire into the run — and a silent,
//     order-dependent pick is worse than failing loudly. On conflict it
//     returns an error naming the skill and every package that exports it.
//
//  2. Every package's Requires.Skills is satisfied somewhere across pkgs — its
//     own skills, or another package's. It runs against the full enabled set
//     (not one package in isolation) because a required skill can legitimately
//     come from a different enabled package.
//
// Skill matching is by bare name only. Version-range resolution is not
// implemented: two packages exporting the same name at different versions still
// conflict, since there is no rule to pick between them.
func ValidateDependencies(pkgs []Package) error {
	// skill -> package names that export it, in input order (sorted for the
	// error message below). A skill with more than one provider is a conflict.
	providers := map[string][]string{}
	for _, pkg := range pkgs {
		for _, skill := range pkg.Skills {
			providers[skill] = append(providers[skill], pkg.Manifest.Name)
		}
	}

	for _, pkg := range pkgs {
		// A package does not conflict with itself merely because it requires a
		// skill it also exports; that's the satisfied-by-own-skill case.
		for _, required := range pkg.Manifest.Requires.Skills {
			if _, ok := providers[required]; !ok {
				return fmt.Errorf("package %q requires skill %q, which is not provided by any enabled package", pkg.Manifest.Name, required)
			}
		}
	}

	// Report duplicate exports deterministically: sort by skill name so the
	// error is stable regardless of package/skill input order.
	var conflicts []string
	for skill, names := range providers {
		if len(names) < 2 {
			continue
		}
		sort.Strings(names)
		conflicts = append(conflicts, fmt.Sprintf("skill %q is exported by %s", skill, providerPhrase(names)))
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return fmt.Errorf("cannot enable package set: %s", strings.Join(conflicts, "; "))
	}
	return nil
}

// providerPhrase renders a list of package names as a quoted phrase suitable
// for a conflict error: ["foo","bar"] -> `both "foo" and "bar"`, and
// ["a","b","c"] -> `"a", "b", and "c"`.
func providerPhrase(names []string) string {
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = fmt.Sprintf("%q", n)
	}
	switch len(quoted) {
	case 0:
		return ""
	case 1:
		return quoted[0]
	case 2:
		return "both " + quoted[0] + " and " + quoted[1]
	default:
		return strings.Join(quoted[:len(quoted)-1], ", ") + ", and " + quoted[len(quoted)-1]
	}
}
