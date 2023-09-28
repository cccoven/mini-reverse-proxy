package main

import (
	"fmt"
	"testing"
)

var hosts = []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}

func TestRandomSelector(t *testing.T) {
	s := RandomSelector{}
	for i := 0; i < len(hosts); i++ {
		fmt.Println(s.Select(hosts))
	}
}

func TestRoundRobinSelector(t *testing.T) {
	s := RoundRobinSelector{}
	for i := 0; i < len(hosts); i++ {
		fmt.Println(s.Select(hosts))
	}
}
