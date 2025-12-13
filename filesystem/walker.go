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
func Walk(root string, excludes []string) (*Node, error) {
	rootNode := &Node{
		Name:  filepath.Base(root),
		Path:  root,
		IsDir: true,
	}

	fileListQueue := StreamFiles(root)

	for f := range fileListQueue {
		if shouldExclude(f.Location, root, excludes) {
			continue
		}

		if IsTestFile(f.Filename) {
			addPathToTree(rootNode, f.Location, root)
		}
	}

	return rootNode, nil
}

func shouldExclude(path, root string, excludes []string) bool {
	if len(excludes) == 0 {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)

	for _, result := range excludes {
		// Exact match or subdirectory match
		// If exclude is "foo", matches "foo", "foo/bar"
		cleanResult := filepath.ToSlash(result)
		if rel == cleanResult || strings.HasPrefix(rel, cleanResult+"/") {
			return true
		}

		// Glob match
		matched, _ := filepath.Match(cleanResult, rel)
		if matched {
			return true
		}
	}
	return false
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
