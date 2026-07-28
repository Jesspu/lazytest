# Spec Plan: Parser Support for Dynamic Imports, Re-exports, and TS Path Aliases

## Problem Statement

The current import parser (`analysis/parser.go`) uses basic regular expressions:
- `importFromRegex`: `import ... from '...'`
- `importSideEffectRegex`: `import '...'`
- `requireRegex`: `require('...')`
- `jestMockRegex`: `jest.mock('...')`

This leaves several modern JavaScript/TypeScript module patterns undetected:
1. **Dynamic Imports**: `const mod = await import('./module')`
2. **Re-exports**: `export { foo } from './module'` and `export * from './module'`
3. **TypeScript Path Aliases**: `import { api } from '@/services/api'` or `import { db } from '~/db'` (currently ignored because `resolvePaths` skips non-relative paths not starting with `.`)

Unparsed import relations lead to missing edges in the dependency graph, resulting in missed test runs when aliased or re-exported modules are modified.

## Goals

1. Expand `analysis/parser.go` regex matching to capture dynamic imports and re-export statements.
2. Add support for resolving TypeScript path aliases (`paths` in `tsconfig.json`).
3. Ensure resolved alias paths are normalized to absolute filesystem paths.

## Proposed Changes

### 1. Additional Regex Patterns (`analysis/parser.go`)

Add regexes for dynamic imports and re-exports:

```go
var (
    // existing regexes...

    // dynamic import: import('...')
    dynamicImportRegex = regexp.MustCompile(`import\s*\(\s*['"]([^'"]+)['"]\s*\)`)
    
    // re-export: export ... from '...'
    exportFromRegex = regexp.MustCompile(`export[\s\S]*?from\s+['"]([^'"]+)['"]`)
)
```

In `Parser.ParseImports`, execute these additional patterns against file content.

### 2. TSConfig Path Alias Resolver (`analysis/config_resolver.go`)

Create `TSConfig` reader to parse compiler options:

```go
type TSConfig struct {
    CompilerOptions struct {
        BaseUrl string              `json:"baseUrl"`
        Paths   map[string][]string `json:"paths"`
    } `json:"compilerOptions"`
}
```

Implement `ResolveAlias(importPath string, root string) (string, bool)`:
- Read `tsconfig.json` at execution root.
- Map patterns like `@/*` -> `./src/*`.
- Resolve `@/components/button` to `<root>/src/components/button`.

### 3. Update `resolvePaths` (`analysis/parser.go`)

Integrate alias resolution into `resolvePaths`:

```go
for _, imp := range imports {
    var absPath string
    if strings.HasPrefix(imp, ".") {
        absPath = filepath.Join(dir, imp)
    } else if resolvedAlias, ok := p.resolveAlias(imp); ok {
        absPath = resolvedAlias
    } else {
        continue // node_modules or unresolvable
    }
    
    // Continue with findFile matching...
}
```

## Verification Plan

1. **Unit Tests (`analysis/analysis_test.go`)**:
   - Test dynamic import `await import('./utils')`.
   - Test re-exports `export * from './helper'`.
   - Test tsconfig alias resolution (`@/utils` resolving to `src/utils.ts`).
2. **Integration Test**:
   - Verify graph correctly links a test file importing via `@/...` alias to its target source file.
