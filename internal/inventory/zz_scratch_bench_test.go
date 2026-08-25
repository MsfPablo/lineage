package inventory

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func mustWriteB(tb testing.TB, p, content string) {
	tb.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		tb.Fatal(err)
	}
}

func synth(tb testing.TB, nFiles int, mdFrac float64, mdLines int) string {
	tb.Helper()
	root := tb.TempDir()
	rng := rand.New(rand.NewSource(1))
	nMD := int(float64(nFiles) * mdFrac)
	nOther := nFiles - nMD
	var scripts []string
	dirs := []string{"scripts", "skills/alpha", "skills/beta", "references", "workflows/deploy", "lib/util", "data/2024-01-01"}
	for i := 0; i < nOther; i++ {
		d := dirs[i%len(dirs)]
		p := fmt.Sprintf("%s/file%04d.sh", d, i)
		scripts = append(scripts, p)
		mustWriteB(tb, filepath.Join(root, filepath.FromSlash(p)), "#!/bin/sh\necho hi\n")
	}
	words := []string{"the", "workflow", "runs", "then", "check", "output", "and", "verify", "before", "shipping", "carefully", "again"}
	for i := 0; i < nMD; i++ {
		d := dirs[i%len(dirs)]
		var sb strings.Builder
		for l := 0; l < mdLines; l++ {
			line := make([]string, 0, 10)
			for w := 0; w < 10; w++ {
				line = append(line, words[rng.Intn(len(words))])
			}
			if l%12 == 0 && len(scripts) > 0 {
				line[4] = scripts[rng.Intn(len(scripts))]
			}
			sb.WriteString(strings.Join(line, " "))
			sb.WriteString("\n")
		}
		p := fmt.Sprintf("%s/doc%04d.md", d, i)
		mustWriteB(tb, filepath.Join(root, filepath.FromSlash(p)), sb.String())
	}
	return root
}

var sizes = []int{50, 100, 250, 500, 1000, 2000, 5000}

