package main

import (
	"testing"
)

func TestChannelStat(t *testing.T) {
	s := &ChannelStats{}
	s.IncrCreated()
	if s.Created != 1 {
		t.Errorf("channel static error")
	}

	s.IncrExpired()
	if s.Expired != 1 {
		t.Errorf("channel static error")
	}

	s.IncrRefreshed()
	if s.Expired != 1 {
		t.Errorf("channel static error")
	}
}

func BenchmarkChannelStat(b *testing.B) {
	s := &ChannelStats{}
	for i := 0; i < b.N; i++ {
		s.IncrCreated()
	}
}
