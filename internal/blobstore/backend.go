package blobstore

import (
	"errors"
	"io"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

var (
	ErrNotFound = errors.New("blobstore: chunk not found")
	ErrClosed   = errors.New("blobstore: closed")
	ErrCorrupt  = errors.New("blobstore: chunk corrupt")
)

// Backend 内容寻址分片存储。相同 ID 只保留一份。
type Backend interface {
	Put(id hashx.ID, data []byte) error
	Get(id hashx.ID) ([]byte, error)
	Open(id hashx.ID) (io.ReadCloser, error)
	Has(id hashx.ID) (bool, error)
	Delete(id hashx.ID) error
	List(fn func(hashx.ID) error) error
	Bytes() int64
	Count() int
	Close() error
}

// Copy 把 src 中 ids 指定的分片拷到 dst（已存在则跳过）。
func Copy(dst, src Backend, ids []hashx.ID) error {
	for _, id := range ids {
		ok, err := dst.Has(id)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		data, err := src.Get(id)
		if err != nil {
			return err
		}
		if err := dst.Put(id, data); err != nil {
			return err
		}
	}
	return nil
}

// Stat 描述后端占用。
type Stat struct {
	Chunks int
	Bytes  int64
}

func Inspect(b Backend) Stat {
	return Stat{Chunks: b.Count(), Bytes: b.Bytes()}
}
