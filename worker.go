package zico

import (
	"fmt"
	"runtime"
	"sync/atomic"
)

type Worker struct {
	index        uint32
	isRunning    uint32
	pool         *Pool
}

func (w *Worker) task(f func()) {
	go func() {
		fmt.Println(w.index)
		defer func(p *Pool) {
			if atomic.LoadUint32(&p.runnings) == p.capacity {
				p.available<- struct{}{}
			}
			atomic.AddUint32(&p.runnings, ^uint32(1-1))
			atomic.StoreUint32(&w.isRunning, 0)
			w.pool.waitWorkers.Put(w)

			if _p := recover(); _p != nil {
				_buf := make([]byte, 0, 4096)
				n := runtime.Stack(_buf, false)
				fmt.Printf("call stack: %s\n", _buf[:n])
			}
		}(w.pool)
		f()
	}()
}