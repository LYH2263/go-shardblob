package shardblob

import (
	"errors"
	"testing"
)

func TestBug03_GetAfterCloseNoPanic(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(8))
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.PutBytes([]byte("after-close"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = s.GetBytes(id)
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("GetBytes after Close: %v want ErrClosed", err)
	}
}
