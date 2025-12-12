package analysis

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/jesspatton/lazytest/filesystem"
)

// DependencyType represents the type of dependency.
type DependencyType int

const (
	DepRegular DependencyType = iota
	DepMocked
)

// Graph represents the dependency graph of the project.
type Graph struct {
	// Forward: File -> [Dependencies] -> Type
	Forward map[string]map[string]DependencyType
	// Reverse: File -> [Dependents] -> Type
	Reverse map[string]map[string]DependencyType
	// PendingImports: ImportPath -> [Dependents] -> Type
	PendingImports map[string]map[string]DependencyType

	parser *Parser
	mu     sync.RWMutex
}

// NewGraph creates a new dependency graph.
func NewGraph() *Graph {
	return &Graph{
		Forward:        make(map[string]map[string]DependencyType),
		Reverse:        make(map[string]map[string]DependencyType),
		PendingImports: make(map[string]map[string]DependencyType),
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
				if filesystem.IsSourceFile(f.Filename) {
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
	if !filesystem.IsSourceFile(filepath.Base(path)) {
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
	g.Forward[path] = make(map[string]DependencyType)
	for _, imp := range result.Resolved {
		depType := DepRegular
		if imp.Mocked {
			depType = DepMocked
		}
		g.Forward[path][imp.Path] = depType
		g.addReverseDependency(imp.Path, path, depType)
	}

	// Add unresolved to PendingImports
	for _, unresolved := range result.Unresolved {
		depType := DepRegular
		if unresolved.Mocked {
			depType = DepMocked
		}
		g.addPendingImport(unresolved.Path, path, depType)
	}

	// Check if this new/updated file resolves any pending imports.
	// The pending import path is the absolute path WITHOUT extension (from resolvePaths).
	// Instead of iterating over all pending imports (O(N)), we generate the possible keys
	// this file could satisfy and look them up directly (O(1)).

	candidates := []string{}

	// 1. Exact match (e.g. import "./foo.js" -> /path/to/foo.js)
	candidates = append(candidates, path)

	// 2. Strip extension (e.g. import "./foo" -> /path/to/foo)
	ext := filepath.Ext(path)
	if ext != "" {
		candidates = append(candidates, strings.TrimSuffix(path, ext))
	}

	// 3. Index files (e.g. import "./foo" -> /path/to/foo/index.ts -> /path/to/foo)
	// We check if the file is an index file and add the parent directory as a candidate.
	name := filepath.Base(path)
	nameNoExt := strings.TrimSuffix(name, ext)
	if nameNoExt == "index" {
		candidates = append(candidates, filepath.Dir(path))
	}

	for _, candidate := range candidates {
		if dependents, ok := g.PendingImports[candidate]; ok {
			// It's a match! Link them.
			for dep, depType := range dependents {
				g.addReverseDependency(path, dep, depType)
				// Add to Forward map of the dependent
				if g.Forward[dep] == nil {
					g.Forward[dep] = make(map[string]DependencyType)
				}
				g.Forward[dep][path] = depType
			}
			// Remove from pending
			delete(g.PendingImports, candidate)
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

// GetDependencyType returns the type of dependency between dependent and dependency.
// Returns DepRegular if not found (or default).
func (g *Graph) GetDependencyType(dependent, dependency string) DependencyType {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if deps, ok := g.Forward[dependent]; ok {
		if depType, ok := deps[dependency]; ok {
			return depType
		}
	}
	return DepRegular
}

// Internal helpers

func (g *Graph) addPendingImport(importPath, dependent string, depType DependencyType) {
	if _, ok := g.PendingImports[importPath]; !ok {
		g.PendingImports[importPath] = make(map[string]DependencyType)
	}
	g.PendingImports[importPath][dependent] = depType
}

func (g *Graph) addReverseDependency(dependency, dependent string, depType DependencyType) {
	if _, ok := g.Reverse[dependency]; !ok {
		g.Reverse[dependency] = make(map[string]DependencyType)
	}
	g.Reverse[dependency][dependent] = depType
}

func (g *Graph) removeReverseDependency(dependency, dependent string) {
	if deps, ok := g.Reverse[dependency]; ok {
		delete(deps, dependent)
		if len(deps) == 0 {
			delete(g.Reverse, dependency)
		}
	}
}
