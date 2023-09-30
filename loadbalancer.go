package main

import (
	"math/rand"
	"sync/atomic"
)

type LoadBalancer interface {
	Select(hosts []*Upstream) *Upstream
}

type RandomSelector struct{}

func NewRandomSelector() LoadBalancer {
	return &RandomSelector{}
}

func (s *RandomSelector) Select(upstreams []*Upstream) *Upstream {
	var randHost *Upstream
	count := 0
	for _, ups := range upstreams {
		count++
		if (rand.Int() % count) == 0 {
			randHost = ups
		}
	}
	return randHost
}

type RoundRobinSelector struct {
	robin uint32
}

func NewRoundRobinSelector() LoadBalancer {
	return &RoundRobinSelector{}
}

func (s *RoundRobinSelector) Select(hosts []*Upstream) *Upstream {
	n := uint32(len(hosts))
	if n == 0 {
		return nil
	}

	host := hosts[s.robin%n]
	atomic.AddUint32(&s.robin, 1)

	return host
}
