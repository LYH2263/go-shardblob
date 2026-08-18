package chunk

import (
	"errors"
	"io"
)

// Piece 一片已切出的数据。Data 归调用方所有。
type Piece struct {
	Index  int
	Offset uint64
	Data   []byte
}

// Splitter 从 Reader 按策略切分。Next 在结束后返回 io.EOF。
type Splitter struct {
	r      io.Reader
	pol    Policy
	fixed  []byte
	carry  []byte
	gear   *Gear
	offset uint64
	index  int
	eof    bool
	err    error
}

// NewSplitter 构造切分器。空流不产生任何 Piece。
func NewSplitter(r io.Reader, pol Policy) *Splitter {
	pol = Normalize(pol)
	s := &Splitter{r: r, pol: pol}
	if pol.Mode == ModeCDC {
		s.gear = NewGear(pol)
		s.carry = make([]byte, 0, pol.Avg)
	} else {
		s.fixed = make([]byte, pol.Avg)
	}
	return s
}

// Next 返回下一片。尾块可以短于 Avg。
func (s *Splitter) Next() (Piece, error) {
	if s.err != nil {
		return Piece{}, s.err
	}
	if s.eof {
		s.err = io.EOF
		return Piece{}, io.EOF
	}
	if s.pol.Mode == ModeCDC {
		return s.nextCDC()
	}
	return s.nextFixed()
}

func (s *Splitter) nextFixed() (Piece, error) {
	n, err := io.ReadFull(s.r, s.fixed)
	if n == 0 && (errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)) {
		s.eof = true
		s.err = io.EOF
		return Piece{}, io.EOF
	}
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		s.err = err
		return Piece{}, err
	}
	data := Clone(s.fixed[:n])
	p := Piece{Index: s.index, Offset: s.offset, Data: data}
	s.offset += uint64(n)
	s.index++
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || n < len(s.fixed) {
		s.eof = true
	}
	return p, nil
}

func (s *Splitter) emit(data []byte) Piece {
	p := Piece{Index: s.index, Offset: s.offset, Data: data}
	s.offset += uint64(len(data))
	s.index++
	return p
}

func (s *Splitter) nextCDC() (Piece, error) {
	s.gear.Reset()
	buf := append([]byte(nil), s.carry...)
	s.carry = s.carry[:0]
	for i := 0; i < len(buf); i++ {
		if s.gear.Push(buf[i]) {
			data := Clone(buf[:i+1])
			s.carry = append(s.carry[:0], buf[i+1:]...)
			return s.emit(data), nil
		}
	}

	tmp := GetBuf(32 << 10)
	defer PutBuf(tmp)

	for {
		if s.eof {
			if len(buf) == 0 {
				s.err = io.EOF
				return Piece{}, io.EOF
			}
			data := Clone(buf)
			return s.emit(data), nil
		}
		n, err := s.r.Read(tmp)
		if n > 0 {
			for i := 0; i < n; i++ {
				buf = append(buf, tmp[i])
				if s.gear.Push(tmp[i]) {
					data := Clone(buf)
					s.carry = append(s.carry[:0], tmp[i+1:n]...)
					if errors.Is(err, io.EOF) && i+1 == n && len(s.carry) == 0 {
						s.eof = true
					}
					return s.emit(data), nil
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.eof = true
				if len(buf) == 0 {
					s.err = io.EOF
					return Piece{}, io.EOF
				}
				data := Clone(buf)
				return s.emit(data), nil
			}
			s.err = err
			return Piece{}, err
		}
	}
}

// SplitAll 一次切完（小对象测试用）。
func SplitAll(r io.Reader, pol Policy) ([]Piece, error) {
	sp := NewSplitter(r, pol)
	var out []Piece
	for {
		p, err := sp.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return out, err
		}
		out = append(out, p)
	}
}
