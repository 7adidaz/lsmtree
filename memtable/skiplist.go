package memtable

import (
	"math/rand/v2"

	"main/interfaces"
)

const (
	maxLevel = 16
	// chance a node grows one more level (Pugh's recommendation).
	p = 0.25
)

/*
A skip list is a sorted linked list with express lanes: every entry lives on
level 1, and a node's random height decides how many lanes it joins. A search
starts at the head's highest lane, moves right while the next key is smaller,
and drops a lane when it would overshoot.

	level 4: H ---------------------------> 21 ----------------> nil
	level 3: H --------------> 9 ---------> 21 ----------------> nil
	level 2: H --> 3 --------> 9 ---------> 21 --> 25 ---------> nil
	level 1: H --> 3 --> 6 --> 9 --> 12 --> 21 --> 25 --> 26 --> nil
*/
type skipNode struct {
	key  interfaces.Comparable
	data []byte
	next []*skipNode
}

type SkipList struct {
	head  *skipNode
	level int
	size  uint32
	count int
}

func NewSkipList() *SkipList {
	return &SkipList{
		head:  &skipNode{next: make([]*skipNode, maxLevel)},
		level: 1,
	}
}

func (s *SkipList) randomLevel() int {
	lvl := 1
	for rand.IntN(int(1/p)) == 0 && lvl < maxLevel {
		lvl++
	}
	return lvl
}

// find returns key's predecessor at every level: prevNode[0].next[0] is the
// first node >= key. Unused levels point at head so inserts can grow the list.
func (s *SkipList) find(key interfaces.Comparable) [maxLevel]*skipNode {
	var prevNode [maxLevel]*skipNode
	for i := s.level; i < maxLevel; i++ {
		prevNode[i] = s.head
	}

	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		for curr.next[i] != nil && curr.next[i].key.Compare(key) < 0 {
			curr = curr.next[i]
		}
		prevNode[i] = curr
	}
	return prevNode
}

func (s *SkipList) Get(key interfaces.Comparable) (bool, []byte) {
	prevNode := s.find(key)
	curr := prevNode[0].next[0]
	if curr != nil && curr.key.Compare(key) == 0 {
		return true, curr.data
	}
	return false, nil
}

func (s *SkipList) Put(key interfaces.Comparable, val []byte) {
	prevNode := s.find(key)
	curr := prevNode[0].next[0]
	if curr != nil && curr.key.Compare(key) == 0 {
		s.size = s.size - uint32(len(curr.data)) + uint32(len(val))
		curr.data = val
		return
	}

	lvl := s.randomLevel()
	if lvl > s.level {
		s.level = lvl
	}

	sNode := &skipNode{key: key, data: val, next: make([]*skipNode, lvl)}

	for i := range lvl {
		sNode.next[i] = prevNode[i].next[i]
		prevNode[i].next[i] = sNode
	}
	s.size += uint32(key.Size() + len(val))
	s.count++
}

func (s *SkipList) Floor(key interfaces.Comparable) []byte {
	prevNode := s.find(key)
	if curr := prevNode[0].next[0]; curr != nil && curr.key.Compare(key) == 0 {
		return curr.data
	}
	if prevNode[0] == s.head {
		return nil
	}
	return prevNode[0].data
}

func (s *SkipList) Higher(key interfaces.Comparable) []byte {
	prevNode := s.find(key)
	curr := prevNode[0].next[0]
	if curr != nil && curr.key.Compare(key) == 0 {
		curr = curr.next[0]
	}
	if curr == nil {
		return nil
	}
	return curr.data
}

func (s *SkipList) Clear() {
	s.level = 1
	s.size = 0
	s.count = 0
	s.head = &skipNode{next: make([]*skipNode, maxLevel)}
}

func (s *SkipList) Size() uint32 {
	return s.size
}

func (s *SkipList) Len() int {
	return s.count
}

func (s *SkipList) ToKVs() []*Entry {
	result := make([]*Entry, 0, s.count)
	for curr := s.head.next[0]; curr != nil; curr = curr.next[0] {
		result = append(result, &Entry{curr.key, curr.data})
	}
	return result
}
