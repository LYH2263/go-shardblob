package index

import (
	"path/filepath"
	"testing"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

func TestIndexUseCount(t *testing.T) {
	x := New()
	id := hashx.SumSHA256([]byte("obj"))
	x.Put(Record{ID: id, UseCount: 1, Size: 10, Chunks: 2})
	if !x.Has(id) || x.Len() != 1 {
		t.Fatal(x.Len())
	}
	if _, err := x.IncUse(id); err != nil {
		t.Fatal(err)
	}
	r, gone, err := x.DecUse(id)
	if err != nil || gone || r.UseCount != 1 {
		t.Fatalf("%+v gone=%v err=%v", r, gone, err)
	}
	_, gone, err = x.DecUse(id)
	if err != nil || !gone {
		t.Fatal(gone, err)
	}
	if x.Has(id) {
		t.Fatal("still present")
	}
}

func TestIndexSnapshot(t *testing.T) {
	dir := t.TempDir()
	x := New()
	id := hashx.SumSHA256([]byte("snap"))
	x.Put(Record{ID: id, UseCount: 2, Size: 99, Chunks: 3})
	p := filepath.Join(dir, "index.bin")
	if err := Save(p, x); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := got.Get(id)
	if !ok || r.UseCount != 2 || r.Size != 99 {
		t.Fatalf("%+v ok=%v", r, ok)
	}
	if got.TotalBytes() != 99 {
		t.Fatal(got.TotalBytes())
	}
}
