package shardblob

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/layout"
)

func TestPutGetRoundtrip(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(8))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	cases := [][]byte{
		nil,
		{},
		[]byte("a"),
		[]byte("1234567"),      // short of 8
		[]byte("12345678"),     // exact
		[]byte("123456789012"), // 8+4
		bytes.Repeat([]byte("xyz"), 100),
	}
	for i, in := range cases {
		id, err := s.PutBytes(in)
		if err != nil {
			t.Fatalf("case %d put: %v", i, err)
		}
		got, err := s.GetBytes(id)
		if err != nil {
			t.Fatalf("case %d get: %v", i, err)
		}
		if !bytes.Equal(got, in) {
			t.Fatalf("case %d mismatch got %q want %q", i, got, in)
		}
		if err := s.Verify(id); err != nil {
			t.Fatalf("case %d verify: %v", i, err)
		}
		info, err := s.Stat(id)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size != uint64(len(in)) {
			t.Fatalf("size %d want %d", info.Size, len(in))
		}
		wantID := hashx.ObjectID(hashx.Default(), in)
		if fromHX(wantID) != id {
			t.Fatalf("object id mismatch")
		}
	}
}

func TestDedupAndUseCount(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(16))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	payload := bytes.Repeat([]byte("same-bytes-content"), 4)
	id1, err := s.PutBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.PutBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatal("content-addressed ids differ")
	}
	info, _ := s.Stat(id1)
	if info.UseCount != 2 {
		t.Fatalf("use %d", info.UseCount)
	}
	st, _ := s.StoreStats()
	if st.Objects != 1 {
		t.Fatalf("objects %d", st.Objects)
	}
	if st.DedupRatio() < 1 {
		t.Fatalf("ratio %v", st.DedupRatio())
	}
	if err := s.Delete(id1); err != nil {
		t.Fatal(err)
	}
	ok, _ := s.Has(id1)
	if !ok {
		t.Fatal("should remain after first delete")
	}
	if err := s.Delete(id1); err != nil {
		t.Fatal(err)
	}
	ok, _ = s.Has(id1)
	if ok {
		t.Fatal("should be gone")
	}
}

func TestSharedPrefixChunks(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(4))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := []byte("AAAA" + "BBBB")
	b := []byte("AAAA" + "CCCC")
	idA, err := s.PutBytes(a)
	if err != nil {
		t.Fatal(err)
	}
	idB, err := s.PutBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	if idA == idB {
		t.Fatal("different content")
	}
	st, _ := s.StoreStats()
	// 3 unique chunks: AAAA, BBBB, CCCC
	if st.Chunks != 3 {
		t.Fatalf("chunks %d want 3", st.Chunks)
	}
	if err := s.Delete(idA); err != nil {
		t.Fatal(err)
	}
	gcst, err := s.GC()
	if err != nil {
		t.Fatal(err)
	}
	if gcst.Deleted != 1 { // BBBB
		t.Fatalf("deleted %d %+v", gcst.Deleted, gcst)
	}
	got, err := s.GetBytes(idB)
	if err != nil || !bytes.Equal(got, b) {
		t.Fatal(err, got)
	}
}

func TestGCDoesNotDeleteLive(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(8), WithAutoGC(true))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id, err := s.PutBytes([]byte("keepthis"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GC(); err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(id); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(id); err != nil {
		t.Fatal(err)
	}
	ok, _ := s.Has(id)
	if ok {
		t.Fatal("deleted object still visible")
	}
}

func TestIncompleteManifestNotVisible(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, WithChunkSize(32))
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.PutBytes([]byte("visible-object"))
	if err != nil {
		t.Fatal(err)
	}
	hid := toHX(id)
	tmp := layout.ManifestTmp(dir, hid)
	if err := os.WriteFile(tmp, []byte("not-a-real-manifest"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 半写文件存在时，索引仍只暴露完整对象
	ok, err := s.Has(id)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	fake := hashx.SumSHA256([]byte("no-such"))
	if s.idx.Has(fake) {
		t.Fatal("fake in index")
	}
	if layout.Exists(layout.ManifestPath(dir, fake)) {
		t.Fatal("fake manifest")
	}
	_ = s.Close()
}

func TestDiskReopen(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, WithChunkSize(16))
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("persist-me-please!!")
	id, err := s.PutBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(dir, WithChunkSize(16))
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	got, err := s2.GetBytes(id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal(string(got))
	}
	if err := s2.VerifyAll(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigMismatch(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, WithChunkSize(32))
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	_, err = Open(dir, WithChunkSize(64))
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("got %v", err)
	}
}

func TestCDCStore(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(256), WithCDC(true))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	data := bytes.Repeat([]byte("cdc-payload-block-"), 40)
	id, err := s.PutBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetBytes(id)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("cdc get mismatch")
	}
	info, _ := s.Stat(id)
	if info.Chunks < 1 {
		t.Fatal(info.Chunks)
	}
}

