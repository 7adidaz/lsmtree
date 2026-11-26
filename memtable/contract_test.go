package memtable

import (
	"bytes"
	"testing"

	"main/keys"
)

// Every implementation must pass the same contract tests.
func implementations() map[string]func() MemTableImplementation {
	return map[string]func() MemTableImplementation{
		"avl":      func() MemTableImplementation { return NewAVLTree() },
		"skiplist": func() MemTableImplementation { return NewSkipList() },
	}
}

func TestMemTableContract(t *testing.T) {
	for name, newImpl := range implementations() {
		t.Run(name, func(t *testing.T) {
			m := newImpl()

			if m.Size() != 0 || m.Len() != 0 {
				t.Errorf("Expected initial size and len to be 0, got %d and %d", m.Size(), m.Len())
			}

			testKeys := []uint32{50, 30, 70, 20, 40, 60, 80, 15, 25, 35, 45, 55, 65, 75, 85}
			for _, key := range testKeys {
				m.Put(keys.NewIntKey(key), []byte{byte(key)})
			}

			// Each entry is a 5-byte encoded IntKey plus a 1-byte value.
			const entrySize = 6

			if m.Size() != uint32(len(testKeys)*entrySize) {
				t.Errorf("Expected size %d after insertions, got %d", len(testKeys)*entrySize, m.Size())
			}
			if m.Len() != len(testKeys) {
				t.Errorf("Expected len %d after insertions, got %d", len(testKeys), m.Len())
			}

			for _, key := range testKeys {
				found, value := m.Get(keys.NewIntKey(key))
				if !found && value == nil {
					t.Errorf("Key %d should be present", key)
				} else if len(value) != 1 || value[0] != byte(key) {
					t.Errorf("Value for key %d is incorrect", key)
				}
			}

			nonExistentKeys := []uint32{10, 90, 42, 58}
			for _, key := range nonExistentKeys {
				found, value := m.Get(keys.NewIntKey(key))
				if found && value != nil {
					t.Errorf("Key %d should not be present", key)
				}
			}

			newKeys := []uint32{42, 58, 90, 5}
			for _, key := range newKeys {
				m.Put(keys.NewIntKey(key), []byte{byte(key)})
			}

			expectedSize := uint32((len(testKeys) + len(newKeys)) * entrySize)
			if m.Size() != expectedSize {
				t.Errorf("Expected size %d after new insertions, got %d",
					expectedSize, m.Size())
			}

			updateKeys := []uint32{50, 40, 80}
			for _, key := range updateKeys {
				m.Put(keys.NewIntKey(key), []byte{byte(key + 100)})
				found, value := m.Get(keys.NewIntKey(key))
				if !found && value == nil || len(value) != 1 || value[0] != byte(key+100) {
					t.Errorf("Updated value for key %d is incorrect", key)
				}
			}

			// Size should not change when updates keep the same value length
			if m.Size() != expectedSize {
				t.Errorf("Expected size to remain %d after updates, got %d",
					expectedSize, m.Size())
			}

			kvs := m.ToKVs()
			if len(kvs) != len(testKeys)+len(newKeys) {
				t.Fatalf("ToKVs returned %d entries, want %d", len(kvs), len(testKeys)+len(newKeys))
			}
			for i := 1; i < len(kvs); i++ {
				if kvs[i-1].Key.Compare(kvs[i].Key) != -1 {
					t.Errorf("ToKVs out of order at index %d", i)
				}
			}

			m.Clear()

			if m.Size() != 0 || m.Len() != 0 {
				t.Errorf("Expected size and len 0 after Clear")
			}
			if len(m.ToKVs()) != 0 {
				t.Errorf("Expected 0 elements after Clear")
			}

			m.Put(keys.NewIntKey(1), []byte{1})
			if found, _ := m.Get(keys.NewIntKey(1)); !found {
				t.Errorf("Expected inserts to work after Clear")
			}
		})
	}
}

func TestFloorInGaps(t *testing.T) {
	cases := []struct {
		key  uint32
		want []byte
	}{
		{5, nil},
		{10, []byte{10}},
		{15, []byte{10}},
		{20, []byte{20}},
		{30, []byte{30}},
		{35, []byte{30}},
	}
	for name, newImpl := range implementations() {
		t.Run(name, func(t *testing.T) {
			m := newImpl()
			for _, key := range []uint32{10, 20, 30} {
				m.Put(keys.NewIntKey(key), []byte{byte(key)})
			}
			for _, c := range cases {
				got := m.Floor(keys.NewIntKey(c.key))
				if !bytes.Equal(got, c.want) {
					t.Errorf("Floor(%d) = %v, want %v", c.key, got, c.want)
				}
			}
		})
	}
}

func TestHigherSkipsExactMatch(t *testing.T) {
	cases := []struct {
		key  uint32
		want []byte
	}{
		{5, []byte{10}},
		{10, []byte{20}},
		{15, []byte{20}},
		{20, []byte{30}},
		{30, nil},
		{35, nil},
	}
	for name, newImpl := range implementations() {
		t.Run(name, func(t *testing.T) {
			m := newImpl()
			for _, key := range []uint32{10, 20, 30} {
				m.Put(keys.NewIntKey(key), []byte{byte(key)})
			}
			for _, c := range cases {
				got := m.Higher(keys.NewIntKey(c.key))
				if !bytes.Equal(got, c.want) {
					t.Errorf("Higher(%d) = %v, want %v", c.key, got, c.want)
				}
			}
		})
	}
}
