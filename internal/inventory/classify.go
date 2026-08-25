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
// relative path or bare filename. Each (line, target) pair becomes one
// Citation — a directed FromPath -> ToPath edge — recorded on both the
// citer's Mentions and the target's ReferencedBy, so the graph reads from
// either end. A line naming the same target twice yields one citation,
// anchored at the first match.
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
//
// Cost is O(markdown lines x candidates), and each markdown file is read
// fully into memory. That is an accepted v1 tradeoff: the expected input is a
// single agent workspace, tens to low hundreds of files. Replacing the scan
// with a token index is the obvious lever if that stops holding — note that
// the citation order for a line falls out of the sorted candidate walk (see
// buildCandidates), so an index-based rewrite has to sort each line's matches
// explicitly to stay deterministic.
func crossReference(root string, relPaths []string, entries map[string]*Entry) error {
	candidates := buildCandidates(relPaths)

	// buildCandidates already worked out which basenames collide; surface that
	// on the entry rather than counting basenames a second time.
	for _, c := range candidates {
		if c.baseShared {
			entries[c.path].AmbiguousBasename = true
		}
	}

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
				at, kind, matched := matchTarget(line, target)
				if at < 0 {
					continue
				}
				citation := Citation{
					FromPath:    rel,
					ToPath:      target.path,
					Line:        lineNo + 1,
					Column:      at + 1,
					MatchKind:   kind,
					MatchedText: matched,
					Snippet:     snippetOf(line),
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

// isPathTokenByte reports whether b can continue a file-path token. A match
// flanked by one of these is part of a longer name rather than a reference to
// the candidate itself: "scripts/deploy.sh" found inside
// "scripts/deploy.sh.bak" is the backup being named, not the script.
//
// "/" is deliberately excluded. Prose routinely names an inventoried file
// through a relative prefix ("see ../../references/notes.csv"), and treating
// the separator as a continuation byte would discard those real citations.
// The cost is that a path mention which merely ends with an inventoried path
// still matches; that is far rarer than the relative-prefix case.
func isPathTokenByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-'
}

// findPathToken returns the byte offset of the first occurrence of needle in
// line that stands on its own path-token boundaries, or -1 if there is none.
// It is the guard against citing a file that was never actually mentioned.
func findPathToken(line, needle string) int {
	if needle == "" {
		return -1
	}
	for start := 0; ; {
		i := strings.Index(line[start:], needle)
		if i < 0 {
			return -1
		}
		i += start
		if boundedLeft(line, i) && boundedRight(line, i+len(needle)) {
			return i
		}
		start = i + 1
	}
}

func boundedLeft(line string, i int) bool {
	if i == 0 {
		return true
	}
	prev := line[i-1]
	// A leading "." would make this the tail of a dotted name ("v2.deploy.sh").
	return !isPathTokenByte(prev) && prev != '.'
}

func boundedRight(line string, end int) bool {
	if end >= len(line) {
		return true
	}
	next := line[end]
	if isPathTokenByte(next) {
		return false
	}
	if next != '.' {
		return true
	}
	// A "." continues the token only when a name follows it: ".bak" in
	// "deploy.sh.bak" disqualifies the match, but the sentence-ending period
	// in "run deploy.sh." does not.
	return end+1 >= len(line) || !isPathTokenByte(line[end+1])
}

// matchTarget reports where and how line names target, along with the
// reference exactly as the prose wrote it. A full-path mention is the stronger
// claim, so it is tried first and wins when both forms match the same line.
func matchTarget(line string, target candidate) (int, MatchKind, string) {
	if at := findPathToken(line, target.path); at >= 0 {
		start, text := writtenRef(line, at, len(target.path))
		return start, MatchPath, text
	}
	if target.baseShared {
		return -1, "", ""
	}
	if at := findPathToken(line, target.base); at >= 0 {
		start, text := writtenRef(line, at, len(target.base))
		return start, MatchBasename, text
	}
	return -1, "", ""
}

// writtenRef widens a match to the whole path reference around it, so a
// citation can report what the prose actually said rather than the needle that
// found it. A line naming "../../references/data.csv" cites
// references/data.csv, but the relative prefix is part of the reference and a
// consumer resolving it needs to see it. Returns the offset and text of the
// widened span.
//
// boundedLeft already guarantees the byte before a match is neither a
// path-token byte nor ".", so this only ever widens across a "/" separator
// and whatever path bytes precede it.
func writtenRef(line string, at, length int) (int, string) {
	start := at
	for start > 0 {
		if b := line[start-1]; !isPathTokenByte(b) && b != '/' && b != '.' {
			break
		}
		start--
	}
	return start, line[start : at+length]
}

type candidate struct {
	path string // full relative path, forward-slashed
	base string // bare filename, always populated
	// baseShared marks a basename that collides with another file's, so
	// bare-name matching is suppressed for it and only a full-path mention
	// can cite it.
	baseShared bool
}

// buildCandidates returns one candidate per relative path, in the same
// (sorted) order Discover already computed relPaths in; crossReference walks
// them in that order, so a line's citations come out sorted by ToPath. Bare-filename matching is only offered
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
		out = append(out, candidate{path: rel, base: base, baseShared: baseCounts[base] > 1})
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

// snippetOf trims and bounds a source line for use as Citation.Snippet.
//
// The clone is load-bearing. readLines splits one string covering the whole
// file, so trimming and truncating only re-slice: without copying, every
// snippet keeps its entire source file alive for as long as the Inventory
// lives, and retention scales with total markdown volume rather than with the
// number of citations. Measured on ten 500 KB files with one citation each,
// 520 bytes of snippet text held 5 MB of heap. Citation is documented as
// cheap to carry into a prompt, and that has to be true of its backing store
// too, not just the struct.
func snippetOf(line string) string {
	return strings.Clone(truncate(strings.TrimSpace(line), maxSnippetLen))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
