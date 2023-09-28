package main

import (
	"math/rand"
	"sync/atomic"
)

type LoadBalancer interface {
	Select(hosts []string) string
}

type RandomSelector struct{}

func (s *RandomSelector) Select(hosts []string) string {
	var randHost string
	count := 0
	for _, host := range hosts {
		count++
		if (rand.Int() % count) == 0 {
			randHost = host
		}
	}
	return randHost
}

type RoundRobinSelector struct {
	robin uint32
}

func (s *RoundRobinSelector) Select(hosts []string) string {
	n := uint32(len(hosts))
	if n == 0 {
		return ""
	}

	host := hosts[s.robin%n]
	atomic.AddUint32(&s.robin, 1)

	return host
}
