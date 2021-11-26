package bufpool

import "sync"

var (
	pools = []struct {
		size int
		pool sync.Pool
	}{
		{
			size: 128,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 128)
				},
			},
		},
		{
			size: 512,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 512)
				},
			},
		},
		{
			size: 1024,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 1024)
				},
			},
		},
		{
			size: 4096,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 4096)
				},
			},
		},
		{
			size: 8192,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 8192)
				},
			},
		},
		{
			size: 16 * 1024,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 16*1024)
				},
			},
		},
		{
			size: 32 * 1024,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 32*1024)
				},
			},
		},
		{
			size: 64 * 1024,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 64*1024)
				},
			},
		},
		{
			size: 65 * 1024,
			pool: sync.Pool{
				New: func() interface{} {
					return make([]byte, 65*1024)
				},
			},
		},
	}
)

// Get returns a buffer size.
func Get(size int) []byte {
	for i := range pools {
		if size <= pools[i].size {
			return pools[i].pool.Get().([]byte)[:size]
		}
	}
	return make([]byte, size)
}

func Put(b []byte) {
	for i := range pools {
		if cap(b) == pools[i].size {
			pools[i].pool.Put(b[:cap(b)])
		}
	}
}
