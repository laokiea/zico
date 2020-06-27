package zico

import (
	"flag"
	"fmt"
)

type Config interface {
	parse()
	set(pool *Pool)
	get() configs
}

type configs map[string]interface{}

type PoolConfig struct {
	Config   configs
}

func NewPoolConfig() *PoolConfig {
	return &PoolConfig{
		Config: make(configs),
	}
}

func (pc *PoolConfig) parse() {
	pc.Config["with-sync-pool"] = flag.Bool("with-sync-pool", true, "with-sync-pool")
	pc.Config["with-log-pool-status"] = flag.Bool("with-log-pool-status", true, "with-log-pool-status")
	flag.Parse()
}

func (pc *PoolConfig) set(pool *Pool) {
	if f, e := pc.Config["with-sync-pool"]; e {
		pool.WithSyncPool(*f.(*bool))
	}

	if f, e := pc.Config["with-log-pool-status"]; e {
		pool.WithLogPoolStatus(*f.(*bool))
	}
}

func (pc *PoolConfig) get() configs {
	return pc.Config
}

func (pc *PoolConfig) format() {
	for k, v := range pc.Config {
		fmt.Println(k, *v.(*bool))
	}
}
