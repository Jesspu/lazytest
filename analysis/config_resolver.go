package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// tsConfig mirrors the subset of tsconfig.json we care about.
type tsConfig struct {
	CompilerOptions struct {
		BaseUrl string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

// configCache caches parsed tsconfig data per project root to avoid re-reading disk on
// every import. The cache is invalidated implicitly when the process restarts.
var configCache sync.Map // map[root string]*tsConfig

// loadTSConfig reads and parses tsconfig.json from root, with in-process caching.
// Returns nil if the file does not exist or cannot be parsed.
func loadTSConfig(root string) *tsConfig {
	if v, ok := configCache.Load(root); ok {
		if cfg, ok := v.(*tsConfig); ok {
			return cfg
		}
	}

	tscPath := filepath.Join(root, "tsconfig.json")
	data, err := os.ReadFile(tscPath)
	if err != nil {
		// File not present – cache nil so we don't retry every call.
		configCache.Store(root, (*tsConfig)(nil))
		return nil
	}

	var cfg tsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		configCache.Store(root, (*tsConfig)(nil))
		return nil
	}

	configCache.Store(root, &cfg)
	return &cfg
}

// InvalidateTSConfigCache removes the cached tsconfig for root.
// Call this when tsconfig.json is known to have changed.
func InvalidateTSConfigCache(root string) {
	configCache.Delete(root)
}

// ResolveAlias attempts to map importPath to an absolute filesystem path using
// the tsconfig.json compilerOptions.paths and baseUrl at root.
//
// Pattern matching supports the wildcard suffix `*`, e.g.:
//
//	"@/*" → ["./src/*"]  resolves  "@/utils"  →  "<root>/src/utils"
//
// Returns (resolvedAbsPath, true) on success, ("", false) if no alias matched.
func ResolveAlias(importPath string, root string) (string, bool) {
	cfg := loadTSConfig(root)
	if cfg == nil {
		return "", false
	}

	baseURL := cfg.CompilerOptions.BaseUrl
	paths := cfg.CompilerOptions.Paths
	if len(paths) == 0 {
		return "", false
	}

	// Try each pattern in order.
	for pattern, targets := range paths {
		if len(targets) == 0 {
			continue
		}

		remainder, matched := matchAliasPattern(pattern, importPath)
		if !matched {
			continue
		}

		// Use the first target (mirrors TypeScript compiler behaviour).
		target := targets[0]

		// Substitute wildcard in target with the remainder.
		var relative string
		if strings.HasSuffix(target, "/*") || strings.HasSuffix(target, "\\*") {
			base := target[:len(target)-2] // strip the trailing /*
			relative = filepath.Join(base, remainder)
		} else {
			relative = target
		}

		// Resolve relative to baseUrl (which itself is relative to root).
		var absPath string
		if baseURL != "" {
			absBase := filepath.Join(root, baseURL)
			absPath = filepath.Join(absBase, relative)
		} else {
			absPath = filepath.Join(root, relative)
		}

		return absPath, true
	}

	return "", false
}

// matchAliasPattern checks whether importPath matches a tsconfig path alias pattern.
//
// Supported forms:
//   - Exact:    "@/utils"   matches   "@/utils"                   (remainder = "")
//   - Wildcard: "@/*"       matches   "@/components/button"       (remainder = "components/button")
func matchAliasPattern(pattern, importPath string) (remainder string, ok bool) {
	if strings.HasSuffix(pattern, "/*") || strings.HasSuffix(pattern, "\\*") {
		// Wildcard pattern: strip trailing /* from pattern and check prefix.
		prefix := pattern[:len(pattern)-2]
		if strings.HasPrefix(importPath, prefix+"/") {
			return importPath[len(prefix)+1:], true
		}
		// Also allow exact match of the prefix (import "@" when pattern is "@/*").
		if importPath == prefix {
			return "", true
		}
		return "", false
	}

	// Exact pattern match.
	if importPath == pattern {
		return "", true
	}

	return "", false
}
