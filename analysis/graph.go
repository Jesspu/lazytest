package analysis

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/jesspatton/lazytest/filesystem"
)

// Graph represents the dependency graph of the project.
type Graph struct {
	// Forward: File -> [Dependencies]
	Forward map[string][]string
	// Reverse: File -> [Dependents] (Files that import this file)
	Reverse map[string][]string
	// PendingImports: ImportPath -> [Dependents] (Files that import a path that wasn't found yet)
	PendingImports map[string][]string

	parser *Parser
	mu     sync.RWMutex
}

// NewGraph creates a new dependency graph.
func NewGraph() *Graph {
	return &Graph{
		Forward:        make(map[string][]string),
		Reverse:        make(map[string][]string),
		PendingImports: make(map[string][]string),
		parser:         NewParser(),
	}
}

// Build walks the root directory and builds the graph.
func (g *Graph) Build(root string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	fileListQueue := filesystem.StreamFiles(root)

	for f := range fileListQueue {
		if isSourceFile(f.Filename) {
			g.processFile(f.Location)
		}
	}

	return nil
}

// Update re-parses a specific file and updates the graph.
func (g *Graph) Update(path string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if isSourceFile(filepath.Base(path)) {
		// Clear old dependencies for this file from Reverse map
		if oldDeps, ok := g.Forward[path]; ok {
			for _, dep := range oldDeps {
				g.removeReverseDependency(dep, path)
			}
		}

		g.processFile(path)

		// Check if this new/updated file resolves any pending imports
		// We check if 'path' (without extension) matches any pending import paths.
		pathNoExt := strings.TrimSuffix(path, filepath.Ext(path))

		// Also handle index files (e.g. /utils/index.ts -> /utils)
		if strings.HasSuffix(pathNoExt, "/index") {
			pathNoExt = filepath.Dir(pathNoExt)
		}

		// We need to check exact matches or matches with extensions.
		// The pending import path is the absolute path WITHOUT extension (from resolvePaths).

		// Iterate over PendingImports (which are AbsPathPrefix -> [Dependents])
		// This is inefficient if PendingImports is huge, but fine for now.
		// Create a slice of keys to iterate over, as we might modify the map
		pendingImportPathsToResolve := make([]string, 0, len(g.PendingImports))
		for impPath := range g.PendingImports {
			pendingImportPathsToResolve = append(pendingImportPathsToResolve, impPath)
		}

		for _, importPath := range pendingImportPathsToResolve {
			// Ensure the importPath still exists in PendingImports, as it might have been resolved by a previous iteration
			dependents, ok := g.PendingImports[importPath]
			if !ok {
				continue
			}

			// Check if the newly created file 'path' satisfies 'importPath'
			// importPath is like /abs/path/to/utils
			// path is like /abs/path/to/utils.ts

			// Check if path starts with importPath and has a valid extension
			if strings.HasPrefix(path, importPath) {
				// Verify extension
				rest := strings.TrimPrefix(path, importPath)
				validExt := false
				for _, ext := range []string{".ts", ".js", ".tsx", ".jsx", "/index.ts", "/index.js", "/index.tsx", "/index.jsx"} {
					if rest == ext || (rest == "" && ext == "") { // The (rest == "" && ext == "") part is for cases like /utils -> /utils/index.ts
						validExt = true
						break
					}
				}

				if validExt {
					// It's a match! Link them.
					for _, dep := range dependents {
						g.addReverseDependency(path, dep)
						// Add to Forward map of the dependent
						// We need to ensure no duplicates and replace the unresolved path with the resolved one
						found := false
						for _, existingDep := range g.Forward[dep] {
							// This logic assumes that the 'importPath' stored in PendingImports
							// is the same string that was originally added to g.Forward[dep]
							// as an unresolved dependency. This might need refinement depending
							// on how ParseImports handles unresolved paths in its output.
							// For now, we'll just append the new resolved path.
							// A more robust solution would be to store the original import string
							// in PendingImports and use that to find and replace in g.Forward[dep].
							// For simplicity, we'll just append and rely on subsequent cleanup or
							// that the parser doesn't add unresolved paths to g.Forward[dep] directly.
							// Let's assume g.Forward[dep] only contains resolved paths.
							// If it contains unresolved paths, we need to remove the old one.
							// The current processFile only adds resolved paths to g.Forward.
							// So, we just append the newly resolved path.
							if existingDep == path { // Check if already added
								found = true
								break
							}
						}
						if !found {
							g.Forward[dep] = append(g.Forward[dep], path)
						}
					}
					// Remove from pending
					delete(g.PendingImports, importPath)
				}
			}
		}
	}
}

// GetDependents returns a list of all files that depend on the given path (transitively).
func (g *Graph) GetDependents(path string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	var dependents []string

	// Queue for BFS
	queue := []string{path}
	visited[path] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Find files that import 'current'
		if deps, ok := g.Reverse[current]; ok {
			for _, dep := range deps {
				if !visited[dep] {
					visited[dep] = true
					dependents = append(dependents, dep)
					queue = append(queue, dep)
				}
			}
		}
	}

	return dependents
}

// Internal helpers

func (g *Graph) processFile(path string) {
	result, err := g.parser.ParseImports(path)
	if err != nil {
		return // Ignore errors for now
	}

	g.Forward[path] = result.Resolved
	for _, imp := range result.Resolved {
		g.addReverseDependency(imp, path)
	}

	// Add unresolved to PendingImports
	for _, unresolved := range result.Unresolved {
		g.addPendingImport(unresolved.Path, path)
	}
}

func (g *Graph) addPendingImport(importPath, dependent string) {
	if _, ok := g.PendingImports[importPath]; !ok {
		g.PendingImports[importPath] = []string{}
	}
	// Check for duplicates
	for _, d := range g.PendingImports[importPath] {
		if d == dependent {
			return
		}
	}
	g.PendingImports[importPath] = append(g.PendingImports[importPath], dependent)
}

func (g *Graph) addReverseDependency(dependency, dependent string) {
	if _, ok := g.Reverse[dependency]; !ok {
		g.Reverse[dependency] = []string{}
	}
	// Check for duplicates
	for _, d := range g.Reverse[dependency] {
		if d == dependent {
			return
		}
	}
	g.Reverse[dependency] = append(g.Reverse[dependency], dependent)
}

func (g *Graph) removeReverseDependency(dependency, dependent string) {
	if deps, ok := g.Reverse[dependency]; ok {
		for i, d := range deps {
			if d == dependent {
				g.Reverse[dependency] = append(deps[:i], deps[i+1:]...)
				return
			}
		}
	}
}

func isSourceFile(name string) bool {
	exts := []string{".ts", ".js", ".tsx", ".jsx"}
	for _, ext := range exts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}
