package manifest

import (
	"bytes"
	"testing"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

func sample(t *testing.T) *Manifest {
	t.Helper()
	oid := hashx.ObjectID(hashx.Default(), []byte("abc"))
	c1 := hashx.ChunkID(hashx.Default(), []byte("ab"))
	c2 := hashx.ChunkID(hashx.Default(), []byte("c"))
	return New(hashx.NameSHA256, 2, false, oid, []Entry{
		{ID: c1, Size: 2, Offset: 0},
		{ID: c2, Size: 1, Offset: 2},
	})
}

func TestEncodeDecode(t *testing.T) {
	m := sample(t)
	raw, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalSize != 3 || got.ChunkCount() != 2 || got.LastSize() != 1 {
		t.Fatalf("%+v", got)
	}
	if !got.ObjectID.Equal(m.ObjectID) {
		t.Fatal("oid")
	}
	if got.CDC() {
		t.Fatal("cdc flag")
	}
}

func TestCRCDetect(t *testing.T) {
	raw, err := Encode(sample(t))
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 0xff
	if _, err := Decode(raw); err == nil {
		t.Fatal("expected crc error")
	}
}

func TestValidateGap(t *testing.T) {
	m := sample(t)
	m.Chunks[1].Offset = 9
	if err := Validate(m); err == nil {
		t.Fatal("expected gap error")
	}
}

func TestEmptyObject(t *testing.T) {
	oid := hashx.ObjectID(hashx.Default(), nil)
	m := New(hashx.NameSHA256, 64, false, oid, nil)
	raw, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalSize != 0 || got.ChunkCount() != 0 {
		t.Fatal(got.TotalSize, got.ChunkCount())
	}
	cl := got.Clone()
	if !bytes.Equal(cl.ObjectID[:], got.ObjectID[:]) {
		t.Fatal("clone")
	}
}
