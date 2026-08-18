package gc

import (
	"testing"

	"github.com/LYH2263/go-shardblob/internal/blobstore"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/refcount"
)

func TestGCKeepsLive(t *testing.T) {
	blobs := blobstore.NewMemory()
	refs := refcount.NewTable()
	algo := hashx.Default()
	live := []byte("keep-me")
	dead := []byte("drop-me")
	lid := hashx.ChunkID(algo, live)
	did := hashx.ChunkID(algo, dead)
	_ = blobs.Put(lid, live)
	_ = blobs.Put(did, dead)
	_ = refs.Add(lid, 1)

	col := New(blobs, refs)
	st, err := col.Run()
	if err != nil {
		t.Fatal(err)
	}
	if st.Deleted != 1 || st.Retained != 1 {
		t.Fatalf("%+v", st)
	}
	ok, _ := blobs.Has(lid)
	if !ok {
		t.Fatal("live chunk deleted")
	}
	ok, _ = blobs.Has(did)
	if ok {
		t.Fatal("dead chunk remains")
	}
	if err := AssertSafe(blobs, refs); err != nil {
		t.Fatal(err)
	}
}

func TestDryRun(t *testing.T) {
	blobs := blobstore.NewMemory()
	refs := refcount.NewTable()
	id := hashx.ChunkID(hashx.Default(), []byte("x"))
	_ = blobs.Put(id, []byte("x"))
	c := New(blobs, refs)
	c.Dry = true
	st, err := c.Run()
	if err != nil {
		t.Fatal(err)
	}
	if st.Deleted != 1 {
		t.Fatal(st)
	}
	ok, _ := blobs.Has(id)
	if !ok {
		t.Fatal("dry run deleted")
	}
}
