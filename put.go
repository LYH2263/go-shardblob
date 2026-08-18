package shardblob

import (
	"bytes"
	"fmt"
	"io"

	"github.com/LYH2263/go-shardblob/internal/blobstore"
	"github.com/LYH2263/go-shardblob/internal/chunk"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/index"
	"github.com/LYH2263/go-shardblob/internal/manifest"
)

// Put 切分 reader、写入分片、写清单并建索引。相同内容只增对象引用。
func (s *Store) Put(r io.Reader) (ObjectID, error) {
	done, err := s.beginIO()
	if err != nil {
		return ObjectID{}, err
	}
	defer done()
	if r == nil {
		return ObjectID{}, fmt.Errorf("%w: nil reader", ErrInvalid)
	}

	sp := chunk.NewSplitter(r, s.opts.policy)
	obj := hashx.NewTagged(s.algo, hashx.TagObject)
	var entries []manifest.Entry
	var written []hashx.ID // 本次新写入的分片；失败时回滚以免孤儿块
	rollback := func() { _ = blobstore.Rollback(s.blobs, written) }
	for {
		p, nerr := sp.Next()
		if nerr == io.EOF {
			break
		}
		if nerr != nil {
			rollback()
			return ObjectID{}, nerr
		}
		if _, err := obj.Write(p.Data); err != nil {
			rollback()
			return ObjectID{}, err
		}
		cid := hashx.ChunkID(s.algo, p.Data)
		existed, herr := s.blobs.Has(cid)
		if herr != nil {
			rollback()
			return ObjectID{}, herr
		}
		// 先登记再写入：校验失败时该分片可能已落盘，必须纳入回滚
		if !existed {
			written = append(written, cid)
		}
		if _, err := blobstore.PutChecked(s.blobs, s.algo, p.Data); err != nil {
			rollback()
			return ObjectID{}, err
		}
		entries = append(entries, manifest.Entry{
			ID:     cid,
			Size:   uint32(len(p.Data)),
			Offset: p.Offset,
		})
	}
	oid := obj.ID()
	unlock := s.fence.LockObject(oid)
	defer unlock()

	if rec, ok := s.idx.Get(oid); ok {
		rec.UseCount++
		s.idx.Put(rec)
		if err := s.persist(); err != nil {
			rec.UseCount--
			if rec.UseCount == 0 {
				s.idx.Delete(oid)
			} else {
				s.idx.Put(rec)
			}
			return ObjectID{}, err
		}
		return fromHX(oid), nil
	}

	mf := manifest.New(s.algo.Name(), uint32(s.opts.policy.Avg), s.opts.policy.Mode == chunk.ModeCDC, oid, entries)
	raw, err := manifest.Encode(mf)
	if err != nil {
		return ObjectID{}, err
	}
	if err := s.writeManifest(oid, raw); err != nil {
		return ObjectID{}, err
	}
	if err := s.refs.AddMany(mf.ChunkIDs()); err != nil {
		return ObjectID{}, err
	}
	s.idx.Put(index.Record{
		ID:       oid,
		UseCount: 1,
		Size:     mf.TotalSize,
		Chunks:   uint32(len(entries)),
	})
	if err := s.persist(); err != nil {
		return ObjectID{}, err
	}
	return fromHX(oid), nil
}

// PutBytes 写入内存缓冲。
func (s *Store) PutBytes(p []byte) (ObjectID, error) {
	if p == nil {
		p = []byte{}
	}
	return s.Put(bytes.NewReader(p))
}
