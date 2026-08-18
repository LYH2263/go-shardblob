package shardblob

import (
	"testing"
)

func TestBug07_VerifyFailsWhenChunkMissing(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(8))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id, err := s.PutBytes([]byte("0123456789abcdefXX"))
	if err != nil {
		t.Fatal(err)
	}
	mf, err := s.loadManifest(toHX(id))
	if err != nil || len(mf.Chunks) == 0 {
		t.Fatalf("manifest %v chunks=%v", err, mf)
	}
	if err := s.blobs.Delete(mf.Chunks[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(id); err == nil {
		t.Fatal("verify succeeded for missing chunk")
	}
}
