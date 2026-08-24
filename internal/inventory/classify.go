package inventory

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// instructionFileNames are exact basenames that always classify as
// instruction material, wherever they appear in the tree — these are the
// files an agent actually reads to know what to do.
var instructionFileNames = map[string]bool{
	"CLAUDE.md":   true,
	"AGENTS.md":   true,
	"SKILL.md":    true,
	"WORKFLOW.md": true,
}

// setupFilePrefixes are case-sensitive basename prefixes that classify as
// setup material: install/readme notes, not behavior itself.
var setupFilePrefixes = []string{"README", "SETUP", "INSTALL"}

// packageMetadataFileNames are exact basenames that classify as existing
// package metadata a source tree might already carry.
var packageMetadataFileNames = map[string]bool{
	"lineage.yaml": true,
	"package.yaml": true,
}

// executableExtensions classify as executable helpers by extension.
var executableExtensions = map[string]bool{
	".sh":  true,
	".py":  true,
	".js":  true,
	".ts":  true,
	".rb":  true,
	".ps1": true,
}

// referenceExtensions classify as reference/data material by extension,
// when not otherwise claimed by a more specific rule.
var referenceExtensions = map[string]bool{
	".pdf":  true,
	".csv":  true,
	".json": true,
	".yaml": true,
	".yml":  true,
	".txt":  true,
}

// languageByExtension is a best-effort, hand-rolled extension-to-language
// table — deliberately small and explicit rather than pulling in a
// mimetype/language-detection dependency, matching the rest of this
// codebase's style.
var languageByExtension = map[string]string{
	".go":   "go",
	".py":   "python",
	".js":   "javascript",
	".ts":   "typescript",
	".sh":   "shell",
	".rb":   "ruby",
	".ps1":  "powershell",
	".md":   "markdown",
	".yaml": "yaml",
	".yml":  "yaml",
	".json": "json",
}

// classify assigns an ArtifactKind and a base reason to a file by its
// relative path, using name/extension heuristics only. A file that matches
// no rule is KindUnknown with an honest "no classification rule matched"
// reason rather than a forced guess — ambiguity is surfaced, not resolved,
// here.
func classify(rel string) (ArtifactKind, string) {
	base := path.Base(rel)
	ext := path.Ext(base)
	dir := path.Dir(rel)

	if instructionFileNames[base] {
		return KindInstruction, fmt.Sprintf("filename %q is a recognized instruction file", base)
	}
	if packageMetadataFileNames[base] {
		return KindPackageMetadata, fmt.Sprintf("filename %q is existing package metadata", base)
	}
	// A markdown file sitting directly in a skill's or workflow's own
	// top-level folder is very likely that unit's instruction content even
	// when it doesn't use the canonical SKILL.md/WORKFLOW.md filename —
	// e.g. skills/foo/README.md or workflows/deploy.md from a workspace
	// that hasn't adopted Lineage's naming convention yet. Directory
	// location outranks a generic README/SETUP prefix match here: a README
	// directly inside a skill folder means something different from a
	// README at the workspace root.
	//
	// Deliberately shallow: only skills/<name>/*.md and workflows/*.md or
	// workflows/<name>/*.md qualify — a markdown file nested any deeper
	// (skills/<name>/templates/*.md, skills/<name>/outputs/*.md) is
	// auxiliary material, not the skill's own instructions, and matching it
	// anyway was a real false-positive found by testing this rule against a
	// real repo's skills tree (eval logs and templates got misclassified as
	// instruction). Deeper markdown falls through to the generic unknown
	// markdown case below instead of a guess.
	if ext == ".md" && isSkillOrWorkflowOwnFile(dir) {
		return KindInstruction, fmt.Sprintf("markdown file is this skill/workflow/agent's own top-level file (%s), likely instruction content despite non-canonical filename", dir)
	}
	for _, prefix := range setupFilePrefixes {
		if strings.HasPrefix(base, prefix) {
			return KindSetupMaterial, fmt.Sprintf("filename %q matches setup-note prefix %q", base, prefix)
		}
	}
	if inReferencesDir(dir) {
		return KindReference, fmt.Sprintf("located under a references directory (%s)", dir)
	}
	if executableExtensions[ext] {
		return KindExecutableHelper, fmt.Sprintf("extension %q is a recognized script/executable type", ext)
	}
	if referenceExtensions[ext] {
		return KindReference, fmt.Sprintf("extension %q is a recognized reference/data type", ext)
	}
	if ext == ".md" {
		return KindUnknown, "unrecognized markdown file — possible instruction content, review manually"
	}
	return KindUnknown, "no classification rule matched"
}

// inReferencesDir reports whether dir (a forward-slashed relative
// directory) is, or is nested under, a directory literally named
// "references" or "refs".
func inReferencesDir(dir string) bool {
	return underDir(dir, "references", "refs")
}

// underDir reports whether dir (a forward-slashed relative directory) is,
// or is nested under, a directory whose name matches any of names.
func underDir(dir string, names ...string) bool {
	for _, seg := range strings.Split(dir, "/") {
		for _, name := range names {
			if seg == name {
				return true
			}
		}
	}
	return false
}

