package filesystem

import "testing"

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"ts test", "foo.test.ts", true},
		{"js test", "foo.test.js", true},
		{"tsx test", "foo.test.tsx", true},
		{"spec ts", "foo.spec.ts", true},
		{"normal file", "foo.ts", false},
		{"readme", "README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTestFile(tt.filename); got != tt.want {
				t.Errorf("IsTestFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"ts file", "foo.ts", true},
		{"js file", "foo.js", true},
		{"tsx file", "foo.tsx", true},
		{"jsx file", "foo.jsx", true},
		{"test file", "foo.test.ts", true}, // Test files are also source files
		{"readme", "README.md", false},
		{"json", "package.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSourceFile(tt.filename); got != tt.want {
				t.Errorf("IsSourceFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		// Existing exact-match basenames
		{"package json", "package.json", true},
		{"tsconfig json", "tsconfig.json", true},
		{"lazytest json", ".lazytest.json", true},
		// New exact-match basenames
		{"pnpm workspace yaml", "pnpm-workspace.yaml", true},
		{"babelrc", ".babelrc", true},
		// Existing prefix patterns
		{"jest config js", "jest.config.js", true},
		{"jest config ts", "jest.config.ts", true},
		{"vite config ts", "vite.config.ts", true},
		{"vite config js", "vite.config.js", true},
		{"babel config js", "babel.config.js", true},
		{"webpack config js", "webpack.config.js", true},
		// New prefix patterns
		{"vitest config ts", "vitest.config.ts", true},
		{"vitest config js", "vitest.config.js", true},
		{"vitest workspace ts", "vitest.workspace.ts", true},
		{"vitest workspace js", "vitest.workspace.js", true},
		{"playwright config ts", "playwright.config.ts", true},
		{"playwright config js", "playwright.config.js", true},
		{"mocharc yml", ".mocharc.yml", true},
		{"mocharc json", ".mocharc.json", true},
		{"mocharc js", ".mocharc.js", true},
		// Full path — verify filepath.Base stripping
		{"full path jest config", "/project/root/jest.config.ts", true},
		// Non-config files
		{"source file", "foo.ts", false},
		{"readme", "README.md", false},
		{"random json", "data.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConfigFile(tt.filename); got != tt.want {
				t.Errorf("IsConfigFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
