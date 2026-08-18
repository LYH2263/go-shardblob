package shardblob

import (
	"bytes"
	"testing"
)

func TestBug04_GCOnlyDeletesZeroRefcount(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(8))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	payload := []byte("keepthis")
	id, err := s.PutBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	st, err := s.GC()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(id); err != nil {
		t.Fatalf("gc deleted live chunk (refcount==1): %v stats=%+v", err, st)
	}
	got, err := s.GetBytes(id)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("get after gc: %v %q", err, got)
	}
}
