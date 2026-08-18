package chunk

import (
	"io"
	"sync"
)

const defaultCap = 64 << 10

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, defaultCap)
		return &b
	},
}

// GetBuf 从池中取一块至少 n 字节的缓冲（len==n）。
func GetBuf(n int) []byte {
	if n <= 0 {
		return nil
	}
	bp := bufPool.Get().(*[]byte)
	b := *bp
	if cap(b) < n {
		return make([]byte, n)
	}
	return b[:n]
}

// PutBuf 归还缓冲。过大的切片丢弃以免池膨胀。
func PutBuf(b []byte) {
	if b == nil || cap(b) > 8<<20 {
		return
	}
	b = b[:0]
	bufPool.Put(&b)
}

// Clone 拷贝 p。
func Clone(p []byte) []byte {
	if len(p) == 0 {
		if p == nil {
			return nil
		}
		return []byte{}
	}
	out := make([]byte, len(p))
	copy(out, p)
	return out
}

// ReaderAtBuf 从 ReaderAt 读取 [off, off+n)。
func ReaderAtBuf(r io.ReaderAt, off int64, n int) ([]byte, error) {
	if n <= 0 {
		return []byte{}, nil
	}
	buf := make([]byte, n)
	got, err := io.ReadFull(io.NewSectionReader(r, off, int64(n)), buf)
	if err != nil {
		return buf[:got], err
	}
	return buf, nil
}
