package zico

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type Pool struct {
	capacity      			uint32
	workers       			[]*Worker
	runnings      			uint32
	lastWorkerIndex			uint32
	available     			chan struct{}
	close       			chan struct{}
	mutex         			sync.Mutex
	waitWorkers             sync.Pool
	ctx                     context.Context
	cancel                  context.CancelFunc
}

func NewPool(cap uint32) (pool *Pool) {
	pool = &Pool{
		capacity:    cap,
		runnings:    0,
		workers:     make([]*Worker, 0, cap),
		mutex:       sync.Mutex{},
		available:   make(chan struct{}),
		close:       make(chan struct{}, 1),
		waitWorkers: sync.Pool{},
	}
	pool.ctx,pool.cancel = context.WithCancel(context.Background())
	pool.waitWorkers.New = nil

	return
}

func (p *Pool) SubmitWork(f func()) {
	// workers is idle
	if len(p.close) == 1 {
		fmt.Println("pool closed")
		return
	}

	worker := p.getWorker()
	worker.task(f)
}

func (p *Pool) getWorker() (worker *Worker) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if atomic.LoadUint32(&p.runnings) == p.capacity {
		<-p.available
		worker = p.getWaitWorker()

		if worker == nil {
			p.mutex.Unlock()
			worker = p.getWorker()
		}  else {
			atomic.AddUint32(&p.runnings, 1)
			atomic.StoreUint32(&worker.isRunning, 1)
		}
	} else {
		atomic.AddUint32(&p.runnings, 1)
		l := len(p.workers)

		if uint32(l) < p.capacity {
			worker = &Worker{
				index: uint32(l),
				isRunning: 1,
				pool: p,
			}
			p.workers = p.workers[:l+1]
			p.workers[l] = worker
		} else {
			worker = p.getWaitWorker()
		}
	}

	return worker
}

func (p *Pool) getWaitWorker() (worker *Worker) {
	if worker = p.waitWorkers.Get().(*Worker); worker == nil {
		// 有可能被gc回收掉临时池里的可用worker，导致worker为nil
		for _,w := range p.workers {
			if w.isRunning == 0 {
				worker = w
			}
		}
	}

	return
}

func (p *Pool) Close() {
	p.close<- struct{}{}
}