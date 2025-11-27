package ui

import "github.com/jesspatton/lazytest/filesystem"

// flattenNodes performs a depth-first traversal to create a flat list of nodes.
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
			// Check if we can compact this node
			// We only compact directories that have exactly one child which is also a directory
			// But wait, the traversal needs to handle this.

			// Actually, the better way is: when we visit a node, if it's a directory, check if it can be compacted.
			// If it can, we "skip" adding it as a separate line, but we need to know "where we are".

			// Let's retry the logic.
			// We visit 'n'.
			// If 'n' is a directory and has 1 child which is a dir, we merge it with the child.
			// But we are already visiting 'n'.

			// Let's change the approach.
			// We add 'n' to the list.
			// If 'n' was compacted, we display the compacted name.
			// Then we visit its children.

			// Wait, if we compact "apps" -> "admin" -> "test", we want ONE entry "apps/admin/test".
			// So when we visit "apps", we look ahead.

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
