package shardblob

import (
	"testing"

	"github.com/LYH2263/go-shardblob/internal/blobstore"
	"github.com/LYH2263/go-shardblob/internal/hashx"
)

type poisonGet struct {
	blobstore.Backend
	gets int
}

func (p *poisonGet) Get(id hashx.ID) ([]byte, error) {
	p.gets++
	data, err := p.Backend.Get(id)
	if err != nil {
		return nil, err
	}
	if p.gets >= 2 && len(data) > 0 {
		cp := append([]byte(nil), data...)
		cp[0] ^= 0xff
		return cp, nil
	}
	return data, nil
}

func TestBug02_PutHashFailRollsBackChunks(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(4))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	inner := s.blobs
	s.blobs = &poisonGet{Backend: inner}
	_, err = s.PutBytes([]byte("abcdefghijkl"))
	if err == nil {
		t.Fatal("expected hash/validate failure after writing chunks")
	}
	if inner.Count() != 0 {
		t.Fatalf("orphan leftover chunks=%d", inner.Count())
	}
}
