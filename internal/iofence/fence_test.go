package iofence

import (
	"sync"
	"testing"
	"time"

	"github.com/LYH2263/go-shardblob/internal/hashx"
)

func TestSharedBlocksExclusive(t *testing.T) {
	f := New()
	rel := f.BeginShared()
	ch := make(chan struct{})
	go func() {
		done := f.BeginExclusive()
		close(ch)
		done()
	}()
	select {
	case <-ch:
		t.Fatal("exclusive acquired while shared held")
	case <-time.After(50 * time.Millisecond):
	}
	rel()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("exclusive did not proceed")
	}
}

func TestObjectLockSerial(t *testing.T) {
	f := New()
	id := hashx.SumSHA256([]byte("o"))
	var order []int
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)
	block := make(chan struct{})
	go func() {
		defer wg.Done()
		u := f.LockObject(id)
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		<-block
		u()
	}()
	time.Sleep(20 * time.Millisecond)
	go func() {
		defer wg.Done()
		u := f.LockObject(id)
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		u()
	}()
	time.Sleep(20 * time.Millisecond)
	close(block)
	wg.Wait()
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("order %v", order)
	}
}
