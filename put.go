package shardblob

import (
	"bytes"
	"fmt"
	"io"

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
	for {
		p, nerr := sp.Next()
		if nerr == io.EOF {
			break
		}
		if nerr != nil {
			return ObjectID{}, nerr
		}
		if _, err := obj.Write(p.Data); err != nil {
			return ObjectID{}, err
		}
		cid := hashx.ChunkID(s.algo, p.Data)
		if err := s.blobs.Put(cid, p.Data); err != nil {
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
	if err := s.refs.AddMany(uniqueChunkIDs(mf.ChunkIDs())); err != nil {
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

func uniqueChunkIDs(ids []hashx.ID) []hashx.ID {
	seen := make(map[hashx.ID]struct{}, len(ids))
	out := make([]hashx.ID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