func BenchmarkDiscoverTotal(b *testing.B) {
	for _, n := range sizes {
		root := synth(b, n, 0.2, 120)
		b.Run(fmt.Sprintf("files=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := Discover(root); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// prepare walks+digests only, so crossReference can be timed in isolation.
func prepare(b *testing.B, root string) ([]string, map[string]*Entry) {
	inv, err := Discover(root)
	if err != nil {
		b.Fatal(err)
	}
	relPaths := make([]string, 0, len(inv.Entries))
	entries := make(map[string]*Entry, len(inv.Entries))
	for _, e := range inv.Entries {
		relPaths = append(relPaths, e.Path)
		c := e
		c.Mentions, c.ReferencedBy = nil, nil
		entries[e.Path] = &c
	}
	return relPaths, entries
}

func BenchmarkCrossReferenceOnly(b *testing.B) {
	for _, n := range sizes {
		root := synth(b, n, 0.2, 120)
		b.Run(fmt.Sprintf("files=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				relPaths, entries := prepare(b, root)
				b.StartTimer()
				if err := crossReference(root, relPaths, entries); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDigestOnly(b *testing.B) {
	for _, n := range sizes {
		root := synth(b, n, 0.2, 120)
		var files []string
		filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				files = append(files, p)
			}
			return nil
		})
		b.Run(fmt.Sprintf("files=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, f := range files {
					if _, err := digestFile(f); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

// ---- indexed alternative, for crossover measurement ----

type idxHit struct {
	targetIdx int
	col       int
	kind      MatchKind
	matched   string
}

func isTokByte(b byte) bool {
	return isPathTokenByte(b) || b == '.' || b == '/'
}

func crossReferenceIndexed(root string, relPaths []string, entries map[string]*Entry) error {
	candidates := buildCandidates(relPaths)
	for _, c := range candidates {
		if c.base == "" {
			entries[c.path].AmbiguousBasename = true
		}
	}
	// index: token -> candidate index, for both full path and unique basename
	byPath := make(map[string]int, len(candidates))
	byBase := make(map[string]int, len(candidates))
	for i, c := range candidates {
		byPath[c.path] = i
		if c.base != "" {
			byBase[c.base] = i
		}
	}

	hits := make([]idxHit, 0, 8)
	seen := make(map[int]int, 8) // candidate idx -> position in hits
	for _, rel := range relPaths {
		entry := entries[rel]
		if filepath.Ext(rel) != ".md" {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(rel))
		lines, err := readLines(full)
		if err != nil {
			return fmt.Errorf("read %s for citation scan: %w", rel, err)
		}
		for lineNo, line := range lines {
			hits = hits[:0]
			clear(seen)
			// tokenize
			for i := 0; i < len(line); {
				if !isTokByte(line[i]) {
					i++
					continue
				}
				j := i
				for j < len(line) && isTokByte(line[j]) {
					j++
				}
				tok := line[i:j]
				off := i
				i = j
				// trim trailing dots (sentence punctuation)
				for len(tok) > 0 && tok[len(tok)-1] == '.' {
					tok = tok[:len(tok)-1]
				}
				// every suffix beginning after a '/' is a legal candidate form
				for s := 0; s <= len(tok); {
					sub := tok[s:]
					if sub == "" {
						break
					}
					if ci, ok := byPath[sub]; ok {
						recordHit(&hits, seen, ci, off+s, MatchPath, sub)
					} else if ci, ok := byBase[sub]; ok {
						recordHit(&hits, seen, ci, off+s, MatchBasename, sub)
					}
					k := strings.IndexByte(sub, '/')
					if k < 0 {
						break
					}
					s += k + 1
				}
			}
			if len(hits) == 0 {
				continue
			}
			sort.Slice(hits, func(a, b int) bool { return hits[a].targetIdx < hits[b].targetIdx })
			snippet := truncate(strings.TrimSpace(line), maxSnippetLen)
			for _, h := range hits {
				target := candidates[h.targetIdx]
				if target.path == rel {
					continue
				}
				citation := Citation{
					FromPath: rel, ToPath: target.path, Line: lineNo + 1,
					Column: h.col + 1, MatchKind: h.kind, MatchedText: h.matched, Snippet: snippet,
				}
				entry.Mentions = append(entry.Mentions, citation)
				if te, ok := entries[target.path]; ok {
					te.ReferencedBy = append(te.ReferencedBy, citation)
				}
			}
		}
	}
	return nil
}

func recordHit(hits *[]idxHit, seen map[int]int, ci, col int, kind MatchKind, matched string) {
	if p, ok := seen[ci]; ok {
		h := (*hits)[p]
		// path beats basename; otherwise first (leftmost) wins
		if h.kind == MatchBasename && kind == MatchPath {
			(*hits)[p] = idxHit{ci, col, kind, matched}
		}
		return
	}
	seen[ci] = len(*hits)
	*hits = append(*hits, idxHit{ci, col, kind, matched})
}

func BenchmarkCrossReferenceIndexed(b *testing.B) {
	for _, n := range sizes {
		root := synth(b, n, 0.2, 120)
		b.Run(fmt.Sprintf("files=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				relPaths, entries := prepare(b, root)
				b.StartTimer()
				if err := crossReferenceIndexed(root, relPaths, entries); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestIndexedEquivalence checks the indexed variant reproduces the linear one
// on the real fixtures.
func TestScratchIndexedEquivalence(t *testing.T) {
	root := synth(t, 200, 0.2, 60)
	rp1, e1 := prepareT(t, root)
	if err := crossReference(root, rp1, e1); err != nil {
		t.Fatal(err)
	}
	rp2, e2 := prepareT(t, root)
	if err := crossReferenceIndexed(root, rp2, e2); err != nil {
		t.Fatal(err)
	}
	diff := 0
	for _, p := range rp1 {
		a, b := e1[p], e2[p]
		if len(a.Mentions) != len(b.Mentions) || len(a.ReferencedBy) != len(b.ReferencedBy) {
			diff++
			if diff < 5 {
				t.Errorf("%s: linear mentions=%d refby=%d, indexed mentions=%d refby=%d", p, len(a.Mentions), len(a.ReferencedBy), len(b.Mentions), len(b.ReferencedBy))
			}
			continue
		}
		for i := range a.Mentions {
			if a.Mentions[i] != b.Mentions[i] {
				diff++
				if diff < 5 {
					t.Errorf("%s mention %d:\n linear=%+v\nindexed=%+v", p, i, a.Mentions[i], b.Mentions[i])
				}
			}
		}
	}
	t.Logf("total diffs: %d", diff)
}

func prepareT(t *testing.T, root string) ([]string, map[string]*Entry) {
	inv, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	relPaths := make([]string, 0, len(inv.Entries))
	entries := make(map[string]*Entry, len(inv.Entries))
	for _, e := range inv.Entries {
		relPaths = append(relPaths, e.Path)
		c := e
		c.Mentions, c.ReferencedBy = nil, nil
		entries[e.Path] = &c
	}
	return relPaths, entries
}
