package storage

import (
	"sync"
)

// Pool of byte slices of fixed size
type BytePool struct {
	pool *sync.Pool
	size int
}

// Creates a new byte pool with slices of the specified size
func NewBytePool(size int) *BytePool {
	return &BytePool{
		pool: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, size)
				return &b
			},
		},
		size: size,
	}
}

func (p *BytePool) Get() *[]byte {
	return p.pool.Get().(*[]byte)
}

func (p *BytePool) Put(b *[]byte) {
	if cap(*b) < p.size {
		// If the slice is too small, discard it
		return
	}

	// Reset the slice before returning it to the pool
	*b = (*b)[:p.size]
	for i := range *b {
		(*b)[i] = 0
	}

	p.pool.Put(b)
}

type PoolManager struct {
	pools map[int]*BytePool
	mu    sync.RWMutex
}

func NewPoolManager() *PoolManager {
	return &PoolManager{
		pools: make(map[int]*BytePool),
	}
}

func (pm *PoolManager) GetPool(size int) *BytePool {
	pm.mu.RLock()
	pool, ok := pm.pools[size]
	pm.mu.RUnlock()

	if ok {
		return pool
	}

	// Create a new pool if it doesn't exist
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check again in case another goroutine created the pool
	if pool, ok = pm.pools[size]; ok {
		return pool
	}

	pool = NewBytePool(size)
	pm.pools[size] = pool
	return pool
}

// GetBuffer gets a buffer of the specified size from the appropriate pool
func (pm *PoolManager) GetBuffer(size int) *[]byte {
	return pm.GetPool(size).Get()
}

// PutBuffer returns a buffer to the appropriate pool
func (pm *PoolManager) PutBuffer(buffer *[]byte) {
	pm.GetPool(cap(*buffer)).Put(buffer)
}

// Cleanup releases all pools
func (pm *PoolManager) Cleanup() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Clear reference to all pools to allow garbage collection
	pm.pools = make(map[int]*BytePool)
}
