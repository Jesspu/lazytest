package ui

import (
	"testing"

	"github.com/jesspatton/lazytest/filesystem"
)

func TestFlattenNodes_Basic(t *testing.T) {
	// root -> a(file), b(file)
	root := &filesystem.Node{
		Name:  ".",
		IsDir: true,
		Children: []*filesystem.Node{
			{Name: "a", IsDir: false},
			{Name: "b", IsDir: false},
		},
	}

	nodes := flattenNodes(root)

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].DisplayName != "a" {
		t.Errorf("Expected 'a', got '%s'", nodes[0].DisplayName)
	}
	if nodes[1].DisplayName != "b" {
		t.Errorf("Expected 'b', got '%s'", nodes[1].DisplayName)
	}
}

func TestFlattenNodes_Compaction(t *testing.T) {
	// root -> a(dir) -> b(dir) -> c(file)
	// Should compact to "a/b" and then "c"
	root := &filesystem.Node{
		Name:  ".",
		IsDir: true,
		Children: []*filesystem.Node{
			{
				Name:  "a",
				IsDir: true,
				Children: []*filesystem.Node{
					{
						Name:  "b",
						IsDir: true,
						Children: []*filesystem.Node{
							{Name: "c", IsDir: false},
						},
					},
				},
			},
		},
	}

	nodes := flattenNodes(root)

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}

	// Check compacted directory
	if nodes[0].DisplayName != "a/b" {
		t.Errorf("Expected compacted name 'a/b', got '%s'", nodes[0].DisplayName)
	}
	if !nodes[0].Node.IsDir {
		t.Error("Expected first node to be a directory")
	}

	// Check file
	if nodes[1].DisplayName != "c" {
		t.Errorf("Expected 'c', got '%s'", nodes[1].DisplayName)
	}
}

func TestFlattenNodes_Mixed(t *testing.T) {
	// root
	//  -> src(dir) -> main.go(file)
	//  -> pkg(dir) -> api(dir) -> handler.go(file)
	//
	// Should result in:
	// - src
	// - main.go
	// - pkg/api
	// - handler.go

	root := &filesystem.Node{
		Name:  ".",
		IsDir: true,
		Children: []*filesystem.Node{
			{
				Name:  "src",
				IsDir: true,
				Children: []*filesystem.Node{
					{Name: "main.go", IsDir: false},
				},
			},
			{
				Name:  "pkg",
				IsDir: true,
				Children: []*filesystem.Node{
					{
						Name:  "api",
						IsDir: true,
						Children: []*filesystem.Node{
							{Name: "handler.go", IsDir: false},
						},
					},
				},
			},
		},
	}

	nodes := flattenNodes(root)

	if len(nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(nodes))
	}

	expected := []string{"src", "main.go", "pkg/api", "handler.go"}
	for i, name := range expected {
		if nodes[i].DisplayName != name {
			t.Errorf("Index %d: expected '%s', got '%s'", i, name, nodes[i].DisplayName)
		}
	}
}

func TestFlattenNodes_Empty(t *testing.T) {
	if len(flattenNodes(nil)) != 0 {
		t.Error("Expected empty list for nil tree")
	}

	emptyRoot := &filesystem.Node{Name: ".", IsDir: true}
	if len(flattenNodes(emptyRoot)) != 0 {
		t.Error("Expected empty list for empty root")
	}
}
