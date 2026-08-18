package shardblob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

type cancelAfter struct {
	r      io.Reader
	n      int
	after  int
	cancel context.CancelFunc
}

func (c *cancelAfter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	if c.n >= c.after {
		c.cancel()
	}
	return n, err
}

func TestBug06_CanceledPutRollsBackChunks(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(4))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &cancelAfter{r: bytes.NewReader([]byte("abcdefghijkl")), after: 4, cancel: cancel}
	_, err = s.PutContext(ctx, r)
	if err == nil {
		t.Fatal("expected canceled put")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v want context.Canceled", err)
	}
	if s.blobs.Count() != 0 {
		t.Fatalf("orphan leftover chunks=%d", s.blobs.Count())
	}
}
