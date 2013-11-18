package mmhash

import (
	"fmt"
	"testing"
)

func TestMurMurHash2(t *testing.T) {
	i := MurMurHash2("maojian")
	j := MurMurHash2("maojian")
	if i != j {
		t.Errorf("murmurhash2 failed %d != %d", i, j)
	}

	k := MurMurHash2("test")
	if i == k {
		t.Errorf("murmurhash2 failed %d == %d", i, k)
	}

	fmt.Printf("i = %d, j = %d, k = %d\n", i, j, k)
}
