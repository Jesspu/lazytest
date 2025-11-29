package filesystem

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Ignorer handles file and directory ignoring logic based on default patterns and .gitignore.
type Ignorer struct {
	patterns []string
}

// NewIgnorer creates a new Ignorer and loads patterns from .gitignore if present.
func NewIgnorer(root string) *Ignorer {
	ign := &Ignorer{
		patterns: []string{
			"node_modules",
			".git",
			"dist",
			"build",
			"coverage",
			".DS_Store",
			"*.log",
		},
	}

	f, err := os.Open(filepath.Join(root, ".gitignore"))
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			ign.patterns = append(ign.patterns, line)
		}
	}
	return ign
}

// ShouldIgnore checks if the given path should be ignored.
// It checks against the file name (basename) and the relative path.
func (i *Ignorer) ShouldIgnore(path string, root string) bool {
	name := filepath.Base(path)
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		// If we can't get relative path, just check name
		relPath = name
	}

	for _, p := range i.patterns {
		// Handle directory marker
		cleanP := strings.TrimSuffix(p, "/")

		// Handle anchor
		isAnchored := strings.HasPrefix(cleanP, "/")
		cleanP = strings.TrimPrefix(cleanP, "/")

		// If it was anchored, we match against relPath
		if isAnchored {
			if relPath == cleanP || strings.HasPrefix(relPath, cleanP+string(os.PathSeparator)) {
				return true
			}
			continue
		}

		// If not anchored

		// Check if it matches name (basename)
		// This covers "node_modules", "*.log", "ignored_dir/" (cleanP="ignored_dir")
		matched, _ := filepath.Match(cleanP, name)
		if matched {
			return true
		}

		// Check if it matches relPath (for patterns like "src/foo")
		if relPath == cleanP || strings.HasPrefix(relPath, cleanP+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
