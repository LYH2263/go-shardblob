package index

import (
	"fmt"
	"sync"
	"time"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

// Record 对象 ID 到清单元数据的映射。只有完整清单才会进入索引。
type Record struct {
	ID        hashx.ID
	UseCount  uint32
	Size      uint64
	Chunks    uint32
	CreatedAt int64
}

// Index 内存对象索引。
type Index struct {
	mu      sync.RWMutex
	objects map[hashx.ID]Record
}

func New() *Index {
	return &Index{objects: make(map[hashx.ID]Record)}
}

func (x *Index) Get(id hashx.ID) (Record, bool) {
	x.mu.RLock()
	defer x.mu.RUnlock()
	r, ok := x.objects[id]
	return r, ok
}

func (x *Index) Put(r Record) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if r.CreatedAt == 0 {
		if old, ok := x.objects[r.ID]; ok && old.CreatedAt != 0 {
			r.CreatedAt = old.CreatedAt
		} else {
			r.CreatedAt = time.Now().UnixNano()
		}
	}
	x.objects[r.ID] = r
}

func (x *Index) Delete(id hashx.ID) {
	x.mu.Lock()
	defer x.mu.Unlock()
	delete(x.objects, id)
}

func (x *Index) Has(id hashx.ID) bool {
	_, ok := x.Get(id)
	return ok
}

func (x *Index) Len() int {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return len(x.objects)
}

func (x *Index) List() []Record {
	x.mu.RLock()
	defer x.mu.RUnlock()
	out := make([]Record, 0, len(x.objects))
	for _, r := range x.objects {
		out = append(out, r)
	}
	return out
}

func (x *Index) LoadAll(recs []Record) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.objects = make(map[hashx.ID]Record, len(recs))
	for _, r := range recs {
		x.objects[r.ID] = r
	}
}

func (x *Index) IncUse(id hashx.ID) (Record, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	r, ok := x.objects[id]
	if !ok {
		return Record{}, fmt.Errorf("index: object %s not found", id.Hex())
	}
	r.UseCount++
	x.objects[id] = r
	return r, nil
}

func (x *Index) DecUse(id hashx.ID) (Record, bool, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	r, ok := x.objects[id]
	if !ok {
		return Record{}, false, fmt.Errorf("index: object %s not found", id.Hex())
	}
	if r.UseCount == 0 {
		return r, true, fmt.Errorf("index: use count already zero")
	}
	r.UseCount--
	gone := r.UseCount == 0
	if gone {
		delete(x.objects, id)
	} else {
		x.objects[id] = r
	}
	return r, gone, nil
}

func (x *Index) TotalBytes() uint64 {
	x.mu.RLock()
	defer x.mu.RUnlock()
	var n uint64
	for _, r := range x.objects {
		n += r.Size
	}
	return n
}
