package refcount

import (
	"path/filepath"
	"testing"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

func TestAddSub(t *testing.T) {
	tb := NewTable()
	id := hashx.SumSHA256([]byte("c"))
	if err := tb.Add(id, 2); err != nil {
		t.Fatal(err)
	}
	if tb.Get(id) != 2 {
		t.Fatal(tb.Get(id))
	}
	if err := tb.Sub(id, 1); err != nil {
		t.Fatal(err)
	}
	if tb.ZeroOrMissing(id) {
		t.Fatal("still live")
	}
	if err := tb.Sub(id, 1); err != nil {
		t.Fatal(err)
	}
	if !tb.ZeroOrMissing(id) {
		t.Fatal("should be zero")
	}
	if err := tb.Sub(id, 1); err == nil {
		t.Fatal("underflow")
	}
}

func TestSnapshotJournal(t *testing.T) {
	dir := t.TempDir()
	tb := NewTable()
	id := hashx.SumSHA256([]byte("j"))
	path := filepath.Join(dir, "refs.bin")
	jp := filepath.Join(dir, "refs.journal")
	j, err := OpenJournal(jp)
	if err != nil {
		t.Fatal(err)
	}
	tb.AttachJournal(j)
	if err := tb.Add(id, 3); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, tb); err != nil {
		t.Fatal(err)
	}
	if err := j.Truncate(); err != nil {
		t.Fatal(err)
	}
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	j2, err := OpenJournal(jp)
	if err != nil {
		t.Fatal(err)
	}
	defer j2.Close()
	if err := loaded.Add(id, 1); err != nil {
		t.Fatal(err)
	}
	loaded.AttachJournal(j2)
	if err := loaded.Add(id, 1); err != nil {
		t.Fatal(err)
	}
	// 未刷快照，靠 journal 恢复
	fresh, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	j3, err := OpenJournal(jp)
	if err != nil {
		t.Fatal(err)
	}
	defer j3.Close()
	if err := j3.Replay(fresh); err != nil {
		t.Fatal(err)
	}
	if fresh.Get(id) != 4 {
		t.Fatalf("got %d want 4", fresh.Get(id))
	}
}
