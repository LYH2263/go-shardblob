package verify

import (
	"testing"

	"github.com/LYH2263/go-shardblob/internal/blobstore"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/manifest"
)

func TestObjectOKAndTamper(t *testing.T) {
	blobs := blobstore.NewMemory()
	algo := hashx.Default()
	data := []byte("abcdef")
	cid := hashx.ChunkID(algo, data)
	_ = blobs.Put(cid, data)
	oid := hashx.ObjectID(algo, data)
	mf := manifest.New(algo.Name(), uint32(len(data)), false, oid, []manifest.Entry{
		{ID: cid, Size: uint32(len(data)), Offset: 0},
	})
	rep, err := Object(blobs, mf)
	if err != nil || !rep.OK() {
		t.Fatalf("ok case: %v %+v", err, rep)
	}

	_ = blobs.Delete(cid)
	_ = blobs.Put(cid, []byte("XXXXXX"))
	rep, err = Object(blobs, mf)
	if err == nil || rep.OK() {
		t.Fatal("expected hash fault")
	}
	found := false
	for _, f := range rep.Faults {
		if f.Kind == KindChunkHash || f.Kind == KindChunkSize {
			found = true
		}
	}
	if !found {
		t.Fatalf("faults %+v", rep.Faults)
	}
}

func TestChunkHelper(t *testing.T) {
	algo := hashx.Default()
	d := []byte("z")
	id := hashx.ChunkID(algo, d)
	if err := Chunk(algo, id, d); err != nil {
		t.Fatal(err)
	}
	if err := Chunk(algo, id, []byte("y")); err == nil {
		t.Fatal("expected mismatch")
	}
}
