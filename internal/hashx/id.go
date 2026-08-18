package hashx

import (
	"encoding/hex"
	"fmt"
)

// ID 是固定 32 字节的内容摘要（SHA-256 / SHA-512/256 等）。
type ID [32]byte

// Zero 全零 ID，表示未赋值。
var Zero ID

// Hex 返回小写十六进制表示。
func (id ID) Hex() string {
	return hex.EncodeToString(id[:])
}

// String 与 Hex 相同，便于日志。
func (id ID) String() string {
	return id.Hex()
}

// IsZero 报告是否为全零。
func (id ID) IsZero() bool {
	return id == Zero
}

// Equal 比较两个 ID。
func (id ID) Equal(other ID) bool {
	return id == other
}

// Prefix 返回前 n 个字节的拷贝；n 超过 32 时截到 32。
func (id ID) Prefix(n int) []byte {
	if n <= 0 {
		return nil
	}
	if n > 32 {
		n = 32
	}
	out := make([]byte, n)
	copy(out, id[:n])
	return out
}

// ParseHex 解析 64 字符十六进制为 ID。
func ParseHex(s string) (ID, error) {
	var id ID
	if len(s) != 64 {
		return id, fmt.Errorf("hashx: id hex length %d, want 64", len(s))
	}
	raw, err := hex.DecodeString(s)
	if err != nil {
		return id, fmt.Errorf("hashx: decode hex: %w", err)
	}
	copy(id[:], raw)
	return id, nil
}

// FromSlice 将恰好 32 字节拷贝为 ID。
func FromSlice(b []byte) (ID, error) {
	var id ID
	if len(b) != 32 {
		return id, fmt.Errorf("hashx: id length %d, want 32", len(b))
	}
	copy(id[:], b)
	return id, nil
}

// MustFromSlice 与 FromSlice 相同，长度不对时 panic。
func MustFromSlice(b []byte) ID {
	id, err := FromSlice(b)
	if err != nil {
		panic(err)
	}
	return id
}

// Compare 返回 -1/0/1，按字典序。
func Compare(a, b ID) int {
	for i := 0; i < 32; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// Less 报告 a 是否字典序小于 b。
func Less(a, b ID) bool {
	return Compare(a, b) < 0
}
