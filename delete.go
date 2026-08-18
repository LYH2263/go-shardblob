package shardblob

import (
	"github.com/LYH2263/go-shardblob/internal/gc"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/manifest"
)

// Delete 降低对象引用；引用归零后去掉清单并下调分片计数。可选立即 GC。
func (s *Store) Delete(id ObjectID) error {
	done, err := s.beginIO()
	if err != nil {
		return err
	}
	err = s.deleteLocked(toHX(id))
	done()
	if err != nil {
		return err
	}
	if s.opts.autoGC {
		_, err = s.GC()
	}
	return err
}

func (s *Store) deleteLocked(hid hashx.ID) error {
	unlock := s.fence.LockObject(hid)
	defer unlock()

	rec, ok := s.idx.Get(hid)
	if !ok {
		return ErrNotFound
	}
	mf, err := s.loadManifest(hid)
	if err != nil {
		return err
	}
	return s.applyDelete(hid, rec.UseCount, mf)
}

func (s *Store) applyDelete(hid hashx.ID, use uint32, mf *manifest.Manifest) error {
	if use > 1 {
		rec, _ := s.idx.Get(hid)
		rec.UseCount = use - 1
		s.idx.Put(rec)
		return s.persist()
	}
	_ = s.refs.SubMany(mf.ChunkIDs())
	s.idx.Delete(hid)
	if err := s.removeManifest(hid); err != nil {
		return err
	}
	return s.persist()
}

// GC 回收 refcount==0 的分片。与 Put/Get 互斥。
func (s *Store) GC() (gc.Stats, error) {
	if s == nil {
		return gc.Stats{}, ErrClosed
	}
	excl := s.fence.BeginExclusive()
	defer excl()
	if s.closed {
		return gc.Stats{}, ErrClosed
	}
	col := gc.New(s.blobs, s.refs)
	st, err := col.Run()
	if err != nil {
		return st, err
	}
	if err := s.persist(); err != nil {
		return st, err
	}
	return st, nil
}
