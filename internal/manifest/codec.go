package manifest

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

const (
	headerSize = 4 + 2 + 2 + 32 + 4 + 4 + 8 + 32 // magic, ver, flags, algo[32], chunkSize, count, total, objectID
	entrySize  = 32 + 4 + 8
	crcSize    = 4
	algoField  = 32
)

// Encode 将清单编码为带 CRC32 的小端二进制。
func Encode(m *Manifest) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest: nil")
	}
	if err := Validate(m); err != nil {
		return nil, err
	}
	n := headerSize + entrySize*len(m.Chunks) + crcSize
	buf := bytes.NewBuffer(make([]byte, 0, n))
	var algo [algoField]byte
	copy(algo[:], []byte(m.Algo))

	_ = binary.Write(buf, binary.LittleEndian, Magic)
	_ = binary.Write(buf, binary.LittleEndian, m.Version)
	_ = binary.Write(buf, binary.LittleEndian, m.Flags)
	_, _ = buf.Write(algo[:])
	_ = binary.Write(buf, binary.LittleEndian, m.ChunkSize)
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(m.Chunks)))
	_ = binary.Write(buf, binary.LittleEndian, m.TotalSize)
	_, _ = buf.Write(m.ObjectID[:])

	for _, e := range m.Chunks {
		_, _ = buf.Write(e.ID[:])
		_ = binary.Write(buf, binary.LittleEndian, e.Size)
		_ = binary.Write(buf, binary.LittleEndian, e.Offset)
	}
	sum := crc32.ChecksumIEEE(buf.Bytes())
	_ = binary.Write(buf, binary.LittleEndian, sum)
	return buf.Bytes(), nil
}

// Decode 解析清单；CRC 或 magic 不对则失败。
func Decode(raw []byte) (*Manifest, error) {
	if len(raw) < headerSize+crcSize {
		return nil, fmt.Errorf("manifest: truncated header (%d bytes)", len(raw))
	}
	body := raw[:len(raw)-crcSize]
	want := binary.LittleEndian.Uint32(raw[len(raw)-crcSize:])
	got := crc32.ChecksumIEEE(body)
	if want != got {
		return nil, fmt.Errorf("manifest: crc mismatch want %08x got %08x", want, got)
	}
	r := bytes.NewReader(body)
	var magic uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if magic != Magic {
		return nil, fmt.Errorf("manifest: bad magic %08x", magic)
	}
	m := &Manifest{}
	if err := binary.Read(r, binary.LittleEndian, &m.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.Flags); err != nil {
		return nil, err
	}
	var algo [algoField]byte
	if _, err := io.ReadFull(r, algo[:]); err != nil {
		return nil, err
	}
	m.Algo = cstring(algo[:])
	if err := binary.Read(r, binary.LittleEndian, &m.ChunkSize); err != nil {
		return nil, err
	}
	var count uint32
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	if count > MaxChunks {
		return nil, fmt.Errorf("manifest: too many chunks %d", count)
	}
	if err := binary.Read(r, binary.LittleEndian, &m.TotalSize); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(r, m.ObjectID[:]); err != nil {
		return nil, err
	}
	if int(count)*entrySize != r.Len() {
		return nil, fmt.Errorf("manifest: entry bytes mismatch count=%d remain=%d", count, r.Len())
	}
	m.Chunks = make([]Entry, count)
	for i := uint32(0); i < count; i++ {
		if _, err := io.ReadFull(r, m.Chunks[i].ID[:]); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &m.Chunks[i].Size); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &m.Chunks[i].Offset); err != nil {
			return nil, err
		}
	}
	if err := Validate(m); err != nil {
		return nil, err
	}
	return m, nil
}

func cstring(b []byte) string {
	n := bytes.IndexByte(b, 0)
	if n < 0 {
		n = len(b)
	}
	return string(b[:n])
}
