package blobstore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

func TestMemoryDedup(t *testing.T) {
	m := NewMemory()
	algo := hashx.Default()
	data := []byte("chunk-data")
	id, err := PutChecked(m, algo, data)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Put(id, []byte("ignored")); err != nil {
		t.Fatal(err)
	}
	got, err := m.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("dedup overwrite")
	}
	if m.Count() != 1 {
		t.Fatal(m.Count())
	}
	if err := VerifyAll(m, algo); err != nil {
		t.Fatal(err)
	}
	rc, err := m.Open(id)
	if err != nil {
		t.Fatal(err)
	}
	_ = rc.Close()
	if err := m.Delete(id); err != nil {
		t.Fatal(err)
	}
	if m.Count() != 0 {
		t.Fatal(m.Count())
	}
}

func TestDiskRoundtrip(t *testing.T) {
	dir := t.TempDir()
	d, err := OpenDisk(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	algo := hashx.Default()
	data := []byte("on-disk")
	id := hashx.ChunkID(algo, data)
	if err := d.Put(id, data); err != nil {
		t.Fatal(err)
	}
	got, err := d.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal(string(got))
	}
	ok, err := d.Has(id)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	n := 0
	if err := d.List(func(hashx.ID) error { n++; return nil }); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatal(n)
	}
	// 半写 tmp 不应被 List 看到
	tmp := filepath.Join(dir, "chunks", "00", "00", "foo.tmp")
	_ = os.MkdirAll(filepath.Dir(tmp), 0o755)
	_ = os.WriteFile(tmp, []byte("x"), 0o644)
	n = 0
	_ = d.List(func(hashx.ID) error { n++; return nil })
	if n != 1 {
		t.Fatalf("tmp counted n=%d", n)
	}
}

func TestMemoryNotFound(t *testing.T) {
	m := NewMemory()
	if _, err := m.Get(hashx.Zero); err != ErrNotFound {
		t.Fatalf("got %v", err)
	}
}
