package mmhash

func MurMurHash2(data string) uint {
	l := len(data)
	c := 0
	h := uint(0 ^ l)
	k := uint(0)

	for l >= 4 {
		k = uint(data[c])
		k |= uint(data[c+1]) << 8
		k |= uint(data[c+2]) << 16
		k |= uint(data[c+3]) << 24

		k *= 0x5bd1e995
		k ^= k >> 24
		k *= 0x5bd1e995

		h *= 0x5bd1e995
		h ^= k

		c += 4
		l -= 4
	}

	switch l {
	case 3:
		h ^= uint(data[c+2]) << 16
	case 2:
		h ^= uint(data[c+1]) << 8
	case 1:
		h ^= uint(data[c])
		h *= 0x5bd1e995
	}

	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15

	return h
}
