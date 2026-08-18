package iofence

import "github.com/LYH2263/go-shardblob/internal/hashx"

// LockObject 按对象 ID 串行化索引更新（重复 Put / Delete）。
func (f *Fence) LockObject(id hashx.ID) func() {
	f.om.Lock()
	ol := f.objs[id]
	if ol == nil {
		ol = &objLock{}
		f.objs[id] = ol
	}
	ol.waiters++
	f.om.Unlock()
	ol.mu.Lock()
	return func() {
		ol.mu.Unlock()
		f.om.Lock()
		ol.waiters--
		if ol.waiters == 0 {
			delete(f.objs, id)
		}
		f.om.Unlock()
	}
}

// LockedCount 当前持有或等待的对象锁数量（测试用）。
func (f *Fence) LockedCount() int {
	f.om.Lock()
	defer f.om.Unlock()
	return len(f.objs)
}
