package cache

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/harshvardha/TerTerChat/internal/database"
)

type cacheShard struct {
	items map[string][]database.Message
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

type DynamicShardedCache struct {
	shards      []*cacheShard
	shardCount  int32
	minShards   int
	maxShards   int
	resizeMutex sync.RWMutex
	stopChan    chan struct{}
	metrics     *cacheMetrics
}

// creating new sharded cache
func NewDynamicShardedCache(minShards, maxShards int) *DynamicShardedCache {
	if minShards < 1 {
		minShards = 1
	}

	if maxShards < minShards {
		maxShards = 4 * minShards
	}

	cache := &DynamicShardedCache{
		shards:     make([]*cacheShard, minShards),
		shardCount: int32(minShards),
		minShards:  minShards,
		maxShards:  maxShards,
		stopChan:   make(chan struct{}),
		metrics:    &cacheMetrics{},
	}

	for i := range minShards {
		cache.shards[i] = &cacheShard{
			items: make(map[string][]database.Message),
		}
	}

	go cache.monitorAndAdjust()
	return cache
}

func (dsc *DynamicShardedCache) monitorAndAdjust() {
	// ticker will be used to monitor the cache to decide whether scaling is required or not
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dsc.checkAndResize()
		case <-dsc.stopChan: // signal to stop cache monitoring
			return
		}
	}
}

func (dsc *DynamicShardedCache) checkAndResize() {
	currentShardCount := atomic.LoadInt32(&dsc.shardCount)

	// calculating load factor(based on hits and misses)
	// if loadFactor is more than 0.2(20%) and currentShardCount < maxShards then scale up
	// if loadFactor is less than 0.2 then and currentShardCount > minShards scale down
	if dsc.metrics.misses > 0 {
		loadFactor := float64((dsc.metrics.misses) / (dsc.metrics.hits + dsc.metrics.misses))
		dsc.metrics.averageLoad = loadFactor
		dsc.metrics.lastCheckTime = time.Now()

		if loadFactor >= 0.2 && currentShardCount < int32(dsc.maxShards) {
			newShardCount := min(int(currentShardCount)*2, dsc.maxShards)
			dsc.resize(newShardCount)
		} else if loadFactor < 0.2 && currentShardCount > int32(dsc.minShards) {
			newShardCount := max(int(currentShardCount)/2, dsc.minShards)
			dsc.resize(newShardCount)
		}
	}
}

// method to stop cache monitoring
func (dsc *DynamicShardedCache) StopCacheMonitoring() {
	close(dsc.stopChan)
}

func (dsc *DynamicShardedCache) resize(newShardCount int) {
	dsc.resizeMutex.Lock()
	defer dsc.resizeMutex.Unlock()

	currentShards := int(dsc.shardCount)
	if currentShards == newShardCount {
		return
	}

	// creating a new shards array
	newShards := make([]*cacheShard, newShardCount)
	for i := range newShardCount {
		newShards[i] = &cacheShard{
			items: make(map[string][]database.Message),
		}
	}

	// rehashing and redistributing the existing items
	for i := range currentShards {
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

func (dsc *DynamicShardedCache) hashKey(key string, shardCount int) int {
	// 32-bit fnv 1-a hashing algorithm is used beacause it is light on cpu
	// because it performs only two simple operations: multiplication and XOR
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return int(hasher.Sum32()) % shardCount
}

func (dsc *DynamicShardedCache) getShard(key string) *cacheShard {
	shardIndex := dsc.hashKey(key, int(dsc.shardCount))
	return dsc.shards[shardIndex]
}

func (dsc *DynamicShardedCache) Get(key string, createdAt time.Time) []database.Message {
	// attaining read lock on cache
	dsc.resizeMutex.RLock()
	defer dsc.resizeMutex.RUnlock()

	// checking if user is requesting group messages or one-to-one conversation
	// for one-to-one conversation len(keys) = 2, fetching the conversation
	// taking user ids of both involved users and creating a hash for the conversation
	// using fnv 1-a then dividing this hash with current shard count and using this value
	// to fetch the target shard and using the raw hash value to access the conversation
	// for group conversations only group id will be used to store the messages

	shard := dsc.getShard(key)
	shard.mutex.RLock()
	defer shard.mutex.RUnlock()

	// if the createdAt time of latest message is <= createdAt argument then return messages
	items, ok := shard.items[key]
	if !ok {
		atomic.AddUint64(&dsc.metrics.misses, 1)
		return nil
	}

	if items[len(items)-1].CreatedAt.Before(createdAt) || items[len(items)-1].CreatedAt.Equal(createdAt) {
		atomic.AddUint64(&dsc.metrics.hits, 1)
		return items
	}

	return nil
}

func (dsc *DynamicShardedCache) Remove(key string) {
	dsc.resizeMutex.RLock()
	defer dsc.resizeMutex.RUnlock()

	shard := dsc.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()

	_, ok := shard.items[key]
	if ok {
		delete(shard.items, key)
		dsc.metrics.evictions++
	}
}

func (dsc *DynamicShardedCache) RemoveMessage(key string, messageID uuid.UUID) {
	dsc.resizeMutex.RLock()
	defer dsc.resizeMutex.RUnlock()

	shard := dsc.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()

	messages := shard.items[key]
	for index, message := range messages {
		if message.ID == messageID {
			newMessagesSlice := make([]database.Message, 10)
			// if index == 0 then just copy all the elements other than the first one
			// if index == len(messages) - 1 then copy all the elements other than last one
			// if index > 0 && index < len(messages) - 1 then copy elements before index and after index
			// after the required operation update the shard map key with new message slice
			if index == 0 {
				newMessagesSlice = append(newMessagesSlice, messages[1:]...)
			} else if index == len(messages)-1 {
				newMessagesSlice = append(newMessagesSlice, messages[0:index+1]...)
			} else {
				newMessagesSlice = append(newMessagesSlice, messages[0:index+1]...)
				newMessagesSlice = append(newMessagesSlice, messages[index+1:]...)
			}
			shard.items[key] = newMessagesSlice
			break
		}
	}
}

func (dsc *DynamicShardedCache) Update(key string, messageID uuid.UUID, description string, received bool, updatedAt time.Time) {
	dsc.resizeMutex.RLock()
	defer dsc.resizeMutex.RUnlock()

	shard := dsc.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()

	// updating the existing message
	messages := shard.items[key]
	for _, message := range messages {
		if message.ID == messageID {
			if len(description) > 0 && message.Description != description {
				message.Description = description
			}
			if !message.Recieved && received {
				message.Recieved = true
			}
			if updatedAt.After(message.UpdatedAt) {
				message.UpdatedAt = updatedAt
			}
			break
		}
	}
}

func (dsc *DynamicShardedCache) Set(key string, value database.Message) {
	dsc.resizeMutex.RLock()
	defer dsc.resizeMutex.RUnlock()

	shard := dsc.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()

	// checking if the key already exist in cache
	// if key already exist then pop out the oldest message and insert the new one
	// if key does not exist then create a new []database.Message and insert into cache with new key
	messages, ok := shard.items[key]
	if ok {
		if len(messages) == 10 {
			values := make([]database.Message, 10)
			values = append(values, messages[1:]...)
			values = append(values, value)
			shard.items[key] = values
		} else {
			messages = append(messages, value)
			shard.items[key] = messages
		}
	} else {
		values := make([]database.Message, 10)
		values[0] = value
		shard.items[key] = values
	}
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
