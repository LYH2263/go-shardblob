package blobstore

import (
	"fmt"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

// PutChecked 写入前校验 data 的内容寻址 ID 是否匹配。
func PutChecked(b Backend, algo hashx.Algo, data []byte) (hashx.ID, error) {
	id := hashx.ChunkID(algo, data)
	if err := b.Put(id, data); err != nil {
		return hashx.Zero, err
	}
	got, err := b.Get(id)
	if err != nil {
		return id, err
	}
	if !hashx.EqualID(hashx.ChunkID(algo, got), id) {
		return id, fmt.Errorf("%w: stored bytes mismatch id %s", ErrCorrupt, id.Hex())
	}
	return id, nil
}

// Rollback 删除本次新写入的分片；校验失败路径必须调用以免孤儿块。
// 只应传入本次 Put 新写入（写入前不存在）的 ID，避免误删共享分片。
func Rollback(b Backend, ids []hashx.ID) error {
	for _, id := range ids {
		if err := b.Delete(id); err != nil {
			return err
		}
	}
	return nil
}

// VerifyID 确认后端中该分片内容与 ID 一致。
func VerifyID(b Backend, algo hashx.Algo, id hashx.ID) error {
	data, err := b.Get(id)
	if err != nil {
		return err
	}
	got := hashx.ChunkID(algo, data)
	if !hashx.EqualID(got, id) {
		return fmt.Errorf("%w: %s != %s", ErrCorrupt, got.Hex(), id.Hex())
	}
	return nil
}

// VerifyAll 遍历全部分片做内容校验。
func VerifyAll(b Backend, algo hashx.Algo) error {
	return b.List(func(id hashx.ID) error {
		return VerifyID(b, algo, id)
	})
}
