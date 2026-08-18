package shardblob

import (
	"bytes"
	"testing"
)

func TestBug05_AddManyPerManifestEntry(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(4))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	payload := []byte("AAAABBBBAAAA") // AAAA appears twice as full chunks
	id, err := s.PutBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetBytes(id)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("roundtrip %v %q", err, got)
	}
	if err := s.Delete(id); err != nil {
		t.Fatalf("delete should SubMany per entry, got %v", err)
	}
	ok, err := s.Has(id)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("object should be gone")
	}
	if _, err := s.GC(); err != nil {
		t.Fatal(err)
	}
}
