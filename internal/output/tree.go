package output

import (
	"fmt"
	"io"
	"strings"
)

type TreeNode struct {
	Label        string      `json:"-"`
	Ref          string      `json:"ref"`
	RelationType string      `json:"relationType,omitempty"`
	Kind         string      `json:"kind,omitempty"`
	Owner        string      `json:"owner,omitempty"`
	Lifecycle    string      `json:"lifecycle,omitempty"`
	Tier         string      `json:"tier,omitempty"`
	Children     []*TreeNode `json:"children,omitempty"`
}

func PrintTree(w io.Writer, root *TreeNode) {
	printTreeNode(w, root, "", true, true)
}

func printTreeNode(w io.Writer, node *TreeNode, prefix string, isLast bool, isRoot bool) {
	display := node.Label
	if display == "" {
		display = node.Ref
	}
	if isRoot {
		fmt.Fprintln(w, display)
	} else {
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		fmt.Fprintf(w, "%s%s%s\n", prefix, connector, display)
	}

	childPrefix := prefix
	if !isRoot {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range node.Children {
		printTreeNode(w, child, childPrefix, i == len(node.Children)-1, false)
	}
}

func TreeToJSON(root *TreeNode) map[string]any {
	result := map[string]any{"ref": root.Ref}
	if root.RelationType != "" {
		result["relationType"] = root.RelationType
	}
	if root.Kind != "" {
		result["kind"] = root.Kind
	}
	if root.Owner != "" {
		result["owner"] = root.Owner
	}
	if root.Lifecycle != "" {
		result["lifecycle"] = root.Lifecycle
	}
	if root.Tier != "" {
		result["tier"] = root.Tier
	}
	if len(root.Children) > 0 {
		children := make([]map[string]any, 0, len(root.Children))
		for _, child := range root.Children {
			children = append(children, TreeToJSON(child))
		}
		result["children"] = children
	}
	return result
}

func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return strings.TrimSpace(string(runes[:max])) + "..."
}
