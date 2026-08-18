package hashx

import (
	"fmt"
	"hash"
	"strings"
	"sync"
)

const (
	NameSHA256    = "sha256"
	NameSHA512256 = "sha512_256"
)

// Algo 摘要算法。所有实现输出 32 字节。
type Algo interface {
	Name() string
	New() hash.Hash
	Sum(data []byte) ID
}

var (
	regMu sync.RWMutex
	reg   = map[string]Algo{
		NameSHA256:    sha256Algo{},
		NameSHA512256: sha512256Algo{},
	}
)

// Lookup 按名称查找算法（大小写不敏感）。
func Lookup(name string) (Algo, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	regMu.RLock()
	a, ok := reg[key]
	regMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("hashx: unknown algorithm %q", name)
	}
	return a, nil
}

// Default 返回 SHA-256。
func Default() Algo {
	return sha256Algo{}
}

// Register 注册自定义算法；name 必须非空。测试或扩展用。
func Register(name string, a Algo) error {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" || a == nil {
		return fmt.Errorf("hashx: invalid register")
	}
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := reg[key]; exists {
		return fmt.Errorf("hashx: algorithm %q already registered", key)
	}
	reg[key] = a
	return nil
}

// Names 返回已注册算法名（未排序）。
func Names() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(reg))
	for k := range reg {
		out = append(out, k)
	}
	return out
}
