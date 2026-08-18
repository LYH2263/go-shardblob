package refcount

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

const (
	journalMagic   = uint32(0x4A465342) // "BSFJ"
	journalRecSize = 4 + 1 + 3 + 32 + 8 + 4
)

type Op uint8

const (
	OpInc        Op = 1
	OpDec        Op = 2
	OpCheckpoint Op = 3
)

// Journal 追加写引用计数变更，Open 时重放。
type Journal struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

func OpenJournal(path string) (*Journal, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	return &Journal{path: path, f: f}, nil
}

func (j *Journal) Append(op Op, id hashx.ID, n uint64) error {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	rec := encodeRec(op, id, n)
	if _, err := j.f.Write(rec); err != nil {
		return fmt.Errorf("refcount: journal write: %w", err)
	}
	return j.f.Sync()
}

func (j *Journal) Replay(t *Table) error {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if _, err := j.f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	buf := make([]byte, journalRecSize)
	for {
		_, err := io.ReadFull(j.f, buf)
		if err == io.EOF {
			return nil
		}
		if err == io.ErrUnexpectedEOF {
			// 尾部半写记录丢弃。
			return nil
		}
		if err != nil {
			return err
		}
		op, id, n, ok := decodeRec(buf)
		if !ok {
			return fmt.Errorf("refcount: journal crc/magic error")
		}
		switch op {
		case OpInc:
			t.Set(id, t.Get(id)+n)
		case OpDec:
			cur := t.Get(id)
			if cur < n {
				return fmt.Errorf("refcount: journal underflow")
			}
			t.Set(id, cur-n)
		case OpCheckpoint:
			// 仅作屏障，不改计数。
		default:
			return fmt.Errorf("refcount: unknown journal op %d", op)
		}
	}
}

func (j *Journal) Truncate() error {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.f.Truncate(0); err != nil {
		return err
	}
	if _, err := j.f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return j.f.Sync()
}

func (j *Journal) Close() error {
	if j == nil {
		return nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.f.Close()
}

func encodeRec(op Op, id hashx.ID, n uint64) []byte {
	buf := make([]byte, journalRecSize)
	binary.LittleEndian.PutUint32(buf[0:4], journalMagic)
	buf[4] = byte(op)
	copy(buf[8:40], id[:])
	binary.LittleEndian.PutUint64(buf[40:48], n)
	sum := crc32.ChecksumIEEE(buf[:48])
	binary.LittleEndian.PutUint32(buf[48:52], sum)
	return buf
}

func decodeRec(buf []byte) (Op, hashx.ID, uint64, bool) {
	if len(buf) < journalRecSize {
		return 0, hashx.Zero, 0, false
	}
	if binary.LittleEndian.Uint32(buf[0:4]) != journalMagic {
		return 0, hashx.Zero, 0, false
	}
	sum := crc32.ChecksumIEEE(buf[:48])
	want := binary.LittleEndian.Uint32(buf[48:52])
	if sum != want {
		return 0, hashx.Zero, 0, false
	}
	var id hashx.ID
	copy(id[:], buf[8:40])
	n := binary.LittleEndian.Uint64(buf[40:48])
	return Op(buf[4]), id, n, true
}
