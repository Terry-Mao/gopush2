package hash

import (
	//	"crypto/md5"
	"fmt"
	"sort"
)

// Convenience types for common cases
// UIntSlice attaches the methods of Interface to []uint, sorting in increasing order.
type UIntSlice []uint

func (p UIntSlice) Len() int           { return len(p) }
func (p UIntSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p UIntSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type Ketama struct {
	node         int             // phsical node number
	vnode        int             // virual node number
	nodes        []uint          // nodes
	nodesMapping map[uint]string // nodes maping
}

// New create a ketama consistent hashing struct
func NewKetama(node, vnode int) *Ketama {
	ketama := &Ketama{}
	ketama.node = node
	ketama.vnode = vnode
	ketama.nodes = []uint{}
	ketama.nodesMapping = map[uint]string{}

	ketama.initCircle()

	return ketama
}

// init consistent hashing circle
func (k *Ketama) initCircle() {
	for idx := 1; idx < k.node+1; idx++ {
		node := fmt.Sprintf("node%d", idx)
		for i := 0; i < k.vnode; i++ {
			//nodeMD5 := md5.New()
			//nodeMD5.Write([]byte(fmt.Sprintf("%s#%d", node, i)))
			//digest := nodeMD5.Sum(nil)
			//for j := 0; j < 4; j++ {
			//	virtualNodePos := hashVal(digest, j)
			//	k.nodes = append(k.nodes, virtualNodePos)
			//	k.nodesMapping[virtualNodePos] = node
			//}

			vnode := fmt.Sprintf("%s#%d", node, i)
			vpos := MurMurHash2(vnode)
			k.nodes = append(k.nodes, vpos)
			k.nodesMapping[vpos] = node
		}
	}

	sort.Sort(UIntSlice(k.nodes))
}

// Node get a consistent hashing node by key
func (k *Ketama) Node(key string) string {
	//keyPos := hashVal(digest, 0)
	//keyPos := MurMurHash2(key)
	idx := searchLeft(k.nodes, MurMurHash2(key))

	pos := k.nodes[0]
	if idx != len(k.nodes) {
		pos = k.nodes[idx]
	}

	return k.nodesMapping[pos]
}

// search the slice left-most value
func searchLeft(a []uint, x uint) int {
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
//func hashVal(digest []byte, times int) int {
//	return int((int(digest[3+times*4]) << 24) | (int(digest[2+times*4]) << 16) | (int(digest[1+times*4]) << 8) | (int(digest[0+times*4])))
//}