// knownProviderRoots are directory prefixes real workspaces commonly nest
// skills/workflows/agents content under, mirroring internal/provider's
// registry (SkillsDir ".claude/skills" for claude, ".agents/skills" for
// codex) — found necessary by testing against a real repo whose agent
// definitions live at .claude/agents/*.md, which the bare (unprefixed)
// shape below doesn't reach on its own.
var knownProviderRoots = []string{".claude", ".agents"}

// isSkillOrWorkflowOwnFile reports whether dir is exactly one of the shapes
// that mean "this skill's/workflow's/agent's own top-level file", not
// merely nested somewhere underneath one, optionally under a known
// provider root (see knownProviderRoots): "workflows" or "agents"
// themselves (a flat workflows/<name>.md or agents/<name>.md file,
// matching how Lineage's own agents/ content dir works — flat files, no
// per-agent subfolder), or a single named child of "skills" or "workflows"
// (skills/<name> or workflows/<name>, a per-unit folder). Anything
// deeper — skills/<name>/templates, skills/<name>/outputs, and so on — is
// auxiliary material colocated with the skill, not the skill's own
// instructions, and intentionally does not match here (found to matter by
// testing against a real repo whose eval logs and templates live exactly
// one level deeper than this).
func isSkillOrWorkflowOwnFile(dir string) bool {
	segs := strings.Split(dir, "/")
	for _, prefix := range knownProviderRoots {
		if len(segs) > 1 && segs[0] == prefix {
			segs = segs[1:]
			break
		}
	}
	if len(segs) == 1 && (segs[0] == "workflows" || segs[0] == "agents") {
		return true
	}
	if len(segs) == 2 && (segs[0] == "skills" || segs[0] == "workflows") {
		return true
	}
	return false
}

// languageFor returns a best-effort language label by extension, or "" if
// the extension isn't in the known table.
func languageFor(rel string) string {
	return languageByExtension[path.Ext(rel)]
}

// crossReference implements pass 2: every markdown file's content is
// scanned line-by-line for literal occurrences of any other entry's
// relative path or bare filename. Each match becomes one Citation, recorded
// on both the citer's Mentions and the target's ReferencedBy.
//
// Eligibility to cite is based on being markdown (prose), not on Kind: a
// real messy workspace has plenty of instruction-shaped .md files that
// don't match the canonical CLAUDE.md/AGENTS.md/SKILL.md/WORKFLOW.md names
// or a skills/workflows directory, and classify() correctly leaves those as
// KindUnknown rather than guessing — but they still literally name other
// files in their prose, and that's real, citable evidence regardless of
// whether classify() could confidently label the citing file itself.
//
// This is deliberately plain string matching, not any form of semantic
// analysis — see the package doc comment for why that boundary is
// intentional.
func crossReference(root string, relPaths []string, entries map[string]*Entry) error {
	candidates := buildCandidates(relPaths)

	for _, rel := range relPaths {
		entry := entries[rel]
		if path.Ext(rel) != ".md" {
			continue
		}

		full := filepath.Join(root, filepath.FromSlash(rel))
		lines, err := readLines(full)
		if err != nil {
			return fmt.Errorf("read %s for citation scan: %w", rel, err)
		}

		for lineNo, line := range lines {
			for _, target := range candidates {
				if target.path == rel {
					continue // a file never cites itself
				}
				matchesPath := strings.Contains(line, target.path)
				matchesBase := target.base != "" && strings.Contains(line, target.base)
				if !matchesPath && !matchesBase {
					continue
				}
				citation := Citation{
					FromPath: rel,
					Line:     lineNo + 1,
					Snippet:  truncate(strings.TrimSpace(line), maxSnippetLen),
				}
				entry.Mentions = append(entry.Mentions, citation)
				if targetEntry, ok := entries[target.path]; ok {
					targetEntry.ReferencedBy = append(targetEntry.ReferencedBy, citation)
				}
			}
		}
	}
	return nil
}

type candidate struct {
	path string // full relative path, forward-slashed
	base string // bare filename
}

// buildCandidates returns one candidate per relative path, in the same
// (sorted) order Discover already computed relPaths in, so citation output
// order is deterministic run to run. Bare-filename matching is only offered
// for basenames that are unique across the whole tree: on a real workspace,
// generic names like "config.json" or dated output like
// "workspace/2024-01-01/report.json" can share a basename with dozens of
// unrelated files, and matching all of them turns one real citation into a
// flood of false ones. An ambiguous basename still matches via its full
// relative path (unambiguous), just not by bare name.
func buildCandidates(relPaths []string) []candidate {
	baseCounts := make(map[string]int, len(relPaths))
	for _, rel := range relPaths {
		baseCounts[path.Base(rel)]++
	}

	out := make([]candidate, 0, len(relPaths))
	for _, rel := range relPaths {
		base := path.Base(rel)
		if baseCounts[base] > 1 {
			base = ""
		}
		out = append(out, candidate{path: rel, base: base})
	}
	return out
}

func readLines(fullPath string) ([]string, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
