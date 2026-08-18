package hashx

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

type sha256Algo struct{}

func (sha256Algo) Name() string { return NameSHA256 }

func (sha256Algo) New() hash.Hash { return sha256.New() }

func (sha256Algo) Sum(data []byte) ID {
	sum := sha256.Sum256(data)
	return ID(sum)
}

type sha512256Algo struct{}

func (sha512256Algo) Name() string { return NameSHA512256 }

func (sha512256Algo) New() hash.Hash { return sha512.New512_256() }

func (sha512256Algo) Sum(data []byte) ID {
	sum := sha512.Sum512_256(data)
	return ID(sum)
}

// SumSHA256 是 SHA-256 的便捷函数。
func SumSHA256(data []byte) ID {
	return sha256Algo{}.Sum(data)
}

// SumSHA512256 是 SHA-512/256 的便捷函数。
func SumSHA512256(data []byte) ID {
	return sha512256Algo{}.Sum(data)
}
