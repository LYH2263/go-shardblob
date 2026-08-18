package shardblob

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LYH2263/go-shardblob/internal/layout"
)

func TestBug08_IndexAfterManifestWrite(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, WithChunkSize(32))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	mfDir := filepath.Join(dir, layout.DirManifests)
	if err := os.RemoveAll(mfDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mfDir, []byte("not-a-dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = s.PutBytes([]byte("index-before-manifest-payload"))
	if err == nil {
		t.Fatal("expected put to fail when manifests path is a file")
	}
	list, lerr := s.List()
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(list) != 0 {
		t.Fatalf("index must wait until manifest write; leaked %d objects", len(list))
	}
}
