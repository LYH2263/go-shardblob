package chunk

import (
	"fmt"
	"io"
)

// Assemble 按顺序把分片写入 w。pieces 必须按下标/偏移递增。
func Assemble(w io.Writer, pieces []Piece) (int64, error) {
	var n int64
	expect := uint64(0)
	for i, p := range pieces {
		if p.Offset != expect {
			return n, fmt.Errorf("chunk: assemble gap at %d: offset %d want %d", i, p.Offset, expect)
		}
		if p.Index != i {
			return n, fmt.Errorf("chunk: assemble index %d want %d", p.Index, i)
		}
		wn, err := w.Write(p.Data)
		n += int64(wn)
		if err != nil {
			return n, err
		}
		if wn != len(p.Data) {
			return n, io.ErrShortWrite
		}
		expect += uint64(len(p.Data))
	}
	return n, nil
}

// AssembleBytes 拼接为单块缓冲。
func AssembleBytes(pieces []Piece) ([]byte, error) {
	total := 0
	for _, p := range pieces {
		total += len(p.Data)
	}
	buf := make([]byte, 0, total)
	for i, p := range pieces {
		if p.Index != i {
			return nil, fmt.Errorf("chunk: assemble index %d want %d", p.Index, i)
		}
		buf = append(buf, p.Data...)
	}
	return buf, nil
}

// Writer 是按片追加的组装器。
type Writer struct {
	w      io.Writer
	n      int64
	index  int
	offset uint64
}

// NewWriter 包装底层 writer。
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Append 追加一片。
func (a *Writer) Append(p Piece) error {
	if p.Index != a.index {
		return fmt.Errorf("chunk: writer index %d want %d", p.Index, a.index)
	}
	if p.Offset != a.offset {
		return fmt.Errorf("chunk: writer offset %d want %d", p.Offset, a.offset)
	}
	wn, err := a.w.Write(p.Data)
	a.n += int64(wn)
	if err != nil {
		return err
	}
	if wn != len(p.Data) {
		return io.ErrShortWrite
	}
	a.index++
	a.offset += uint64(len(p.Data))
	return nil
}

// Bytes 已写出字节。
func (a *Writer) Bytes() int64 { return a.n }

// Count 已接受片数。
func (a *Writer) Count() int { return a.index }
