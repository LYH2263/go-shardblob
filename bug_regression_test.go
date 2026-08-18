package shardblob

import (
	"testing"
)

func TestBug09_DeleteSubManyErrorKeepsRefs(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(8))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	live, err := s.PutBytes([]byte("live-object-aaaa"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(live); err != nil {
		t.Fatalf("normal delete: %v", err)
	}
	if _, err := s.GC(); err != nil {
		t.Fatal(err)
	}
	if s.blobs.Count() != 0 {
		t.Fatalf("refcount lie leaked chunks=%d", s.blobs.Count())
	}

	id, err := s.PutBytes([]byte("second-object-bbb"))
	if err != nil {
		t.Fatal(err)
	}
	s.refs.LoadMap(nil)
	err = s.Delete(id)
	if err == nil {
		t.Fatal("expected SubMany error to surface")
	}
	ok, herr := s.Has(id)
	if herr != nil {
		t.Fatal(herr)
	}
	if !ok {
		t.Fatal("index dropped after SubMany error; GC may kill live chunks or leak")
	}
}
