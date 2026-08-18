package chunk

import (
	"fmt"
	"math/bits"
)

const (
	MinAvgSize     = 1
	MaxAvgSize     = 4 << 20
	DefaultAvgSize = 64 << 10
)

// Mode 分片策略。
type Mode int

const (
	ModeFixed Mode = iota
	ModeCDC
)

func (m Mode) String() string {
	switch m {
	case ModeFixed:
		return "fixed"
	case ModeCDC:
		return "cdc"
	default:
		return fmt.Sprintf("mode(%d)", int(m))
	}
}

// Policy 描述如何把字节流切成片。尾块允许短于 Avg。
type Policy struct {
	Mode Mode
	Avg  int
	Min  int
	Max  int
}

// DefaultPolicy 定长 64KiB。
func DefaultPolicy() Policy {
	return Normalize(Policy{Mode: ModeFixed, Avg: DefaultAvgSize})
}

// CDCPolicy 内容定义分片，平均大小 avg。
func CDCPolicy(avg int) Policy {
	return Normalize(Policy{Mode: ModeCDC, Avg: avg})
}

// Normalize 校正边界：Avg 对齐到 2 的幂，Min=Avg/4，Max=Avg*4。
func Normalize(p Policy) Policy {
	avg := p.Avg
	if avg <= 0 {
		avg = DefaultAvgSize
	}
	if avg < MinAvgSize {
		avg = MinAvgSize
	}
	if avg > MaxAvgSize {
		avg = MaxAvgSize
	}
	// 向上/向下靠到 2 的幂，取更近者。
	if avg&(avg-1) != 0 {
		hi := 1 << bits.Len(uint(avg))
		lo := hi >> 1
		if avg-lo <= hi-avg {
			avg = lo
		} else {
			avg = hi
		}
		if avg < MinAvgSize {
			avg = MinAvgSize
		}
		if avg > MaxAvgSize {
			avg = MaxAvgSize
		}
	}
	min := p.Min
	max := p.Max
	if min <= 0 {
		min = avg / 4
	}
	if max <= 0 {
		max = avg * 4
	}
	if min < 1 {
		min = 1
	}
	if min > avg {
		min = avg
	}
	if max < avg {
		max = avg
	}
	if max > MaxAvgSize*4 {
		max = MaxAvgSize * 4
	}
	return Policy{Mode: p.Mode, Avg: avg, Min: min, Max: max}
}

// Valid 报告策略是否可用。
func (p Policy) Valid() error {
	n := Normalize(p)
	if n.Avg < MinAvgSize || n.Avg > MaxAvgSize {
		return fmt.Errorf("chunk: avg size %d out of range", p.Avg)
	}
	if n.Min < 1 || n.Min > n.Max {
		return fmt.Errorf("chunk: invalid min/max %d/%d", n.Min, n.Max)
	}
	return nil
}

// Mask 返回 CDC 用的低位掩码（约 Avg 字节触发一次边界）。
func (p Policy) Mask() uint64 {
	n := Normalize(p)
	bitsN := bits.TrailingZeros(uint(n.Avg))
	if bitsN <= 0 {
		bitsN = 16
	}
	return uint64(1<<bitsN) - 1
}
