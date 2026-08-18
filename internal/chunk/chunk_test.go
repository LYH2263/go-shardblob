package chunk

import (
	"bytes"
	"io"
	"testing"
)

func TestFixedSplitTail(t *testing.T) {
	pol := Normalize(Policy{Mode: ModeFixed, Avg: 8})
	data := []byte("hello world!!") // 13 bytes -> 8 + 5
	pieces, err := SplitAll(bytes.NewReader(data), pol)
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 2 {
		t.Fatalf("pieces=%d", len(pieces))
	}
	if len(pieces[0].Data) != 8 || len(pieces[1].Data) != 5 {
		t.Fatalf("sizes %d %d", len(pieces[0].Data), len(pieces[1].Data))
	}
	got, err := AssembleBytes(pieces)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("assemble mismatch")
	}
	m := MeasurePieces(pieces, pol)
	if !m.ShortTail || m.LastSize != 5 {
		t.Fatalf("measure %+v", m)
	}
}

func TestFixedEmpty(t *testing.T) {
	pol := DefaultPolicy()
	pieces, err := SplitAll(bytes.NewReader(nil), pol)
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 0 {
		t.Fatalf("empty should have 0 pieces, got %d", len(pieces))
	}
}

func TestFixedExactMultiple(t *testing.T) {
	pol := Normalize(Policy{Mode: ModeFixed, Avg: 4})
	data := []byte("abcdabcd")
	pieces, err := SplitAll(bytes.NewReader(data), pol)
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 2 {
		t.Fatalf("got %d", len(pieces))
	}
	if pieces[1].Offset != 4 {
		t.Fatal(pieces[1].Offset)
	}
}

func TestCDCRoundtrip(t *testing.T) {
	pol := CDCPolicy(256)
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. "), 50)
	pieces, err := SplitAll(bytes.NewReader(data), pol)
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) == 0 {
		t.Fatal("expected some pieces")
	}
	got, err := AssembleBytes(pieces)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("cdc assemble mismatch")
	}
	var off uint64
	for i, p := range pieces {
		if p.Offset != off || p.Index != i {
			t.Fatalf("piece %d off=%d want %d", i, p.Offset, off)
		}
		if len(p.Data) > pol.Max {
			t.Fatalf("piece %d len %d > max %d", i, len(p.Data), pol.Max)
		}
		off += uint64(len(p.Data))
	}
}

func TestAssembleWriter(t *testing.T) {
	pol := Normalize(Policy{Mode: ModeFixed, Avg: 3})
	src := []byte("1234567")
	pieces, err := SplitAll(bytes.NewReader(src), pol)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := NewWriter(&buf)
	for _, p := range pieces {
		if err := w.Append(p); err != nil {
			t.Fatal(err)
		}
	}
	if buf.String() != string(src) || w.Count() != len(pieces) {
		t.Fatal(buf.String(), w.Count())
	}
}

func TestFindBoundary(t *testing.T) {
	pol := CDCPolicy(256)
	p := bytes.Repeat([]byte{1, 2, 3, 4, 5, 7, 9}, 200)
	cut, ok := FindBoundary(p, pol)
	if cut <= 0 || cut > len(p) {
		t.Fatalf("cut=%d ok=%v", cut, ok)
	}
	_ = Fingerprint(p)
	_ = AlignPow2(100, 64)
}

func TestSplitterEOF(t *testing.T) {
	sp := NewSplitter(bytes.NewReader([]byte("ab")), Normalize(Policy{Mode: ModeFixed, Avg: 8}))
	p, err := sp.Next()
	if err != nil {
		t.Fatal(err)
	}
	if string(p.Data) != "ab" {
		t.Fatal(string(p.Data))
	}
	_, err = sp.Next()
	if err != io.EOF {
		t.Fatalf("want EOF got %v", err)
	}
}
