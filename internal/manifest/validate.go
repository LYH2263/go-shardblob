package manifest

import (
	"fmt"
	"strings"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

// Validate 检查清单内部一致性。不完整/错位清单不得对外可见。
func Validate(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("manifest: nil")
	}
	if m.Version == 0 {
		m.Version = Version1
	}
	if m.Version != Version1 {
		return fmt.Errorf("manifest: unsupported version %d", m.Version)
	}
	if strings.TrimSpace(m.Algo) == "" {
		return fmt.Errorf("manifest: empty algo")
	}
	if len(m.Algo) > 31 {
		return fmt.Errorf("manifest: algo name too long")
	}
	if len(m.Chunks) > MaxChunks {
		return fmt.Errorf("manifest: too many chunks")
	}
	if m.ObjectID.IsZero() {
		return fmt.Errorf("manifest: zero object id")
	}
	var sum uint64
	var expectOff uint64
	for i, e := range m.Chunks {
		if e.Size == 0 {
			return fmt.Errorf("manifest: chunk %d empty", i)
		}
		if e.ID.IsZero() {
			return fmt.Errorf("manifest: chunk %d zero id", i)
		}
		if e.Offset != expectOff {
			return fmt.Errorf("manifest: chunk %d offset %d want %d", i, e.Offset, expectOff)
		}
		if m.ChunkSize > 0 && !m.CDC() && i < len(m.Chunks)-1 && e.Size != m.ChunkSize {
			return fmt.Errorf("manifest: chunk %d size %d want %d", i, e.Size, m.ChunkSize)
		}
		if m.ChunkSize > 0 && !m.CDC() && e.Size > m.ChunkSize {
			return fmt.Errorf("manifest: chunk %d longer than chunk size", i)
		}
		sum += uint64(e.Size)
		expectOff += uint64(e.Size)
	}
	if sum != m.TotalSize {
		return fmt.Errorf("manifest: size %d != sum(chunks) %d", m.TotalSize, sum)
	}
	return nil
}

// New 构造清单并填 Flags。
func New(algo string, chunkSize uint32, cdc bool, oid hashx.ID, chunks []Entry) *Manifest {
	copied := make([]Entry, len(chunks))
	copy(copied, chunks)
	m := &Manifest{
		Version:   Version1,
		Algo:      algo,
		ChunkSize: chunkSize,
		ObjectID:  oid,
		Chunks:    copied,
	}
	m.setCDC(cdc)
	for _, e := range copied {
		m.TotalSize += uint64(e.Size)
	}
	return m
}
