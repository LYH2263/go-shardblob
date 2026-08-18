package shardblob

import (
	"bytes"
	"testing"
)

func TestBug01_GetBytesIndependentFromStore(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(8))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	payload := []byte("mutate-me-please!!")
	id, err := s.PutBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetBytes(id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("roundtrip %q", got)
	}
	got[0] ^= 0xff
	got2, err := s.GetBytes(id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got2, payload) {
		t.Fatalf("caller mutate corrupted blobstore: got %q want %q", got2, payload)
	}
}
