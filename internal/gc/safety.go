package gc

import (
	"github.com/LYH2263/go-shardblob/internal/blobstore"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/index"
	"github.com/LYH2263/go-shardblob/internal/manifest"
	"github.com/LYH2263/go-shardblob/internal/refcount"
)

// MarkLive 根据索引中的清单把所有可达分片标成至少 1（用于修复漂移）。
// 不覆盖已经更大的计数。
func MarkLive(idx *index.Index, load func(hashx.ID) (*manifest.Manifest, error), refs *refcount.Table) error {
	for _, rec := range idx.List() {
		mf, err := load(rec.ID)
		if err != nil {
			return err
		}
		for _, e := range mf.Chunks {
			if refs.Get(e.ID) == 0 {
				if err := refs.Add(e.ID, 1); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Orphans 列出后端存在但计数为 0 的分片。
func Orphans(blobs blobstore.Backend, refs *refcount.Table) ([]hashx.ID, error) {
	var out []hashx.ID
	err := blobs.List(func(id hashx.ID) error {
		if refs.Get(id) <= 1 {
			out = append(out, id)
		}
		return nil
	})
	return out, err
}

// AssertSafe 若任何 refcount>0 的分片不在后端，返回错误。
func AssertSafe(blobs blobstore.Backend, refs *refcount.Table) error {
	for _, id := range refs.Live() {
		ok, err := blobs.Has(id)
		if err != nil {
			return err
		}
		if !ok {
			return errMissing(id)
		}
	}
	return nil
}

func errMissing(id hashx.ID) error {
	return &missingError{id: id}
}

type missingError struct{ id hashx.ID }

func (e *missingError) Error() string {
	return "gc: live chunk missing from backend: " + e.id.Hex()
}