func TestVerifyTamper(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, WithChunkSize(8))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id, err := s.PutBytes([]byte("tamperxxYYYYZZZZ"))
	if err != nil {
		t.Fatal(err)
	}
	info, _ := s.Stat(id)
	if info.Chunks < 1 {
		t.Fatal("no chunks")
	}
	// 破坏一个分片文件
	var broken bool
	root := filepath.Join(dir, layout.DirChunks)
	_ = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || broken {
			return nil
		}
		if len(fi.Name()) == 64 {
			b, _ := os.ReadFile(path)
			if len(b) > 0 {
				b[0] ^= 0xff
				_ = os.WriteFile(path, b, 0o644)
				broken = true
			}
		}
		return nil
	})
	if !broken {
		t.Fatal("did not tamper")
	}
	if err := s.Verify(id); err == nil {
		t.Fatal("expected verify fail")
	}
}

func TestConcurrentPut(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(32))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var wg sync.WaitGroup
	ids := make([]ObjectID, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p := bytes.Repeat([]byte{byte(i)}, 100)
			id, err := s.PutBytes(p)
			if err != nil {
				t.Errorf("put %d: %v", i, err)
				return
			}
			ids[i] = id
		}(i)
	}
	wg.Wait()
	for i, id := range ids {
		if id.IsZero() {
			continue
		}
		got, err := s.GetBytes(id)
		if err != nil {
			t.Fatal(err)
		}
		want := bytes.Repeat([]byte{byte(i)}, 100)
		if !bytes.Equal(got, want) {
			t.Fatalf("%d mismatch", i)
		}
	}
}

func TestGetStream(t *testing.T) {
	s, err := OpenMemory(WithChunkSize(5))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	src := []byte("stream-read-all-chunks")
	id, err := s.PutBytes(src)
	if err != nil {
		t.Fatal(err)
	}
	rc, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 3)
	var out []byte
	for {
		n, err := rc.Read(buf)
		out = append(out, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := rc.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, src) {
		t.Fatal(string(out))
	}
}

func TestAlgoSHA512(t *testing.T) {
	s, err := OpenMemory(WithAlgo(hashx.NameSHA512256), WithChunkSize(16))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id, err := s.PutBytes([]byte("sha512"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Verify(id); err != nil {
		t.Fatal(err)
	}
}

func TestListAndNotFound(t *testing.T) {
	s, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Get(ObjectID{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	if err := s.Delete(ObjectID{1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	id, _ := s.PutBytes([]byte("x"))
	list, err := s.List()
	if err != nil || len(list) != 1 || list[0].ID != id {
		t.Fatalf("%+v %v", list, err)
	}
}

func TestParseObjectID(t *testing.T) {
	id, err := sPut(t)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseObjectID(id.String())
	if err != nil || p != id {
		t.Fatal(err, p)
	}
	text, _ := id.MarshalText()
	var u ObjectID
	if err := u.UnmarshalText(text); err != nil || u != id {
		t.Fatal(err)
	}
}

func sPut(t *testing.T) (ObjectID, error) {
	t.Helper()
	s, err := OpenMemory()
	if err != nil {
		return ObjectID{}, err
	}
	t.Cleanup(func() { s.Close() })
	return s.PutBytes([]byte("id"))
}
