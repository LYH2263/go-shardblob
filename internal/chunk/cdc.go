package chunk

import (
	"hash/fnv"
	"math/bits"
)

const gearWindow = 64

var gearTable [256]uint64

func init() {
	// 用 splitmix64 生成稳定的 Gear 表，保证 CDC 边界跨进程可复现。
	var x uint64 = 0xA5A5A5A5C001D00D
	for i := 0; i < 256; i++ {
		x += 0x9E3779B97F4A7C15
		z := x
		z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
		z = (z ^ (z >> 27)) * 0x94D049BB133111EB
		gearTable[i] = z ^ (z >> 31)
	}
}

// Gear 实现 FastCDC 风格的滚动哈希，用于内容定义切分。
type Gear struct {
	hash uint64
	n    int
	min  int
	max  int
	mask uint64
}

// NewGear 按策略构造滚动窗口。
func NewGear(p Policy) *Gear {
	p = Normalize(p)
	return &Gear{
		min:  p.Min,
		max:  p.Max,
		mask: p.Mask(),
	}
}

// Reset 清空滚动状态。
func (g *Gear) Reset() {
	g.hash = 0
	g.n = 0
}

// Push 喂入一字节。返回是否应在该字节之后切分（含该字节）。
func (g *Gear) Push(b byte) bool {
	g.hash = (g.hash << 1) + gearTable[b]
	g.n++
	if g.n < g.min {
		return false
	}
	if g.n >= g.max {
		return true
	}
	return g.hash&g.mask == 0
}

// Size 当前窗口累计字节。
func (g *Gear) Size() int { return g.n }

// Hash 当前滚动值。
func (g *Gear) Hash() uint64 { return g.hash }

// FindBoundary 在 p 中找第一处切分点（相对 p 的结束下标，不含下一片）。
// 若整段都不够切，返回 len(p) 与 false。
func FindBoundary(p []byte, pol Policy) (cut int, ok bool) {
	g := NewGear(pol)
	for i, b := range p {
		if g.Push(b) {
			return i + 1, true
		}
	}
	return len(p), false
}

// Fingerprint 计算分片的 FNV-1a 64 位指纹，便于调试/统计，不参与寻址。
func Fingerprint(p []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(p)
	return h.Sum64()
}

// AlignPow2 把 n 调整为不小于 min 的 2 的幂。
func AlignPow2(n, min int) int {
	if n < min {
		n = min
	}
	if n <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(n-1))
}
