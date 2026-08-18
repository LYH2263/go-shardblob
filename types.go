package shardblob

import (
	"encoding/hex"
	"fmt"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

// ObjectID 对象内容寻址 ID（整段明文的域分离摘要）。
type ObjectID [32]byte

func (id ObjectID) String() string { return hex.EncodeToString(id[:]) }

func (id ObjectID) IsZero() bool { return id == ObjectID{} }

func (id ObjectID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *ObjectID) UnmarshalText(b []byte) error {
	got, err := ParseObjectID(string(b))
	if err != nil {
		return err
	}
	*id = got
	return nil
}

// ParseObjectID 解析 64 位 hex。
func ParseObjectID(s string) (ObjectID, error) {
	hid, err := hashx.ParseHex(s)
	if err != nil {
		return ObjectID{}, fmt.Errorf("shardblob: %w", err)
	}
	return ObjectID(hid), nil
}

func toHX(id ObjectID) hashx.ID { return hashx.ID(id) }

func fromHX(id hashx.ID) ObjectID { return ObjectID(id) }

// Info 对象元数据。
type Info struct {
	ID       ObjectID
	Size     uint64
	Chunks   int
	UseCount uint32
}

// Stats 仓占用。
type Stats struct {
	Objects     int
	Chunks      int
	Logical     uint64
	UniqueBytes int64
}

// DedupRatio 逻辑字节 / 唯一分片字节；无数据时为 1。
func (s Stats) DedupRatio() float64 {
	if s.UniqueBytes <= 0 {
		if s.Logical == 0 {
			return 1
		}
		return 0
	}
	return float64(s.Logical) / float64(s.UniqueBytes)
}
