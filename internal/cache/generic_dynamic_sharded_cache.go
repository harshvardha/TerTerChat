package cache

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

type cacheShard[T any] struct {
	items map[string]T
	mutex sync.RWMutex
}

type cacheMetrics struct {
	hits          uint64
	misses        uint64
	evictions     uint64
	resizeEvents  uint64
	averageLoad   float64
	lastCheckTime time.Time
}

type dynamicShardedCache[T any] struct {
	shards      []*cacheShard[T]
	shardCount  int32
	minShards   int
	maxShards   int
	resizeMutex sync.RWMutex
	stopChan    chan struct{}
	metrics     *cacheMetrics
}

// creating new sharded cache
func NewDynamicShardedCache[T any](minShards, maxShards int) *dynamicShardedCache[T] {
	if minShards < 1 {
		minShards = 1
	}

	if maxShards < minShards {
		maxShards = 4 * minShards
	}

	cache := &dynamicShardedCache[T]{
		shards:     make([]*cacheShard[T], minShards),
		shardCount: int32(minShards),
		minShards:  minShards,
		maxShards:  maxShards,
		stopChan:   make(chan struct{}),
		metrics:    &cacheMetrics{},
	}

	for i := 0; i < minShards; i++ {
		cache.shards[i] = &cacheShard[T]{
			items: make(map[string]T),
		}
	}

	go cache.monitorAndAdjust()
	return cache
}

func (dsc *dynamicShardedCache[T]) monitorAndAdjust() {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		dsc.checkAndResize()
	case <-dsc.stopChan:
		return
	}
}

func (dsc *dynamicShardedCache[T]) checkAndResize() {
	currentShardCount := atomic.LoadInt32(&dsc.shardCount)

	// calculating load factor(based on hits and misses)
	// if loadFactor is more than 0.2(20%) and currentShardCount < maxShards then scale up
	// if loadFactor is less than 0.2 then and currentShardCount > minShards scale down
	loadFactor := float64((dsc.metrics.misses) / (dsc.metrics.hits + dsc.metrics.misses))

	if loadFactor >= 0.2 && currentShardCount < int32(dsc.maxShards) {
		newShardCount := min(int(currentShardCount)*2, dsc.maxShards)
		dsc.resize(newShardCount)
	} else if loadFactor < 0.2 && currentShardCount > int32(dsc.minShards) {
		newShardCount := max(int(currentShardCount)/2, dsc.minShards)
		dsc.resize(newShardCount)
	}
}

func (dsc *dynamicShardedCache[T]) resize(newShardCount int) {
	dsc.resizeMutex.Lock()
	defer dsc.resizeMutex.Unlock()

	currentShards := int(dsc.shardCount)
	if currentShards == newShardCount {
		return
	}

	// creating a new shards array
	newShards := make([]*cacheShard[T], newShardCount)
	for i := 0; i < newShardCount; i++ {
		newShards[i] = &cacheShard[T]{
			items: make(map[string]T),
		}
	}

	// rehashing and redistributing the existing items
	for i := 0; i < currentShards; i++ {
		dsc.shards[i].mutex.Lock()
		for key, value := range dsc.shards[i].items {
			shardIndex := dsc.hashKey(key, newShardCount)
			newShards[shardIndex].mutex.Lock()
			newShards[shardIndex].items[key] = value
			newShards[shardIndex].mutex.Unlock()
		}
	}

	// updating shard count atomically
	atomic.StoreInt32(&dsc.shardCount, int32(newShardCount))
	dsc.shards = newShards
	dsc.metrics.resizeEvents++
}

func (dsc *dynamicShardedCache[T]) hashKey(key string, shardCount int) int {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return int(hasher.Sum32()) % shardCount
}

func (dsc *dynamicShardedCache[T]) getShard(key string) *cacheShard[T] {
	shardIndex := dsc.hashKey(key, int(dsc.shardCount))
	return dsc.shards[shardIndex]
}

func (dsc *dynamicShardedCache[T]) Get(key string) (any, bool) {
	dsc.resizeMutex.RLock()
	defer dsc.resizeMutex.RUnlock()

	shard := dsc.getShard(key)
	shard.mutex.RLock()
	defer shard.mutex.RLock()

	item, ok := shard.items[key]
	if !ok {
		atomic.AddUint64(&dsc.metrics.misses, 1)
		return nil, false
	}

	atomic.AddUint64(&dsc.metrics.hits, 1)
	return item, true
}

func (dsc *dynamicShardedCache[T]) Remove(key string) bool {
	dsc.resizeMutex.RLock()
	defer dsc.resizeMutex.RUnlock()

	return true
}

func (dsc *dynamicShardedCache[T]) Set(key string, value T) {
	dsc.resizeMutex.RLock()
	defer dsc.resizeMutex.RUnlock()

	shard := dsc.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()
}

func min(a int, b int) int {
	if a < b {
		return a
	}

	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}

	return b
}
