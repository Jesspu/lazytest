package analysis

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Parser handles parsing of source files to extract dependencies.
type Parser struct{}

// NewParser creates a new Parser.
func NewParser() *Parser {
	return &Parser{}
}

// Import regex patterns
var (
	// import ... from '...'
	// Use [\s\S]*? to match across newlines non-greedily
	importFromRegex = regexp.MustCompile(`import[\s\S]*?from\s+['"]([^'"]+)['"]`)
	// import '...'
	importSideEffectRegex = regexp.MustCompile(`import\s+['"]([^'"]+)['"]`)
	// require('...')
	requireRegex = regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`)
)

// ImportResult contains resolved and unresolved imports.
type ImportResult struct {
	Resolved   []string
	Unresolved []UnresolvedImport
}

type UnresolvedImport struct {
	Path       string // The raw import string (e.g. "./utils")
	SourcePath string // The file doing the import
}

// ParseImports extracts imported file paths from a given file.
func (p *Parser) ParseImports(filePath string) (*ImportResult, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var rawImports []string
	text := string(content)

	// Check for "import ... from"
	matches := importFromRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			rawImports = append(rawImports, match[1])
		}
	}

	// Check for "import '...'"
	matches = importSideEffectRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			rawImports = append(rawImports, match[1])
		}
	}

	// Check for "require"
	matches = requireRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			rawImports = append(rawImports, match[1])
		}
	}

	return p.resolvePaths(filePath, rawImports), nil
}

// resolvePaths converts relative imports to absolute paths.
func (p *Parser) resolvePaths(sourcePath string, imports []string) *ImportResult {
	result := &ImportResult{
		Resolved:   []string{},
		Unresolved: []UnresolvedImport{},
	}
	dir := filepath.Dir(sourcePath)

	for _, imp := range imports {
		// Skip non-relative imports (node_modules) for now
		if !strings.HasPrefix(imp, ".") {
			continue
		}

		absPath := filepath.Join(dir, imp)

		// Try to find the file with extensions
		if foundPath, ok := p.findFile(absPath); ok {
			result.Resolved = append(result.Resolved, foundPath)
		} else {
			// Store as unresolved, but we need the POTENTIAL absolute path (without extension)
			// to match against later.
			result.Unresolved = append(result.Unresolved, UnresolvedImport{
				Path:       absPath, // This is the absolute path prefix (e.g. /path/to/utils)
				SourcePath: sourcePath,
			})
		}
	}

	return result
}

// findFile attempts to find a file by adding common extensions.
func (p *Parser) findFile(pathWithoutExt string) (string, bool) {
	extensions := []string{"", ".ts", ".js", ".tsx", ".jsx", "/index.ts", "/index.js", "/index.tsx", "/index.jsx"}

	for _, ext := range extensions {
		fullPath := pathWithoutExt + ext
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			// Found a match, now get the actual on-disk name to handle case sensitivity
			dir := filepath.Dir(fullPath)
			base := filepath.Base(fullPath)

			entries, err := os.ReadDir(dir)
			if err != nil {
				// Fallback to fullPath if we can't read dir
				return fullPath, true
			}

			for _, entry := range entries {
				if strings.EqualFold(entry.Name(), base) {
					return filepath.Join(dir, entry.Name()), true
				}
			}

			return fullPath, true
		}
	}

	return "", false
}
