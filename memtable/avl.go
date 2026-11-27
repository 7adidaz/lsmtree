package memtable

import (
	"main/interfaces"
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
	head  *Node
	size  uint32
	count int
}

func NewAVLTree() *AVLTree {
	return &AVLTree{size: 0}
}

func (t *AVLTree) Size() uint32 {
	return t.size
}

func (t *AVLTree) Len() int {
	return t.count
}

func (t *AVLTree) Clear() {
	t.head = nil
	t.size = 0
	t.count = 0
}

func (t *AVLTree) Get(key interfaces.Comparable) (bool, []byte) {
	var curr *Node = t.head
	for curr != nil {
		compareResult := curr.key.Compare(key)
		switch compareResult {
		case 0:
			return true, curr.data
		case -1:
			curr = curr.right
		default:
			curr = curr.left
		}
	}
	return false, nil
}

// Put inserts or overwrites a key. Overwrites mutate in place, so insert
// reports the replaced value's length (-1 for a new key) to keep size right.
// TODO: once writes carry sequence numbers, every Put becomes a new node that
// shadows older versions, and oldValueLen goes away.
func (t *AVLTree) Put(key interfaces.Comparable, val []byte) {
	oldValueLen := -1
	t.head = insert(t.head, key, val, &oldValueLen)
	if oldValueLen >= 0 {
		t.size = t.size - uint32(oldValueLen) + uint32(len(val))
		return
	}
	t.size += uint32(key.Size() + len(val))
	t.count++
}

func (t *AVLTree) Floor(key interfaces.Comparable) []byte {
	curr := t.head
	var candidate *Node

	for curr != nil {
		compareResult := curr.key.Compare(key)
		switch compareResult {
		case 0:
			return curr.data
		case 1:
			curr = curr.left
		default:
			candidate = curr
			curr = curr.right
		}
	}

	if candidate != nil {
		return candidate.data
	}
	return nil
}

func (t *AVLTree) Higher(key interfaces.Comparable) []byte {
	curr := t.head
	var candidate *Node

	for curr != nil {
		compareResult := curr.key.Compare(key)
		switch compareResult {
		case 1:
			candidate = curr
			curr = curr.left
		default:
			curr = curr.right
		}
	}

	if candidate != nil {
		return candidate.data
	}
	return nil
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

func insert(node *Node, key interfaces.Comparable, data []byte, oldValueLen *int) *Node {
	if node == nil {
		return NewNode(key, data)
	} else if node.key.Compare(key) == -1 {
		node.right = insert(node.right, key, data, oldValueLen)
	} else if node.key.Compare(key) == 1 {
		node.left = insert(node.left, key, data, oldValueLen)
	} else {
		*oldValueLen = len(node.data)
		node.data = data
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
