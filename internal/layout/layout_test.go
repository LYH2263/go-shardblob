package layout

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

func TestPathsAndAtomic(t *testing.T) {
	root := t.TempDir()
	if err := EnsureRoot(root); err != nil {
		t.Fatal(err)
	}
	id := hashx.SumSHA256([]byte("p"))
	p := ChunkPath(root, id)
	if err := WriteAtomicExclusive(p, []byte("body")); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomicExclusive(p, []byte("other")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "body" {
		t.Fatal(string(b))
	}
	cfg := Config{Algo: "sha256", ChunkSize: 1024, CDC: false}
	if err := SaveConfig(root, cfg); err != nil {
		t.Fatal(err)
	}
	got, ok, err := LoadConfig(root)
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	if err := Compatible(got, cfg); err != nil {
		t.Fatal(err)
	}
	if err := Compatible(got, Config{Algo: "sha256", ChunkSize: 1}); err == nil {
		t.Fatal("expected mismatch")
	}
	if !Exists(ManifestPath(root, id) + TmpSuffix) {
		// 半写路径尚未创建是正常的
	}
	_ = HexID(id)
	_ = ChunkDir(root, id)
	_ = filepath.Join(root, DirMeta)
}

func TestRemoveMissing(t *testing.T) {
	if err := RemoveAllIfExist(filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Fatal(err)
	}
}
