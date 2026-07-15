package proxy

import "sync"

const httpBufferSize = 32 * 1024

type httpBufferPool struct {
	pool sync.Pool
}

func newHTTPBufferPool() *httpBufferPool {
	pool := &httpBufferPool{}
	pool.pool.New = func() any {
		return make([]byte, httpBufferSize)
	}
	return pool
}

func (b *httpBufferPool) Get() []byte {
	return b.pool.Get().([]byte)
}

func (b *httpBufferPool) Put(bytes []byte) {
	b.pool.Put(bytes)
}
