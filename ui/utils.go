package ui

import "github.com/jesspatton/lazytest/filesystem"

// flattenNodes performs a depth-first traversal to create a flat list of nodes.
func flattenNodes(tree *filesystem.Node) []*filesystem.Node {
	nodes := []*filesystem.Node{}
	if tree == nil {
		return nodes
	}
	// Depth-first traversal
	var traverse func(*filesystem.Node)
	traverse = func(n *filesystem.Node) {
		// Don't add root itself if it's just "."
		if n != tree {
			nodes = append(nodes, n)
		}
		for _, child := range n.Children {
			traverse(child)
		}
	}
	traverse(tree)
	return nodes
}
