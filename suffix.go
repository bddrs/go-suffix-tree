package suffix

import (
	"bytes"
	"sort"
)

// Return
// the first index of the mismatch byte (from right to left, starts from 1)
// len(left)+1 if left byte sequence is shorter than right one
// 0 if two byte sequences are equal
// -len(right)-1 if left byte sequence is longer than right one
func suffixDiff(left, right []byte) int {
	leftLen := len(left)
	rightLen := len(right)
	minLen := leftLen
	if minLen > rightLen {
		minLen = rightLen
	}
	for i := 1; i <= minLen; i++ {
		if left[leftLen-i] != right[rightLen-i] {
			return i
		}
	}
	if leftLen < rightLen {
		return leftLen + 1
	} else if leftLen == rightLen {
		return 0
	}
	return -rightLen - 1
}

type _Edge struct {
	label []byte
	// Could be either Node or Leaf
	point interface{}
}

type _Leaf struct {
	// For LongestSuffix and so on. We choice to use more memory(24 bytes per node)
	// over appending keys each time.
	originKey []byte
}

type _Node struct {
	edges []*_Edge
}

func (node *_Node) insertEdge(edge *_Edge) {
	newEdgeLabelLen := len(edge.label)
	idx := sort.Search(len(node.edges), func(i int) bool {
		return newEdgeLabelLen < len(node.edges[i].label)
	})
	node.edges = append(node.edges, nil)
	copy(node.edges[idx+1:], node.edges[idx:])
	node.edges[idx] = edge
}

func (node *_Node) removeEdge(idx int) {
	copy(node.edges[idx:], node.edges[idx+1:])
	node.edges[len(node.edges)-1] = nil
	node.edges = node.edges[:len(node.edges)-1]
}

// Reorder edge which is not shorter than before
func (node *_Node) backwardEdge(idx int) {
	edge := node.edges[idx]
	edgeLabelLen := len(edge.label)
	edgesLen := len(node.edges)
	if idx == edgesLen-1 {
		// Still longest, no need to change
		return
	}
	// Get the first edge which's label is longer than this edge...
	i := sort.Search(edgesLen-idx-1, func(j int) bool {
		return edgeLabelLen < len(node.edges[j+idx+1].label)
	})
	// ... and insert before it. (Note that we just add `idx` instead of `idx+1`)
	i += idx
	copy(node.edges[idx:i], node.edges[idx+1:i+1])
	node.edges[i] = edge
}

// Reorder edge which is shorter than before
func (node *_Node) forwardEdge(idx int) {
	edge := node.edges[idx]
	edgeLabelLen := len(edge.label)
	i := sort.Search(idx, func(j int) bool {
		return edgeLabelLen < len(node.edges[j].label)
	})
	copy(node.edges[i+1:idx+1], node.edges[i:idx])
	node.edges[i] = edge
}

