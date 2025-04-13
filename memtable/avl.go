package memtable

import (
	"fmt"
	"lsmtree/interfaces"
)

type Node struct {
	key    interfaces.Comparable
	data   []byte
	left   *Node
	right  *Node
	height int
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
}

func (t *AVLTree) Get(key interfaces.Comparable) []byte {
	var curr *Node = t.head
	for curr != nil {
		compareResult := curr.key.Compare(key)
		if compareResult == 0 {
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
	t.head = insert(t.head, key, val)
}

func (t AVLTree) Delete(key uint) {
}

func (t *AVLTree) Dump(withData bool) []any {
	var arr []any
	inOrderTraversal(t.head, &arr, withData)
	return arr
}

func inOrderTraversal(node *Node, result *[]any, withData bool) {
	if node == nil {
		return
	}

	// Traverse left subtree
	inOrderTraversal(node.left, result, withData)

	if withData {
		*result = append(*result, struct {
			Key  any
			Data []byte
		}{
			Key:  node.key.GetValue(),
			Data: node.data,
		})
	} else {
		*result = append(*result, node.key.GetValue())
	}

	inOrderTraversal(node.right, result, withData)
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
	var x *Node
	if b != nil && b.left != nil {
		x = b.left
		b.left = node
	}
	node.right = x

	node.height = 1 + max(getHight(node.left), getHight(node.right))
	if b != nil && b.right != nil {
		b.height = 1 + max(getHight(node), getHight(b.right))
	}

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
	var x *Node
	if b != nil && b.right != nil {
		x = b.right
		b.right = node
	}
	node.left = x

	node.height = 1 + max(getHight(node.left), getHight(node.right))
	if b != nil && b.left != nil {
		b.height = 1 + max(getHight(b.left), getHight(node))
	}

	return b
}

func insert(node *Node, key interfaces.Comparable, data []byte) *Node {
	if node == nil {
		println("new Node", fmt.Sprintf("%v", key.GetValue()))
		return NewNode(key, data)
	} else if node.key.Compare(key) == -1 {
		println(fmt.Sprintf("%v", key.GetValue()), "key is larger, going right")
		node.right = insert(node.right, key, data)
	} else {
		println(fmt.Sprintf("%v", key.GetValue()), "key is smaller, going left")
		node.left = insert(node.left, key, data)
	}

	node.height = 1 + max(getHight(node.left), getHight(node.right))
	balanceFactor := getBalanceFactor(node)
	/* TODO: debug this
	$ go test ./memtable
	new Node 10
	90 key is larger, going right
	new Node 90
	BF -1 false true false
	45 key is larger, going right
	45 key is smaller, going left
	new Node 45
	BF 1 false false true
	BF -2 false true false
	45 Right then Left Rotation
	--- FAIL: TestSimpleTree (0.00s)
		avl_test.go:14: Tree keys: []
		avl_test.go:18: Tree with data: []
	panic: runtime error: invalid memory address or nil pointer dereference [recovered]
			panic: runtime error: invalid memory address or nil pointer dereference
	[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x5060b2]

	*/
	println("BF", balanceFactor, node == nil, node.left == nil, node.right == nil)

	if balanceFactor > 1 && key.Compare(node.left.key) == -1 {
		println(fmt.Sprintf("%v", key.GetValue()), "Right Rotation")
		return rightRotation(node)
	}

	if balanceFactor < -1 && key.Compare(node.right.key) == 1 {
		println(fmt.Sprintf("%v", key.GetValue()), "Left Rotation")
		return leftRotation(node)
	}

	if balanceFactor > 1 && key.Compare(node.left.key) == 1 {
		println(fmt.Sprintf("%v", key.GetValue()), "Left then Right Rotation")
		node.left = leftRotation(node.left)
		return rightRotation(node)
	}

	if balanceFactor < -1 && key.Compare(node.right.key) == -1 {
		println(fmt.Sprintf("%v", key.GetValue()), "Right then Left Rotation")
		node.right = leftRotation(node.right)
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
