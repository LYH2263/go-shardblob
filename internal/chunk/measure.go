package chunk

import "fmt"

// Measure 描述一次切分的统计，便于埋点：尾块长度、片数、是否 CDC。
type Measure struct {
	Pieces    int
	Total     uint64
	LastSize  int
	AvgSize   int
	MaxPiece  int
	MinPiece  int
	CDC       bool
	ShortTail bool
}

func (m Measure) String() string {
	return fmt.Sprintf("pieces=%d total=%d last=%d short_tail=%v cdc=%v",
		m.Pieces, m.Total, m.LastSize, m.ShortTail, m.CDC)
}

// MeasurePieces 根据已切分结果汇总。
func MeasurePieces(pieces []Piece, pol Policy) Measure {
	pol = Normalize(pol)
	m := Measure{
		Pieces:   len(pieces),
		AvgSize:  pol.Avg,
		CDC:      pol.Mode == ModeCDC,
		MinPiece: int(^uint(0) >> 1),
	}
	if len(pieces) == 0 {
		m.MinPiece = 0
		return m
	}
	for _, p := range pieces {
		n := len(p.Data)
		m.Total += uint64(n)
		if n > m.MaxPiece {
			m.MaxPiece = n
		}
		if n < m.MinPiece {
			m.MinPiece = n
		}
	}
	m.LastSize = len(pieces[len(pieces)-1].Data)
	if pol.Mode == ModeFixed {
		m.ShortTail = m.LastSize > 0 && m.LastSize < pol.Avg
	}
	return m
}
