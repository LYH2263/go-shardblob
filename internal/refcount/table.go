package refcount

import (
	"fmt"
	"sync"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

// Table 分片引用计数。GC 只能回收计数为 0 的分片。
type Table struct {
	mu     sync.Mutex
	counts map[hashx.ID]uint64
	dirty  int
	j      *Journal
}

func NewTable() *Table {
	return &Table{counts: make(map[hashx.ID]uint64)}
}

// AttachJournal 绑定崩溃恢复日志；之后 Add/Sub 会追加记录。
func (t *Table) AttachJournal(j *Journal) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.j = j
}

func (t *Table) Add(id hashx.ID, n uint64) error {
	if n == 0 {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts[id] += n
	t.dirty++
	if t.j != nil {
		if err := t.j.Append(OpInc, id, n); err != nil {
			t.counts[id] -= n
			if t.counts[id] == 0 {
				delete(t.counts, id)
			}
			t.dirty--
			return err
		}
	}
	return nil
}

func (t *Table) Sub(id hashx.ID, n uint64) error {
	if n == 0 {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	cur, ok := t.counts[id]
	if !ok || cur < n {
		return fmt.Errorf("refcount: underflow id=%s cur=%d sub=%d", id.Hex(), cur, n)
	}
	cur -= n
	if cur == 0 {
		delete(t.counts, id)
	} else {
		t.counts[id] = cur
	}
	t.dirty++
	if t.j != nil {
		if err := t.j.Append(OpDec, id, n); err != nil {
			t.counts[id] = cur + n
			t.dirty--
			return err
		}
	}
	return nil
}

func (t *Table) Get(id hashx.ID) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.counts[id]
}

func (t *Table) Set(id hashx.ID, n uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n == 0 {
		delete(t.counts, id)
	} else {
		t.counts[id] = n
	}
	t.dirty++
}

func (t *Table) Live() []hashx.ID {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]hashx.ID, 0, len(t.counts))
	for id, n := range t.counts {
		if n > 0 {
			out = append(out, id)
		}
	}
	return out
}

func (t *Table) ZeroOrMissing(id hashx.ID) bool {
	return t.Get(id) == 0
}

func (t *Table) Snapshot() map[hashx.ID]uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[hashx.ID]uint64, len(t.counts))
	for k, v := range t.counts {
		out[k] = v
	}
	return out
}

func (t *Table) LoadMap(m map[hashx.ID]uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.counts = make(map[hashx.ID]uint64, len(m))
	for k, v := range m {
		if v > 0 {
			t.counts[k] = v
		}
	}
	t.dirty = 0
}

func (t *Table) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.counts)
}

func (t *Table) Dirty() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.dirty
}

func (t *Table) MarkClean() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.dirty = 0
}

// AddMany 批量为分片 +1。
func (t *Table) AddMany(ids []hashx.ID) error {
	seen := make(map[hashx.ID]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if err := t.Add(id, 1); err != nil {
			return err
		}
	}
	return nil
}

// SubMany 批量为分片 -1。
func (t *Table) SubMany(ids []hashx.ID) error {
	for _, id := range ids {
		if err := t.Sub(id, 1); err != nil {
			return err
		}
	}
	return nil
}
