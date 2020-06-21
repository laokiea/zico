package zico

import (
	"fmt"
	"runtime"
	"sync/atomic"
)

type Worker struct {
	index        	uint32
	isRunning    	uint32
	pool         	*Pool
	tasks        	chan func()
}

const workerTasksChanSize uint32 = 10

func (w *Worker) task() {
	go func() {
		fmt.Println("index:", w.index)
		defer func(p *Pool) {
			if atomic.LoadUint32(&p.runnings) == p.capacity {
				w.pool.mutex.Lock()
				p.available<- struct{}{}
				w.pool.mutex.Unlock()
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

		for f := range w.tasks {
			if f == nil {
				return
			}
			f()
			if l := len(w.tasks); l == 0 {
				return
			}
		}
	}()
}

func NewWorker(p *Pool, index uint32) (worker *Worker) {
	worker = &Worker{
		index: index,
		isRunning: 1,
		pool: p,
	}
	worker.tasks = make(chan func(), workerTasksChanSize)
	return
}