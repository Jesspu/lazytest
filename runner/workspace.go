package runner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Workspace represents a single package within a monorepo.
type Workspace struct {
	Name   string // Package name from package.json (or directory name as fallback)
	Root   string // Absolute path to the package directory
	Config Config // Per-package resolved config
}

// DiscoverWorkspaces scans the project root for workspace definitions and
// returns a Workspace entry for each package. It supports:
//   - npm/yarn: "workspaces" array in root package.json
//   - pnpm: "packages" list in pnpm-workspace.yaml
//
// If no workspace definitions are found (single-package repo), an empty slice
// is returned and callers should fall back to root-level config.
func DiscoverWorkspaces(projectRoot string) []Workspace {
	globs := workspaceGlobs(projectRoot)
	if len(globs) == 0 {
		return nil
	}

	dirs := expandGlobs(projectRoot, globs)
	workspaces := make([]Workspace, 0, len(dirs))
	for _, dir := range dirs {
		ws := buildWorkspace(dir)
		workspaces = append(workspaces, ws)
	}
	return workspaces
}

// workspaceGlobs reads workspace glob patterns from either package.json
// (npm/yarn) or pnpm-workspace.yaml (pnpm) at the given root.
func workspaceGlobs(root string) []string {
	// 1. npm / yarn: package.json "workspaces"
	if globs := npmWorkspaceGlobs(root); len(globs) > 0 {
		return globs
	}
	// 2. pnpm: pnpm-workspace.yaml "packages"
	if globs := pnpmWorkspaceGlobs(root); len(globs) > 0 {
		return globs
	}
	return nil
}

// npmWorkspaceGlobs parses the "workspaces" field from package.json.
// It handles both the plain array form and the Yarn object form
// {"workspaces": {"packages": [...]}}.
func npmWorkspaceGlobs(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return nil
	}

	// Try plain array form first.
	var plain struct {
		Workspaces []string `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &plain); err == nil && len(plain.Workspaces) > 0 {
		return plain.Workspaces
	}

	// Try Yarn object form: {"workspaces": {"packages": [...]}}
	var yarn struct {
		Workspaces struct {
			Packages []string `json:"packages"`
		} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &yarn); err == nil && len(yarn.Workspaces.Packages) > 0 {
		return yarn.Workspaces.Packages
	}

	return nil
}

// pnpmWorkspaceGlobs parses the "packages:" list from pnpm-workspace.yaml
// using a minimal line-by-line parser — avoids a YAML dependency.
func pnpmWorkspaceGlobs(root string) []string {
	f, err := os.Open(filepath.Join(root, "pnpm-workspace.yaml"))
	if err != nil {
		return nil
	}
	defer f.Close()

	var globs []string
	inPackages := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "packages:" {
			inPackages = true
			continue
		}
		if inPackages {
			// A new top-level key ends the packages block.
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && line[0] != '-' {
				break
			}
			if strings.HasPrefix(trimmed, "- ") {
				val := strings.TrimPrefix(trimmed, "- ")
				// Strip inline YAML comments before unquoting.
				if idx := strings.Index(val, " #"); idx != -1 {
					val = strings.TrimSpace(val[:idx])
				}
				val = strings.Trim(val, "'\"")
				if val != "" {
					globs = append(globs, val)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil
	}
	return globs
}

// expandGlobs resolves workspace glob patterns (e.g. "packages/*") to a list
// of directories that contain a package.json. Only single-level wildcards are
// supported (as per the workspace spec — deep globs are not common in practice).
func expandGlobs(root string, globs []string) []string {
	seen := make(map[string]struct{})
	var dirs []string

	for _, pattern := range globs {
		// Resolve the glob relative to root.
		absPattern := filepath.Join(root, filepath.FromSlash(pattern))

		// filepath.Glob handles "*" but not "**". Split on "**" to keep
		// compatibility; for now only the first segment is expanded.
		matches, err := filepath.Glob(absPattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if _, err := os.Stat(filepath.Join(match, "package.json")); err != nil {
				continue // not a package directory
			}
			if _, dup := seen[match]; dup {
				continue
			}
			seen[match] = struct{}{}
			dirs = append(dirs, match)
		}
	}
	return dirs
}

// buildWorkspace constructs a Workspace for the given package directory.
func buildWorkspace(pkgDir string) Workspace {
	name := packageName(pkgDir)
	config := LoadConfig(pkgDir)
	return Workspace{
		Name:   name,
		Root:   pkgDir,
		Config: config,
	}
}

// packageName returns the "name" field from the package's package.json, or
// the directory basename if the file is missing or the field is empty.
func packageName(pkgDir string) string {
	data, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return filepath.Base(pkgDir)
	}
	var pkg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Name == "" {
		return filepath.Base(pkgDir)
	}
	return pkg.Name
}
