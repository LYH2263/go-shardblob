package hashx

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseHexRoundtrip(t *testing.T) {
	id := SumSHA256([]byte("hello"))
	got, err := ParseHex(id.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(id) {
		t.Fatalf("got %s want %s", got, id)
	}
}

func TestParseHexBad(t *testing.T) {
	if _, err := ParseHex("zz"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := ParseHex(strings.Repeat("0", 63)); err == nil {
		t.Fatal("expected length error")
	}
}

func TestFromSlice(t *testing.T) {
	id := SumSHA256([]byte("x"))
	got, err := FromSlice(id[:])
	if err != nil {
		t.Fatal(err)
	}
	if Compare(id, got) != 0 {
		t.Fatal("mismatch")
	}
	if _, err := FromSlice([]byte{1, 2}); err == nil {
		t.Fatal("expected error")
	}
}

func TestTaggedDiffers(t *testing.T) {
	a := Default()
	data := []byte("same")
	c := ChunkID(a, data)
	o := ObjectID(a, data)
	if c.Equal(o) {
		t.Fatal("chunk and object ids should be domain-separated")
	}
}

func TestSinkCopy(t *testing.T) {
	s := NewSink(Default())
	n, err := s.Copy(bytes.NewReader([]byte("abcdef")))
	if err != nil || n != 6 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if s.Bytes() != 6 {
		t.Fatal(s.Bytes())
	}
	id, n2, err := SumReader(Default(), bytes.NewReader([]byte("abcdef")))
	if err != nil || n2 != 6 || !EqualID(id, s.ID()) {
		t.Fatalf("sumreader mismatch")
	}
}

func TestLookupAndRegister(t *testing.T) {
	if _, err := Lookup("sha256"); err != nil {
		t.Fatal(err)
	}
	if _, err := Lookup("nope"); err == nil {
		t.Fatal("expected unknown")
	}
	if !Less(Zero, SumSHA256([]byte{1})) {
		t.Fatal("zero should be less")
	}
	_ = Names()
}
