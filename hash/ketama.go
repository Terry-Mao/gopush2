package hash

import (
	"crypto/md5"
	"fmt"
	"sort"
)

type Ketama struct {
	nodeNum        int            // phsical node number
	virtualNodeNum int            // virual node number
	nodes          []int          // nodes
	nodesMapping   map[int]string // nodes maping
}

// New create a ketama consistent hashing struct
func NewKetama(node, virtualNode int) *Ketama {
	ketama := &Ketama{}
	ketama.nodeNum = node
	ketama.virtualNodeNum = virtualNode
	ketama.nodes = []int{}
	ketama.nodesMapping = map[int]string{}

	ketama.initCircle()

	return ketama
}

// init consistent hashing circle
func (k *Ketama) initCircle() {
	for idx := 1; idx < k.nodeNum+1; idx++ {
		node := fmt.Sprintf("node%d", idx)
		for i := 0; i < k.virtualNodeNum/4; i++ {
			nodeMD5 := md5.New()
			nodeMD5.Write([]byte(fmt.Sprintf("%s%d", node, i)))
			digest := nodeMD5.Sum(nil)
			for j := 0; j < 4; j++ {
				virtualNodePos := hashVal(digest, j)
				k.nodes = append(k.nodes, virtualNodePos)
				k.nodesMapping[virtualNodePos] = node
			}
		}
	}

	sort.Sort(sort.IntSlice(k.nodes))
}

// Node get a consistent hashing node by key
func (k *Ketama) Node(key string) string {
	keyMD5 := md5.New()
	keyMD5.Write([]byte(key))
	digest := keyMD5.Sum(nil)
	keyPos := hashVal(digest, 0)
	idx := searchLeft(k.nodes, keyPos)

	pos := k.nodes[0]
	if idx != len(k.nodes) {
		pos = k.nodes[idx]
	}

	return k.nodesMapping[pos]
}

// search the slice left-most value
func searchLeft(a []int, x int) int {
	lo := 0
	hi := len(a)
	for lo < hi {
		mid := (lo + hi) / 2
		if a[mid] < x {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	return lo
}

// MD5-based hash algorithm used by ketama
func hashVal(digest []byte, times int) int {
	return int((int(digest[3+times*4]) << 24) | (int(digest[2+times*4]) << 16) | (int(digest[1+times*4]) << 8) | (int(digest[0+times*4])))
}