func (node *_Node) insert(key []byte) {

	start := 0
	if len(node.edges) > 0 && len(node.edges[0].label) == 0 {
		// handle empty label as a special case, so the rest of labels don't share
		// common suffix
		if len(key) == 0 {
			return
		}
		start++
	}
	for i := start; i < len(node.edges); i++ {
		edge := node.edges[i]
		gap := suffixDiff(key, edge.label)
		if gap == 0 {
			// CASE 1: key == label
			switch point := edge.point.(type) {
			case *_Leaf:
				return
			case *_Node:
				// Node hitted, insert a leaf under this Node
				point.insert([]byte{})
				return
			}
		} else if gap < 0 {
			// CASE 2: key > label
			gap = -gap
			label := key[:len(key)-gap+1]
			switch point := edge.point.(type) {
			case *_Leaf:
				// Before: Node - "label" -> Leaf(Value1)
				// After: Node - "label" - Node - "" -> Leaf(Value1)
				//							|- "s" -> Leaf(Value2)
				// Create new Node, move old Leaf under new Node, and then
				//	insert a new Leaf
				newNode := &_Node{
					edges: []*_Edge{
						{
							label: []byte{},
							point: point,
						},
						{
							label: label,
							point: &_Leaf{},
						},
					},
				}
				edge.point = newNode
				return
			case *_Node:
				// Before: Node - "label" -> Node - "" -> Leaf(Value1)
				// After: Node - "label" - Node - "" -> Leaf(Value1)
				//							|- "s" -> Leaf(Value2)
				// Insert a new Leaf with extra data as label
				point.insert(label)
				return
			}
		} else if gap > 1 {
			// CASE 3: mismatch(key, label) after first letter or key < label
			// Before: Node - "labels" -> Node/Leaf(Value1)
			// After: Node - "label" - Node - "s" -> Node/Leaf(Value1)
			//						    |- "" -> Leaf(Value2)
			// Before: Node - "label" -> Node/Leaf(Value1)
			// After: Node - "lab" - Node - "el" -> Node/Leaf(Value1)
			//							|- "or" -> Leaf(Value2)
			newEdge := &_Edge{
				label: edge.label[:len(edge.label)-gap+1],
				point: edge.point,
			}
			keyEdge := &_Edge{
				label: key[:len(key)-gap+1],
				point: &_Leaf{},
			}
			newNode := &_Node{
				edges: make([]*_Edge, 2),
			}
			if len(newEdge.label) < len(keyEdge.label) {
				newNode.edges[0], newNode.edges[1] = newEdge, keyEdge
			} else {
				newNode.edges[0], newNode.edges[1] = keyEdge, newEdge
			}
			edge.point = newNode
			edge.label = edge.label[len(edge.label)-gap+1:]
			node.forwardEdge(i)
			return
		}
		// CASE 4: totally mismatch
	}

	leaf := &_Leaf{}
	edge := &_Edge{
		label: key,
		point: leaf,
	}
	node.insertEdge(edge)
	return
}

func (node *_Node) mergeChildNode(idx int, child *_Node) {
	if len(child.edges) == 1 {
		edge := node.edges[idx]
		edge.point = child.edges[0].point
		edge.label = append(child.edges[0].label, edge.label...)
		node.backwardEdge(idx)
	}
	// When child has only one edge, we will remove the child and merge its label,
	// So there is no case that child has no edge.
}

// Tree represents a suffix tree.
type Tree struct {
	root *_Node
}

// NewTree create a suffix tree for future usage.
func NewTree() *Tree {
	return &Tree{
		root: &_Node{
			edges: []*_Edge{},
		},
	}
}

func (tree *Tree) Insert(key []byte) bool {
	if key == nil {
		return false
	}
	tree.root.insert(key)
	return true
}

func (node *_Node) hasSequence(key []byte) bool {
	edges := node.edges
	start := 0
	if len(key) == 0 {
		return true
	}

	if len(edges[0].label) == 0 {
		// handle empty label as a special case, so the rest of labels don't share
		// common suffix
		if len(key) == 0 {
			return true
		}
		start++
	}

	keyLen := len(key)
	for i := start; i < len(edges); i++ {
		edge := edges[i]
		edgeLabelLen := len(edge.label)
		if keyLen > edgeLabelLen {
			if bytes.Equal(key[keyLen-edgeLabelLen:], edge.label) {
				subKey := key[:keyLen-edgeLabelLen]
				switch point := edge.point.(type) {
				case *_Leaf:
					return true
				case *_Node:
					found := point.hasSequence(subKey)
					if found {
						return true
					}
				}
			}
		} else if keyLen == edgeLabelLen {
			if bytes.Equal(key, edge.label) {
				switch point := edge.point.(type) {
				case *_Leaf:
					return true
				case *_Node:
					found := point.hasSequence([]byte{})
					if found {
						return true
					}
				}
			}
		} else if keyLen < edgeLabelLen {
			if bytes.Equal(key, edge.label[edgeLabelLen-keyLen:]) {
				return true
			}
		}
	}

	if start == 1 {
		return true
	}

	return false
}

func (tree *Tree) HasSequence(key []byte) bool {
	if key == nil || len(tree.root.edges) == 0 {
		return false
	}
	return tree.root.hasSequence(key)
}
