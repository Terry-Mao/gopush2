package hash

import (
	"testing"
)

func TestKetama(t *testing.T) {
	k := NewKetama(5, 200)
	n := k.Node("e10ae660af4f11e0a9c1002564a97bb7")
	if n != "node4" {
		t.Error("e10ae660af4f11e0a9c1002564a97bb7 must hit node4")
	}
}
