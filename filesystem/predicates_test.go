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
