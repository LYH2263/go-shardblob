package index

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	"github.com/LYH2263/go-shardblob/internal/layout"
)

const (
	idxMagic   = uint32(0x58444953) // "SIDX"
	idxVersion = uint16(1)
)

// Save 原子写入索引快照。
func Save(path string, x *Index) error {
	recs := x.List()
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, idxMagic)
	_ = binary.Write(&buf, binary.LittleEndian, idxVersion)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(recs)))
	for _, r := range recs {
		_, _ = buf.Write(r.ID[:])
		_ = binary.Write(&buf, binary.LittleEndian, r.UseCount)
		_ = binary.Write(&buf, binary.LittleEndian, r.Size)
		_ = binary.Write(&buf, binary.LittleEndian, r.Chunks)
		_ = binary.Write(&buf, binary.LittleEndian, r.CreatedAt)
	}
	sum := crc32.ChecksumIEEE(buf.Bytes())
	_ = binary.Write(&buf, binary.LittleEndian, sum)
	return layout.WriteAtomic(path, buf.Bytes())
}

// Load 读取快照。文件不存在则空索引。
func Load(path string) (*Index, error) {
	x := New()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return x, nil
		}
		return nil, err
	}
	if len(raw) < 4+2+4+4 {
		return nil, fmt.Errorf("index: snapshot too short")
	}
	body := raw[:len(raw)-4]
	want := binary.LittleEndian.Uint32(raw[len(raw)-4:])
	if crc32.ChecksumIEEE(body) != want {
		return nil, fmt.Errorf("index: snapshot crc")
	}
	r := bytes.NewReader(body)
	var magic uint32
	var ver uint16
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if magic != idxMagic {
		return nil, fmt.Errorf("index: bad magic")
	}
	if err := binary.Read(r, binary.LittleEndian, &ver); err != nil {
		return nil, err
	}
	if ver != idxVersion {
		return nil, fmt.Errorf("index: version %d", ver)
	}
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	recs := make([]Record, 0, n)
	for i := uint32(0); i < n; i++ {
		var rec Record
		if _, err := io.ReadFull(r, rec.ID[:]); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.UseCount); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.Size); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.Chunks); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &rec.CreatedAt); err != nil {
			return nil, err
		}
		if rec.ID.IsZero() {
			return nil, fmt.Errorf("index: zero object id in snapshot")
		}
		recs = append(recs, rec)
	}
	x.LoadAll(recs)
	return x, nil
}
