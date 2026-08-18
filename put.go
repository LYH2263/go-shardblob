package shardblob

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/LYH2263/go-shardblob/internal/chunk"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/index"
	"github.com/LYH2263/go-shardblob/internal/manifest"
)

// Put 切分 reader、写入分片、写清单并建索引。相同内容只增对象引用。
func (s *Store) Put(r io.Reader) (ObjectID, error) {
	return s.PutContext(context.Background(), r)
}

// PutContext 带取消语义的写入；取消后不得留下已写分片。
func (s *Store) PutContext(ctx context.Context, r io.Reader) (ObjectID, error) {
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
	var written []hashx.ID
	// rollback 删除本轮已新写的分片；取消或出错时调用，确保不留孤儿。
	rollback := func() { _ = s.abortNewChunks(written) }
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
		if err := s.blobs.Put(cid, p.Data); err != nil {
			rollback()
			return ObjectID{}, err
		}
		// 仅记录本轮新建的分片，回滚时才不会误删被既有对象引用的分片。
		if !existed {
			written = append(written, cid)
		}
		if err := s.ctxErr(ctx); err != nil {
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
		rollback()
		return ObjectID{}, err
	}
	if err := s.writeManifest(oid, raw); err != nil {
		rollback()
		return ObjectID{}, err
	}
	if err := s.refs.AddMany(mf.ChunkIDs()); err != nil {
		_ = s.removeManifest(oid)
		rollback()
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
