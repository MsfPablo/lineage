// Package inventory walks an arbitrary, possibly-messy source workspace —
// one that is not yet a valid Lineage package — and produces a deterministic,
// read-only inventory of what's there: every file's kind, why it was
// classified that way, and which markdown files (whether or not classify()
// could confidently name their role — canonical instruction files like
// CLAUDE.md/AGENTS.md/SKILL.md/WORKFLOW.md, or an unclassified README-style
// file that still contains real prose) literally name it in their text.
//
// This package deliberately does not interpret, execute, or understand
// intent. Citation matching (see Citation) is plain string/path matching
// against literal filenames — it catches "run scripts/deploy.sh" but not
// "run the deploy script" with no filename in it. That gap is intentional:
// resolving paraphrased or implicit references requires real understanding,
// which belongs to a later, model-assisted analysis stage that consumes
// this inventory (including Mentions) plus raw file content. Treat Mentions
// and ReferencedBy as a precise but incomplete evidence trail, never as a
// complete call graph.
//
// Because that later stage ingests this as evidence, every citation is a
// self-describing directed edge (see Citation): it names both ends, where in
// the source it was found, and how it matched, so the graph is walkable from
// either side without re-reading the workspace. Entry.AmbiguousBasename
// records where matching was deliberately withheld, so an empty ReferencedBy
// is never silently mistaken for "unused".
//
// Symlinks are refused rather than followed, including one passed as the
// discovery root itself: an inventory reports on the tree literally named,
// and quietly resolving elsewhere would make Inventory.Root a different path
// from the one the caller asked about.
package inventory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type ArtifactKind string

const (
	KindInstruction      ArtifactKind = "instruction"
	KindExecutableHelper ArtifactKind = "executable_helper"
	KindReference        ArtifactKind = "reference"
	KindSetupMaterial    ArtifactKind = "setup_material"
	KindPackageMetadata  ArtifactKind = "package_metadata"
	KindUnknown          ArtifactKind = "unknown"
)

// MatchKind records how a citation was found, which is a statement about how
// strong the evidence is rather than how certain it is. A full relative path
// is a far more specific claim than a bare filename that merely happens to be
// unique in this tree, and a consumer weighing evidence should treat the two
// differently.
type MatchKind string

const (
	MatchPath     MatchKind = "path"
	MatchBasename MatchKind = "basename"
)

// Citation is one directed edge in the evidence graph: markdown file FromPath
// names inventoried artifact ToPath, on this line, by this literal text. It
// is the "why is this in scope" evidence, sized to be cheap to carry into an
// LLM prompt later without re-opening source files.
//
// The same Citation value is recorded twice — on the citer's Mentions and on
// the target's ReferencedBy — so the graph can be walked from either end.
// ToPath is therefore redundant on the ReferencedBy side; that redundancy is
// what makes a single edge self-describing wherever it is read.
type Citation struct {
	FromPath    string    `json:"from_path"`    // the markdown file doing the citing
	ToPath      string    `json:"to_path"`      // the inventoried artifact being named
	Line        int       `json:"line"`         // 1-indexed line number within FromPath
	Column      int       `json:"column"`       // 1-indexed byte offset of the match within that line
	MatchKind   MatchKind `json:"match_kind"`   // how it matched, i.e. how strong the evidence is
	MatchedText string    `json:"matched_text"` // the literal token as written in the source
	Snippet     string    `json:"snippet"`      // that line's text, truncated
}

// maxSnippetLen bounds Citation.Snippet so a citation is always small and
// boundable, regardless of how long the source line actually is.
const maxSnippetLen = 200

