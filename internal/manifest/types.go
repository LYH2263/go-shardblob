package manifest

import "github.com/LYH2263/go-shardblob/internal/hashx"

const (
	Magic     = uint32(0x424C4253) // "SBLB" little-endian
	Version1  = uint16(1)
	FlagCDC   = uint16(1 << 0)
	MaxChunks = 1 << 20
)

// Entry 清单中的一片。
type Entry struct {
	ID     hashx.ID
	Size   uint32
	Offset uint64
}

// Manifest 对象清单：分片序列、总长、对象摘要。半写文件不得进入索引。
type Manifest struct {
	Version   uint16
	Flags     uint16
	Algo      string
	ChunkSize uint32
	TotalSize uint64
	ObjectID  hashx.ID
	Chunks    []Entry
}

// CDC 是否内容定义切分。
func (m *Manifest) CDC() bool {
	return m != nil && m.Flags&FlagCDC != 0
}

// ChunkCount 分片数。空对象为 0。
func (m *Manifest) ChunkCount() int {
	if m == nil {
		return 0
	}
	return len(m.Chunks)
}

// Clone 深拷贝。
func (m *Manifest) Clone() *Manifest {
	if m == nil {
		return nil
	}
	out := *m
	if m.Chunks != nil {
		out.Chunks = make([]Entry, len(m.Chunks))
		copy(out.Chunks, m.Chunks)
	}
	return &out
}

// ChunkIDs 返回所有分片 ID。
func (m *Manifest) ChunkIDs() []hashx.ID {
	if m == nil {
		return nil
	}
	ids := make([]hashx.ID, len(m.Chunks))
	for i, e := range m.Chunks {
		ids[i] = e.ID
	}
	return ids
}

// LastSize 尾块长度；无分片时为 0。
func (m *Manifest) LastSize() uint32 {
	if m == nil || len(m.Chunks) == 0 {
		return 0
	}
	return m.Chunks[len(m.Chunks)-1].Size
}

func (m *Manifest) setCDC(on bool) {
	if on {
		m.Flags |= FlagCDC
	} else {
		m.Flags &^= FlagCDC
	}
}
