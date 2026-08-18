package verify

import (
	"fmt"

	"github.com/LYH2263/go-shardblob/internal/blobstore"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/manifest"
)

// Object 重读全部分片、重算校验，确认拼接后摘要等于写入摘要。
func Object(blobs blobstore.Backend, mf *manifest.Manifest) (*Report, error) {
	rep := &Report{Chunks: -1, Faults: nil}
	if mf == nil {
		rep.add(KindManifest, -1, "nil manifest")
		return rep, rep
	}
	rep.ObjectHex = mf.ObjectID.Hex()
	rep.Chunks = len(mf.Chunks)
	rep.Bytes = mf.TotalSize

	algo, err := hashx.Lookup(mf.Algo)
	if err != nil {
		rep.add(KindAlgo, -1, err.Error())
		return rep, rep
	}
	if vErr := manifest.Validate(mf); vErr != nil {
		rep.add(KindManifest, -1, vErr.Error())
		return rep, rep
	}

	obj := hashx.NewTagged(algo, hashx.TagObject)
	var wrote uint64
	for i, e := range mf.Chunks {
		data, gerr := blobs.Get(e.ID)
		if gerr != nil {
			rep.add(KindChunkMiss, i, gerr.Error())
			continue
		}
		if uint32(len(data)) != e.Size {
			rep.add(KindChunkSize, i, fmt.Sprintf("got %d want %d", len(data), e.Size))
		}
		got := hashx.ChunkID(algo, data)
		if !hashx.EqualID(got, e.ID) {
			rep.add(KindChunkHash, i, fmt.Sprintf("got %s want %s", got.Hex(), e.ID.Hex()))
		}
		_, _ = obj.Write(data)
		wrote += uint64(len(data))
	}
	if len(rep.Faults) > 0 {
		return rep, rep
	}
	if wrote != mf.TotalSize {
		rep.add(KindChunkSize, -1, fmt.Sprintf("assembled %d want %d", wrote, mf.TotalSize))
		return rep, rep
	}
	oid := obj.ID()
	if !hashx.EqualID(oid, mf.ObjectID) {
		rep.add(KindObjectHash, -1, fmt.Sprintf("got %s want %s", oid.Hex(), mf.ObjectID.Hex()))
		return rep, rep
	}
	return rep, nil
}

// Chunk 校验单片。
func Chunk(algo hashx.Algo, id hashx.ID, data []byte) error {
	got := hashx.ChunkID(algo, data)
	if !hashx.EqualID(got, id) {
		return fmt.Errorf("verify: chunk hash %s want %s", got.Hex(), id.Hex())
	}
	return nil
}
