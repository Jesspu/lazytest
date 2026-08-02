package filesystem

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Node represents a file or directory in the test tree
type Node struct {
	Name     string
	Path     string
	IsDir       bool
	Children    []*Node
	ChildrenMap map[string]*Node
	Parent      *Node
}

// NodeFromPath constructs a lightweight Node from a raw absolute path.
// The Name is set to the base filename (the segment after the last separator).
func NodeFromPath(path string) *Node {
	return &Node{
		Path:        path,
		Name:        path[strings.LastIndex(path, string(os.PathSeparator))+1:],
		ChildrenMap: make(map[string]*Node),
	}
}

// Walk traverses the root directory and builds a tree of test files
func Walk(root string, excludes []string) (*Node, error) {
	rootNode := &Node{
		Name:        filepath.Base(root),
		Path:        root,
		IsDir:       true,
		ChildrenMap: make(map[string]*Node),
	}

	fileListQueue := StreamFiles(root)

	for f := range fileListQueue {
		if shouldExclude(f.Location, root, excludes) {
			continue
		}

		if IsTestFileByPath(f.Location) {
			addPathToTree(rootNode, f.Location, root)
		}
	}

	rootNode.Sort()
	return rootNode, nil
}

// Sort recursively sorts the children of this node.
// Directories come first, then files. Both are sorted alphabetically.
func (n *Node) Sort() {
	sort.Slice(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if a.IsDir && !b.IsDir {
			return true
		}
		if !a.IsDir && b.IsDir {
			return false
		}
		return a.Name < b.Name
	})
	for _, child := range n.Children {
		child.Sort()
	}
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
			if _, exists := currentNode.ChildrenMap[part]; !exists {
				child := &Node{
					Name:        part,
					Path:        path,
					IsDir:       false,
					Parent:      currentNode,
					ChildrenMap: make(map[string]*Node),
				}
				currentNode.Children = append(currentNode.Children, child)
				currentNode.ChildrenMap[part] = child
			}
			return
		}

		// Check if directory node already exists in map
		if child, exists := currentNode.ChildrenMap[part]; exists && child.IsDir {
			currentNode = child
		} else {
			// If not found, create it
			dirPath := filepath.Join(currentNode.Path, part)
			newNode := &Node{
				Name:        part,
				Path:        dirPath,
				IsDir:       true,
				Parent:      currentNode,
				ChildrenMap: make(map[string]*Node),
			}
			currentNode.Children = append(currentNode.Children, newNode)
			currentNode.ChildrenMap[part] = newNode
			currentNode = newNode
		}
	}
}

// AddNode adds a path incrementally to the tree and sorts the affected branch
func (n *Node) AddNode(path string) {
	// Re-use addPathToTree but with the current node as the root.
	// Since we don't have the exact rootPath, we can just use the node's Path as rootPath.
	if !strings.HasPrefix(path, n.Path) {
		return
	}
	addPathToTree(n, path, n.Path)
	// We might need to sort from the parent of the newly added node up to the root, or just re-sort the whole tree for simplicity, or just the current node. 
	// The problem statement says: "When fsnotify reports a file change (Create, Remove, Rename), the engine should mutate the specific branch of the in-memory Node tree."
	// Let's sort the tree after adding. Sorting the whole tree is fast enough if we don't do full traversal.
	// We can just sort the branch. For simplicity, just sort the modified node's children in addPathToTree.
	n.Sort()
}

// RemoveNode removes a path incrementally from the tree
func (n *Node) RemoveNode(path string) {
	if !strings.HasPrefix(path, n.Path) {
		return
	}
	
	relPath, err := filepath.Rel(n.Path, path)
	if err != nil || relPath == "." {
		return
	}

	parts := strings.Split(relPath, string(os.PathSeparator))
	currentNode := n

	for i, part := range parts {
		if child, exists := currentNode.ChildrenMap[part]; exists {
			if i == len(parts)-1 {
				// Remove the child
				delete(currentNode.ChildrenMap, part)
				// Remove from Children slice
				for j, c := range currentNode.Children {
					if c.Name == part {
						currentNode.Children = append(currentNode.Children[:j], currentNode.Children[j+1:]...)
						break
					}
				}
				// Clean up empty directories upwards (optional, but good for completeness)
				// Not strictly necessary for functionality.
				return
			}
			currentNode = child
		} else {
			break // Path not found
		}
	}
}
