package hashx

import (
	"hash"
	"io"
)

// Sink 是带字节计数的流式摘要。
type Sink struct {
	h hash.Hash
	n int64
}

// NewSink 用给定算法构造空 Sink。
func NewSink(a Algo) *Sink {
	if a == nil {
		a = Default()
	}
	return &Sink{h: a.New()}
}

// Write 实现 io.Writer。
func (s *Sink) Write(p []byte) (int, error) {
	n, err := s.h.Write(p)
	s.n += int64(n)
	return n, err
}

// WriteByte 写入单字节。
func (s *Sink) WriteByte(c byte) error {
	_, err := s.Write([]byte{c})
	return err
}

// ID 返回当前摘要（可重复调用，不复位）。
func (s *Sink) ID() ID {
	var id ID
	sum := s.h.Sum(nil)
	copy(id[:], sum)
	return id
}

// Bytes 返回已写入字节数。
func (s *Sink) Bytes() int64 {
	return s.n
}

// Reset 清空状态。
func (s *Sink) Reset() {
	s.h.Reset()
	s.n = 0
}

// Copy 把 r 全部写入 Sink，返回字节数。
func (s *Sink) Copy(r io.Reader) (int64, error) {
	return io.Copy(s, r)
}

// SumReader 读取 r 的全部内容并返回摘要与长度。
func SumReader(a Algo, r io.Reader) (ID, int64, error) {
	s := NewSink(a)
	n, err := s.Copy(r)
	if err != nil {
		return Zero, n, err
	}
	return s.ID(), n, nil
}

// EqualID 常量时间比较两个 ID。
func EqualID(a, b ID) bool {
	var v byte
	for i := 0; i < 32; i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
