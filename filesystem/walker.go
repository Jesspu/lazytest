package filesystem

import (
	"os"
	"path/filepath"
	"strings"
)

// Node represents a file or directory in the test tree
type Node struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*Node
	Parent   *Node
}

// Walk traverses the root directory and builds a tree of test files
func Walk(root string) (*Node, error) {
	rootNode := &Node{
		Name:  filepath.Base(root),
		Path:  root,
		IsDir: true,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip root itself in the walk callback to avoid infinite recursion if we were doing manual recursion,
		// but filepath.Walk handles it. We just need to handle children.
		if path == root {
			return nil
		}

		// Filter ignored directories
		if info.IsDir() {
			if shouldIgnore(info.Name()) {
				return filepath.SkipDir
			}
			// We don't add directories immediately; we add them when we find a file inside them
			// OR we can build the tree structure as we go.
			// For simplicity in this MVP, let's build the full tree of directories that contain tests.
			return nil
		}

		// Check for test files
		if isTestFile(info.Name()) {
			addPathToTree(rootNode, path, root)
		}

		return nil
	})

	return rootNode, err
}

func shouldIgnore(name string) bool {
	ignored := []string{"node_modules", ".git", "dist", "build", "coverage"}
	for _, i := range ignored {
		if name == i {
			return true
		}
	}
	return false
}

func isTestFile(name string) bool {
	return strings.HasSuffix(name, ".test.ts") ||
		strings.HasSuffix(name, ".test.js") ||
		strings.HasSuffix(name, ".spec.ts") ||
		strings.HasSuffix(name, ".spec.js")
}

// addPathToTree adds a file path to the tree, creating intermediate directory nodes as needed
func addPathToTree(root *Node, path string, rootPath string) {
	relPath, err := filepath.Rel(rootPath, path)
	if err != nil {
		return
	}

	parts := strings.Split(relPath, string(os.PathSeparator))
	currentNode := root

	for i, part := range parts {
		// If it's the last part, it's the file
		if i == len(parts)-1 {
			child := &Node{
				Name:   part,
				Path:   path,
				IsDir:  false,
				Parent: currentNode,
			}
			currentNode.Children = append(currentNode.Children, child)
			return
		}

		// Check if directory node already exists
		found := false
		for _, child := range currentNode.Children {
			if child.Name == part && child.IsDir {
				currentNode = child
				found = true
				break
			}
		}

		// If not found, create it
		if !found {
			dirPath := filepath.Join(currentNode.Path, part)
			newNode := &Node{
				Name:   part,
				Path:   dirPath,
				IsDir:  true,
				Parent: currentNode,
			}
			currentNode.Children = append(currentNode.Children, newNode)
			currentNode = newNode
		}
	}
}
