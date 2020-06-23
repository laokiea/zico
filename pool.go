package zico

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"


	log "github.com/sirupsen/logrus"
)

type PoolStatus struct {
	ExecTasksNum            uint32 `json:"exec_tasks_num"`
	ResizeTimes             uint8  `json:"resize_times"`
	MaxWorkersNum           uint32 `json:"max_worker_num"`
}

type Pool struct {
	capacity      			uint32
	workers       			[]*Worker
	runnings      			uint32
	lastWorkerIndex			uint32
	available     			chan struct{}
	close       			chan struct{}
	mutex         			sync.Mutex
	ctx                     context.Context
	cancel                  context.CancelFunc
	waitWorkers             sync.Pool
	withSyncPool            bool
	withLogPoolstatus       bool
	poolStatus              *PoolStatus
	logStatusTick           uint8
}

var ticker *time.Ticker

func NewPool(cap uint32) (pool *Pool) {
	pool = &Pool{
		capacity:      cap,
		runnings:      0,
		workers:       make([]*Worker, 0, cap),
		mutex:         sync.Mutex{},
		available:     make(chan struct{}),
		close:         make(chan struct{}, 1),
		poolStatus:    new(PoolStatus),
	}

	pool.poolStatus.MaxWorkersNum = cap
	pool.ctx,pool.cancel = context.WithCancel(context.Background())
	pool.waitWorkers.New = nil

	pool.logStatusTick = func(cap uint32) uint8 {
		i := 1
		for {
			if uint32(1 << i) > cap {
				break
			}
			i++
		}
		return uint8(1 << i)
	} (cap)

	go pool.waitQuitSignal()
	go pool.startLogStatusTicker()

	return
}

func (p *Pool) SubmitWork(f ...func()) {
	// workers is idle
	if len(p.close) == 1 {
		fmt.Println("pool closed")
		return
	}

	worker := p.getWorker()
	for _,_f := range f {
		worker.tasks<- _f
 	}
	worker.task()
}

func (p *Pool) WithSyncPool(with bool) {
	p.withSyncPool = with
	if p.withSyncPool == true {
		p.waitWorkers = sync.Pool{}
	}
}

func (p *Pool) WithLogPoolStatus(with bool) {
	p.withLogPoolstatus = with;
}

func (p *Pool) getWorker() (worker *Worker) {
	if atomic.LoadUint32(&p.runnings) == p.capacity {
		<-p.available
		worker = p.getAvailableWorker()
		atomic.AddUint32(&p.runnings, 1)
		atomic.StoreUint32(&worker.isRunning, 1)
	} else {
		atomic.AddUint32(&p.runnings, 1)
		l := len(p.workers)

		if uint32(l) < p.capacity {
			defer p.mutex.Unlock()
			p.mutex.Lock()
			worker = NewWorker(p, uint32(l))
			p.workers = p.workers[:l+1]
			p.workers[l] = worker
		} else {
			worker = p.getAvailableWorker()
		}
	}

	return
}

func (p *Pool) getAvailableWorker() (worker *Worker) {
	if p.withSyncPool == true {
		worker = p.getWaitWorker()
	} else {
		worker = p.getAvailableWorker()
	}

	return
}

func (p *Pool) getWaitWorker() (worker *Worker) {
	if w := p.waitWorkers.Get(); w == nil {
		// 有可能被gc回收掉临时池里的可用worker，导致worker为nil
		worker = p.getOriginWorker()
	} else {
		worker = w.(*Worker)
	}

	return
}

func (p *Pool) getOriginWorker() (worker *Worker) {
	for _,_w := range p.workers {
		if _w.isRunning == 0 {
			worker = _w
			break
		}
	}

	return
}

func (p *Pool) Close() {
	p.close<- struct{}{}
	ticker.Stop()
}

func (p *Pool) startLogStatusTicker() {
	ticker = time.NewTicker(time.Second * time.Duration(p.logStatusTick))
	for {
		select {
		case <- ticker.C:
			p.logPoolStatus()
		}
	}
}

func (p *Pool) logPoolStatus() {
	// single goroutine
	poolLog, err := json.Marshal(p.poolStatus)
	if err != nil {
		log.Error(err)
		return
	}

	log.Info(poolLog)
}

func (p *Pool) waitQuitSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
		case <-quit:
			p.logPoolStatus()
			return
	}
}