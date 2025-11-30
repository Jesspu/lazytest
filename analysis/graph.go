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
	Forward map[string]map[string]bool
	// Reverse: File -> [Dependents] (Files that import this file)
	Reverse map[string]map[string]bool
	// PendingImports: ImportPath -> [Dependents] (Files that import a path that wasn't found yet)
	PendingImports map[string]map[string]bool

	parser *Parser
	mu     sync.RWMutex
}

// NewGraph creates a new dependency graph.
func NewGraph() *Graph {
	return &Graph{
		Forward:        make(map[string]map[string]bool),
		Reverse:        make(map[string]map[string]bool),
		PendingImports: make(map[string]map[string]bool),
		parser:         NewParser(),
	}
}

// Build walks the root directory and builds the graph.
func (g *Graph) Build(root string) error {
	fileListQueue := filesystem.StreamFiles(root)
	var wg sync.WaitGroup

	// Use a fixed number of workers for now, or could be runtime.NumCPU()
	numWorkers := 10

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range fileListQueue {
				if isSourceFile(f.Filename) {
					g.Update(f.Location)
				}
			}
		}()
	}

	wg.Wait()
	return nil
}

// Update re-parses a specific file and updates the graph.
func (g *Graph) Update(path string) {
	// Parse outside the lock
	if !isSourceFile(filepath.Base(path)) {
		return
	}

	result, err := g.parser.ParseImports(path)
	if err != nil {
		return // Ignore errors for now
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Clear old dependencies for this file from Reverse map
	if oldDeps, ok := g.Forward[path]; ok {
		for dep := range oldDeps {
			g.removeReverseDependency(dep, path)
		}
	}

	// Update Forward map
	g.Forward[path] = make(map[string]bool)
	for _, imp := range result.Resolved {
		g.Forward[path][imp] = true
		g.addReverseDependency(imp, path)
	}

	// Add unresolved to PendingImports
	for _, unresolved := range result.Unresolved {
		g.addPendingImport(unresolved.Path, path)
	}

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
				for dep := range dependents {
					g.addReverseDependency(path, dep)
					// Add to Forward map of the dependent
					if g.Forward[dep] == nil {
						g.Forward[dep] = make(map[string]bool)
					}
					g.Forward[dep][path] = true
				}
				// Remove from pending
				delete(g.PendingImports, importPath)
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
			for dep := range deps {
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
	// This method is now largely redundant as Update handles the logic,
	// but kept if needed for internal use, though Update is preferred.
	// For now, we'll remove it to avoid confusion and duplication since Update covers it.
}

func (g *Graph) addPendingImport(importPath, dependent string) {
	if _, ok := g.PendingImports[importPath]; !ok {
		g.PendingImports[importPath] = make(map[string]bool)
	}
	g.PendingImports[importPath][dependent] = true
}

func (g *Graph) addReverseDependency(dependency, dependent string) {
	if _, ok := g.Reverse[dependency]; !ok {
		g.Reverse[dependency] = make(map[string]bool)
	}
	g.Reverse[dependency][dependent] = true
}

func (g *Graph) removeReverseDependency(dependency, dependent string) {
	if deps, ok := g.Reverse[dependency]; ok {
		delete(deps, dependent)
		if len(deps) == 0 {
			delete(g.Reverse, dependency)
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
