package record

import (
	"bytes"
	"testing"
)

func TestRecordRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		rec  Record
		want []byte // [1 byte type][value]
	}{
		{"value", Record{Type: RecordValue, Value: []byte("hello")}, []byte{0x00, 'h', 'e', 'l', 'l', 'o'}},
		{"tombstone", Record{Type: RecordTombstone}, []byte{0x01}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			payload := c.rec.ToBytes()
			if !bytes.Equal(payload, c.want) {
				t.Errorf("ToBytes: got %v, want %v", payload, c.want)
			}

			got, err := Parse(payload)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if got.Type != c.rec.Type {
				t.Errorf("type: got %v, want %v", got.Type, c.rec.Type)
			}
			if !bytes.Equal(got.Value, c.rec.Value) {
				t.Errorf("value: got %v, want %v", got.Value, c.rec.Value)
			}
		})
	}
}
