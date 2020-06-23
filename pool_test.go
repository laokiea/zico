package zico

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestPool_SubmitWork(t *testing.T) {
	pool := NewPool(3)
	pool.WithSyncPool(false)
	var wg sync.WaitGroup
	for i := 0;i < 5;i++ {
		wg.Add(1)
		pool.SubmitWork(func() {
			defer wg.Done()
			time.Sleep(time.Second * 5)
			fmt.Println("hello")
		}, func() {
			time.Sleep(time.Second * 5)
			fmt.Println("world")
		})
	}
	wg.Wait()
}
