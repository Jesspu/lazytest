package ui

import "github.com/jesspatton/lazytest/filesystem"

// flattenNodes performs a depth-first traversal to create a flat list of nodes.
// It merges single-child directories to reduce vertical space.
func flattenNodes(tree *filesystem.Node) []DisplayNode {
	nodes := []DisplayNode{}
	if tree == nil {
		return nodes
	}

	// Helper to get the compacted name and the final node
	var getCompacted func(*filesystem.Node, string) (*filesystem.Node, string)
	getCompacted = func(n *filesystem.Node, currentName string) (*filesystem.Node, string) {
		if n.IsDir && len(n.Children) == 1 && n.Children[0].IsDir {
			child := n.Children[0]
			return getCompacted(child, currentName+"/"+child.Name)
		}
		return n, currentName
	}

	// Depth-first traversal
	var traverse func(*filesystem.Node, int)
	traverse = func(n *filesystem.Node, depth int) {
		// Don't add root itself if it's just "."
		if n != tree {
			finalNode, displayName := getCompacted(n, n.Name)

			nodes = append(nodes, DisplayNode{
				Node:        finalNode,
				DisplayName: displayName,
				Depth:       depth,
			})

			// Now we need to continue traversal from the finalNode's children
			for _, child := range finalNode.Children {
				traverse(child, depth+1)
			}
		} else {
			// For root, we just traverse children
			for _, child := range n.Children {
				traverse(child, depth)
			}
		}
	}
	traverse(tree, 0)
	return nodes
}
