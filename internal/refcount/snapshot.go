package refcount

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/layout"
)

const (
	snapMagic   = uint32(0x53465253) // "SRFS"
	snapVersion = uint16(1)
)

// Save 把表快照原子写入 path。
func Save(path string, t *Table) error {
	m := t.Snapshot()
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, snapMagic)
	_ = binary.Write(&buf, binary.LittleEndian, snapVersion)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(m)))
	for id, n := range m {
		_, _ = buf.Write(id[:])
		_ = binary.Write(&buf, binary.LittleEndian, n)
	}
	sum := crc32.ChecksumIEEE(buf.Bytes())
	_ = binary.Write(&buf, binary.LittleEndian, sum)
	return layout.WriteAtomic(path, buf.Bytes())
}

// Load 读取快照到新表。文件不存在则返回空表。
func Load(path string) (*Table, error) {
	t := NewTable()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return nil, err
	}
	if len(raw) < 4+2+4+4 {
		return nil, fmt.Errorf("refcount: snapshot too short")
	}
	body := raw[:len(raw)-4]
	want := binary.LittleEndian.Uint32(raw[len(raw)-4:])
	if crc32.ChecksumIEEE(body) != want {
		return nil, fmt.Errorf("refcount: snapshot crc")
	}
	r := bytes.NewReader(body)
	var magic uint32
	var ver uint16
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if magic != snapMagic {
		return nil, fmt.Errorf("refcount: bad snapshot magic")
	}
	if err := binary.Read(r, binary.LittleEndian, &ver); err != nil {
		return nil, err
	}
	if ver != snapVersion {
		return nil, fmt.Errorf("refcount: snapshot version %d", ver)
	}
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	m := make(map[hashx.ID]uint64, n)
	for i := uint32(0); i < n; i++ {
		var id hashx.ID
		if _, err := io.ReadFull(r, id[:]); err != nil {
			return nil, err
		}
		var c uint64
		if err := binary.Read(r, binary.LittleEndian, &c); err != nil {
			return nil, err
		}
		if c > 0 {
			m[id] = c
		}
	}
	t.LoadMap(m)
	return t, nil
}
