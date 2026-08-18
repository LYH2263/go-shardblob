package iofence

import (
	"sync"
	"sync/atomic"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

// Fence 协调 Put/Get/Delete 与 GC。
// 共享侧（IO）持读锁，GC 持写锁，从而 GC 不会删正在写入或读取的分片。
type Fence struct {
	io     sync.RWMutex
	active atomic.Int64
	om     sync.Mutex
	objs   map[hashx.ID]*objLock
}

type objLock struct {
	mu      sync.Mutex
	waiters int
}

func New() *Fence {
	return &Fence{objs: make(map[hashx.ID]*objLock)}
}

// BeginShared 开始一次读写操作。返回的释放函数必须调用。
func (f *Fence) BeginShared() func() {
	f.io.RLock()
	f.active.Add(1)
	return func() {
		f.active.Add(-1)
		f.io.RUnlock()
	}
}

// BeginExclusive 开始 GC。会等待所有共享操作结束。
func (f *Fence) BeginExclusive() func() {
	f.io.Lock()
	return func() {
		f.io.Unlock()
	}
}

// InFlight 当前共享操作数。
func (f *Fence) InFlight() int64 {
	return f.active.Load()
}

// TryExclusive 尝试获取 GC 锁，失败则 ok=false。
func (f *Fence) TryExclusive() (done func(), ok bool) {
	if f.active.Load() != 0 {
		return nil, false
	}
	// 仍可能与即将进入的共享操作竞争；真正互斥靠 io.Lock。
	f.io.Lock()
	return func() { f.io.Unlock() }, true
}
