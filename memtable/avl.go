package memtable

/*
 * This is a sorted in-memory data structure, When it gets
 * bigger than some threshold, it write it out to disk.
 *
 *  TODO: concurrency ... crowd goes: BOOOOO
 */

import (
	"bytes"
	"lsmtree/interfaces"
)

type Node struct {
	key    interfaces.Comparable
	data   []byte
	left   *Node
	right  *Node
	height int
}

func (n *Node) getKV() *Entry {
	return &Entry{n.key, n.data}
}

func NewNode(key interfaces.Comparable, data []byte) *Node {
	return &Node{
		key:    key,
		data:   data,
		height: 1,
	}
}

type AVLTree struct {
	head *Node
	size uint32
}

func NewAVLTree() *AVLTree {
	return &AVLTree{size: 0}
}

func (t *AVLTree) Size() uint32 {
	return t.size
}

func (t *AVLTree) Clear() {
	t.head = nil
	t.size = 0
}

func (t *AVLTree) Get(key interfaces.Comparable) []byte {
	var curr *Node = t.head
	for curr != nil {
		compareResult := curr.key.Compare(key)
		if compareResult == 0 {
			if curr.data != nil && bytes.Equal(curr.data, []byte{0x7f}) {
				return nil
			}
			return curr.data
		} else if compareResult == -1 {
			curr = curr.right
		} else {
			curr = curr.left
		}
	}
	return nil
}

func (t *AVLTree) Put(key interfaces.Comparable, val []byte) {
	// updated := new(bool)
	updated := false
	t.head = insert(t.head, key, val, &updated)
	if !updated {
		t.size++
	}
}

func (t *AVLTree) Delete(key interfaces.Comparable) {
	t.head = insert(t.head, key, []byte{0x7f}, nil)
}

func (t *AVLTree) Dump(log bool) []*Node {
	var arr []*Node
	inOrderTraversal(t.head, &arr)

	println(len(arr))
	if log {
		for i, node := range arr {
			println(
				"Node", i, ": K=", node.key.GetValue(),
				", D=", string(node.data),
				", H=", node.height,
				", BF=", getBalanceFactor(node),
			)
		}
	}

	return arr
}

func (t *AVLTree) ToKVs() []*Entry {
	var arr []*Node
	inOrderTraversal(t.head, &arr)
	result := make([]*Entry, len(arr))
	for i, node := range arr {
		result[i] = node.getKV()
	}

	return result
}

func inOrderTraversal(node *Node, result *[]*Node) {
	if node == nil {
		return
	}

	inOrderTraversal(node.left, result)
	*result = append(*result, node)
	inOrderTraversal(node.right, result)
}

/*
		 A                   B
		/ \       -->       / \
	   a   B               A   C
		  / \             / \
		 b   C           a   b
*/
func leftRotation(node *Node) *Node {
	b := node.right
	x := b.left
	b.left = node
	node.right = x

	node.height = 1 + max(getHight(node.left), getHight(node.right))
	b.height = 1 + max(getHight(node), getHight(b.right))

	return b
}

/*
		   C                 B
		  / \     -->       / \
		 B   c             A   C
	    / \                   / \
	   A   b                 b   c
*/
func rightRotation(node *Node) *Node {
	b := node.left
	x := b.right
	b.right = node
	node.left = x

	node.height = 1 + max(getHight(node.left), getHight(node.right))
	b.height = 1 + max(getHight(b.left), getHight(node))

	return b
}

func insert(node *Node, key interfaces.Comparable, data []byte, updated *bool) *Node {
	if node == nil {
		return NewNode(key, data)
	} else if node.key.Compare(key) == -1 {
		node.right = insert(node.right, key, data, updated)
	} else if node.key.Compare(key) == 1 {
		node.left = insert(node.left, key, data, updated)
	} else {
		node.data = data
		if updated != nil {
			*updated = true
		}
		return node
	}

	node.height = 1 + max(getHight(node.left), getHight(node.right))
	balanceFactor := getBalanceFactor(node)

	if balanceFactor > 1 && key.Compare(node.left.key) == -1 {
		return rightRotation(node)
	}

	if balanceFactor < -1 && key.Compare(node.right.key) == 1 {
		return leftRotation(node)
	}

	if balanceFactor > 1 && key.Compare(node.left.key) == 1 {
		node.left = leftRotation(node.left)
		return rightRotation(node)
	}

	if balanceFactor < -1 && key.Compare(node.right.key) == -1 {
		node.right = rightRotation(node.right)
		return leftRotation(node)
	}

	return node
}

func getHight(n *Node) int {
	if n != nil {
		return n.height
	}
	return 0
}

func getBalanceFactor(n *Node) int {
	if n == nil {
		return 0
	}

	return getHight(n.left) - getHight(n.right)
}
