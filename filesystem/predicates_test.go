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

func TestIsSourceFileMjsCjs(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"mjs file", "foo.mjs", true},
		{"cjs file", "foo.cjs", true},
		{"ts file", "foo.ts", true},
		{"md file", "README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSourceFile(tt.filename); got != tt.want {
				t.Errorf("IsSourceFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsTestFileMjsCjs(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"mjs test", "add.test.mjs", true},
		{"mjs spec", "add.spec.mjs", true},
		{"cjs test", "add.test.cjs", true},
		{"cjs spec", "add.spec.cjs", true},
		{"plain mjs", "add.mjs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTestFile(tt.filename); got != tt.want {
				t.Errorf("IsTestFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsTestFileByPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		// Suffix-based (existing behaviour preserved)
		{"spec suffix", "/project/src/math.spec.ts", true},
		{"test suffix", "/project/src/math.test.js", true},
		// Directory-based — test/
		{"test dir js", "/project/test/math.js", true},
		{"test dir mjs", "/project/test/add.mjs", true},
		{"test dir nested", "/project/test/unit/add.js", true},
		// Directory-based — tests/
		{"tests dir ts", "/project/tests/util.ts", true},
		// Directory-based — __tests__/
		{"__tests__ dir tsx", "/project/src/__tests__/comp.tsx", true},
		// Helper exclusion
		{"setup helper", "/project/test/setup.js", false},
		{"helpers helper", "/project/test/helpers.ts", false},
		{"underscore helper", "/project/test/_mocks.js", false},
		{"fixtures subdir", "/project/test/fixtures/data.js", false},
		{"mocks subdir", "/project/test/mocks/api.js", false},
		// Compound helper names (substring match)
		{"setupTest compound", "/project/test/setupTest.ts", false},
		{"testHelper compound", "/project/test/testHelper.tsx", false},
		{"mockFactory compound", "/project/test/mockFactory.ts", false},
		// Non-source files inside test dir
		{"json in test dir", "/project/test/config.json", false},
		// Normal source files outside test dir
		{"normal source", "/project/src/math.ts", false},
		{"readme", "/project/test/README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTestFileByPath(tt.path); got != tt.want {
				t.Errorf("IsTestFileByPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
