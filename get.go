package shardblob

import (
	"bytes"
	"fmt"
	"io"

	"github.com/LYH2263/go-shardblob/internal/manifest"
)

// Get 返回只读流。调用方必须 Close，以便释放与 GC 的共享栅栏。
func (s *Store) Get(id ObjectID) (io.ReadCloser, error) {
	done, err := s.beginIO()
	if err != nil {
		return nil, err
	}
	hid := toHX(id)
	if !s.idx.Has(hid) {
		done()
		return nil, ErrNotFound
	}
	mf, err := s.loadManifest(hid)
	if err != nil {
		done()
		if err == ErrNotFound {
			return nil, ErrIncomplete
		}
		return nil, err
	}
	return &blobReader{s: s, mf: mf, release: done}, nil
}

// GetBytes 读出全部内容。
func (s *Store) GetBytes(id ObjectID) ([]byte, error) {
	rc, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Has 报告对象是否在索引中（完整清单才可见）。
func (s *Store) Has(id ObjectID) (bool, error) {
	done, err := s.beginIO()
	if err != nil {
		return false, err
	}
	defer done()
	return s.idx.Has(toHX(id)), nil
}

// Stat 返回对象元数据。
func (s *Store) Stat(id ObjectID) (Info, error) {
	done, err := s.beginIO()
	if err != nil {
		return Info{}, err
	}
	defer done()
	rec, ok := s.idx.Get(toHX(id))
	if !ok {
		return Info{}, ErrNotFound
	}
	return Info{
		ID:       fromHX(rec.ID),
		Size:     rec.Size,
		Chunks:   int(rec.Chunks),
		UseCount: rec.UseCount,
	}, nil
}

type blobReader struct {
	s       *Store
	mf      *manifest.Manifest
	i       int
	buf     []byte
	off     int
	closed  bool
	release func()
}

func (r *blobReader) Read(p []byte) (int, error) {
	if r.closed {
		return 0, fmt.Errorf("shardblob: reader closed")
	}
	for {
		if r.off < len(r.buf) {
			n := copy(p, r.buf[r.off:])
			r.off += n
			return n, nil
		}
		if r.i >= len(r.mf.Chunks) {
			return 0, io.EOF
		}
		e := r.mf.Chunks[r.i]
		data, err := r.s.blobs.Get(e.ID)
		if err != nil {
			return 0, err
		}
		r.buf = data
		r.off = 0
		r.i++
	}
}

func (r *blobReader) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	if r.release != nil {
		r.release()
		r.release = nil
	}
	return nil
}