// Entry describes one file discovered under a workspace root.
type Entry struct {
	Path string       `json:"path"` // relative to Inventory.Root, forward-slashed
	Kind ArtifactKind `json:"kind"`
	// Reason is the base classification reason (the heuristic rule that
	// matched). It is independent of citations and is never overwritten by
	// the cross-reference pass.
	Reason   string `json:"reason"`
	Digest   string `json:"digest"` // "sha256:<hex>" over this file's content
	Size     int64  `json:"size"`
	Language string `json:"language,omitempty"` // by extension, best-effort

	// AmbiguousBasename is true when this file's basename is shared with at
	// least one other file in the tree, so bare-name citation matching was
	// suppressed for it (see buildCandidates) and only a full-path mention
	// could cite it.
	//
	// Without this flag an empty ReferencedBy is ambiguous: a consumer cannot
	// tell a genuinely unreferenced file from one whose weak matches were
	// deliberately withheld, and must not conclude "unused" from the second
	// case. Reported honestly rather than papered over.
	AmbiguousBasename bool `json:"ambiguous_basename,omitempty"`

	// Mentions is populated for every markdown (.md) entry, regardless of
	// Kind: the outgoing edges from this file, one Citation per (line,
	// target) pair naming another inventoried artifact. Read ToPath to learn
	// what was named. Citation eligibility depends only on being prose, not
	// on whether classify() could confidently label the citing file itself —
	// an unclassified markdown file can still cite.
	Mentions []Citation `json:"mentions,omitempty"`

	// ReferencedBy is populated on the target side, any kind: the incoming
	// edges naming this entry. Empty means literally unreferenced by any
	// markdown file's prose — a real, honestly-reported fact, not an
	// omission — but read AmbiguousBasename before drawing a conclusion
	// from it.
	ReferencedBy []Citation `json:"referenced_by,omitempty"`
}

// SchemaVersion is the version of the serialized Inventory shape. Discover
// stamps it on every result so a later consumer reading a stored inventory
// can tell which field set it was written with.
const SchemaVersion = 1

// Inventory is the deterministic result of Discover.
type Inventory struct {
	SchemaVersion int     `json:"schema_version"`
	Root          string  `json:"root"`
	Entries       []Entry `json:"entries"` // sorted by Path
}

// defaultIgnoredDirs are directory names skipped anywhere in the tree by
// default: version control, dependency folders, and common build output.
var defaultIgnoredDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"vendor":        true,
	"dist":          true,
	"build":         true,
	"venv":          true,
	".venv":         true,
	"__pycache__":   true,
	".pytest_cache": true,
	".mypy_cache":   true,
	".tox":          true,
	".idea":         true,
	".vscode":       true,
	"target":        true,
	".next":         true,
}

// Discover walks root read-only and returns a deterministic inventory. It
// never creates, edits, deletes, or executes anything under root.
func Discover(root string) (Inventory, error) {
	inv := Inventory{SchemaVersion: SchemaVersion, Root: root}

	var relPaths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if isIgnoredDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refusing to include symlink %s", rel)
		}
		if !d.Type().IsRegular() {
			return nil
		}

		relPaths = append(relPaths, rel)
		return nil
	})
	if err != nil {
		return Inventory{}, err
	}
	sort.Strings(relPaths)

	entries := make(map[string]*Entry, len(relPaths))
	for _, rel := range relPaths {
		full := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Lstat(full)
		if statErr != nil {
			return Inventory{}, fmt.Errorf("stat %s: %w", rel, statErr)
		}

		digest, digestErr := digestFile(full)
		if digestErr != nil {
			return Inventory{}, fmt.Errorf("digest %s: %w", rel, digestErr)
		}

		kind, reason := classify(rel)
		entries[rel] = &Entry{
			Path:     rel,
			Kind:     kind,
			Reason:   reason,
			Digest:   digest,
			Size:     info.Size(),
			Language: languageFor(rel),
		}
	}

	if err := crossReference(root, relPaths, entries); err != nil {
		return Inventory{}, err
	}

	result := make([]Entry, 0, len(entries))
	for _, rel := range relPaths {
		result = append(result, *entries[rel])
	}
	inv.Entries = result
	return inv, nil
}

func isIgnoredDir(name string) bool {
	return defaultIgnoredDirs[name]
}

func digestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}
