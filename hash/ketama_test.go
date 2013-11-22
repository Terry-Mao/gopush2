package hash

import (
	"testing"
)

func TestKetama(t *testing.T) {
	k := NewKetama(5, 256)
	n := k.Node("e10ae660af4f11e0a9c1002564a97bb7")
	if n != "node1" {
		t.Error("e10ae660af4f11e0a9c1002564a97bb7 must hit node1")
	}

	n = k.Node("110ae660af4f11e0a9c1002564a97bb7")
	if n != "node4" {
		t.Error("e10ae660af4f11e0a9c1002564a97bb7 must hit node1")
	}

	n = k.Node("210ae660af4f11e0a9c1002564a97bb7")
	if n != "node3" {
		t.Error("e10ae660af4f11e0a9c1002564a97bb7 must hit node1")
	}

	n = k.Node("310ae660af4f11e0a9c1002564a97bb7")
	if n != "node2" {
		t.Error("e10ae660af4f11e0a9c1002564a97bb7 must hit node1")
	}

	n = k.Node("410ae660af4f11e0a9c1002564a97bb7")
	if n != "node5" {
		t.Error("e10ae660af4f11e0a9c1002564a97bb7 must hit node1")
	}
}
