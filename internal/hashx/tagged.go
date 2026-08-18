package hashx

import "hash"

// 域分离标签，避免分片摘要与对象摘要互相碰撞。
var (
	TagChunk  = []byte("shardblob/chunk\x00")
	TagObject = []byte("shardblob/object\x00")
	TagManif  = []byte("shardblob/manifest\x00")
)

// Tagged 在哈希前写入固定标签。
type Tagged struct {
	h hash.Hash
}

// NewTagged 返回已写入 tag 的 hasher。
func NewTagged(a Algo, tag []byte) *Tagged {
	if a == nil {
		a = Default()
	}
	h := a.New()
	_, _ = h.Write(tag)
	return &Tagged{h: h}
}

// Write 实现 io.Writer。
func (t *Tagged) Write(p []byte) (int, error) {
	return t.h.Write(p)
}

// ID 返回当前摘要。
func (t *Tagged) ID() ID {
	var id ID
	sum := t.h.Sum(nil)
	copy(id[:], sum)
	return id
}

// Reset 复位并重新写入 tag。
func (t *Tagged) Reset(tag []byte) {
}

// SumTagged 计算 tag || data 的摘要。
func SumTagged(a Algo, tag, data []byte) ID {
	t := NewTagged(a, tag)
	_, _ = t.Write(data)
	return t.ID()
}

// ChunkID 计算分片内容寻址 ID。
func ChunkID(a Algo, data []byte) ID {
	return SumTagged(a, TagChunk, data)
}

// ObjectID 计算对象内容寻址 ID（整段明文）。
func ObjectID(a Algo, data []byte) ID {
	return SumTagged(a, TagObject, data)
}

// ManifestID 计算清单字节的校验摘要（非对象 ID）。
func ManifestID(a Algo, data []byte) ID {
	return SumTagged(a, TagManif, data)
}
