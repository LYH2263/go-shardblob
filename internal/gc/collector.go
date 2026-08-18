package gc

import (
	"fmt"

	"github.com/LYH2263/go-shardblob/internal/blobstore"
	"github.com/LYH2263/go-shardblob/internal/hashx"
	"github.com/LYH2263/go-shardblob/internal/refcount"
)

// Stats GC 结果。
type Stats struct {
	Scanned   int
	Deleted   int
	Retained  int
	BytesFree int64
	Skipped   int
}

func (s Stats) String() string {
	return fmt.Sprintf("scanned=%d deleted=%d retained=%d bytes_free=%d skipped=%d",
		s.Scanned, s.Deleted, s.Retained, s.BytesFree, s.Skipped)
}

// Collector 扫描后端，删除 refcount==0 的分片。不得删除 refcount>0 的分片。
type Collector struct {
	Blobs blobstore.Backend
	Refs  *refcount.Table
	Dry   bool
}

// New 构造收集器。
func New(blobs blobstore.Backend, refs *refcount.Table) *Collector {
	return &Collector{Blobs: blobs, Refs: refs}
}

// Run 执行一次回收。调用方须持有排他栅栏，避免与 Put 竞态。
func (c *Collector) Run() (Stats, error) {
	var st Stats
	if c.Blobs == nil || c.Refs == nil {
		return st, fmt.Errorf("gc: nil deps")
	}
	err := c.Blobs.List(func(id hashx.ID) error {
		st.Scanned++
		n := c.Refs.Get(id)
		if n > 0 {
			st.Retained++
			return nil
		}
		data, gerr := c.Blobs.Get(id)
		if gerr != nil {
			st.Skipped++
			return nil
		}
		sz := int64(len(data))
		if c.Dry {
			st.Deleted++
			st.BytesFree += sz
			return nil
		}
		// 删除前再次确认，避免误删刚被引用的分片。
		if c.Refs.Get(id) > 0 {
			st.Retained++
			return nil
		}
		if err := c.Blobs.Delete(id); err != nil {
			return err
		}
		st.Deleted++
		st.BytesFree += sz
		return nil
	})
	return st, err
}

// SweepIDs 只回收给定 ID 中计数为 0 的项。
func SweepIDs(blobs blobstore.Backend, refs *refcount.Table, ids []hashx.ID) (Stats, error) {
	var st Stats
	for _, id := range ids {
		st.Scanned++
		if refs.Get(id) > 0 {
			st.Retained++
			continue
		}
		data, err := blobs.Get(id)
		if err != nil {
			st.Skipped++
			continue
		}
		if err := blobs.Delete(id); err != nil {
			return st, err
		}
		st.Deleted++
		st.BytesFree += int64(len(data))
	}
	return st, nil
}
