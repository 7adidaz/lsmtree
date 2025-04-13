package memtable

import "testing"

func TestSimpleTree(t *testing.T) {
	avlTree := new(AVLTree)

	avlTree.Put(NewIntKey(10), []byte("a"))
	avlTree.Put(NewIntKey(90), []byte("a"))
	avlTree.Put(NewIntKey(45), []byte("a"))

	// Dump and print the tree keys
	keysOnly := avlTree.Dump(false)
	t.Logf("Tree keys: %v", keysOnly)

	// Dump and print the tree with both keys and data
	withData := avlTree.Dump(true)
	t.Logf("Tree with data: %v", withData)

	if avlTree.head.key.Compare(NewIntKey(45)) != 0 {
		t.Errorf("Expected root key to be 45, got a %d value", avlTree.head.key.GetValue())
	}
}
