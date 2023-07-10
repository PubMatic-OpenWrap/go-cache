package cache

import (
	"sync"
	"time"
)

type AsyncCache struct {
	*Cache
	keys sync.Map
}

// lockAndLoad calls DB only once for same requests
func (c *AsyncCache) lockAndLoad(key string, block bool, callback func()) {
	waitCh := make(chan struct{})
	lockCh, present := c.keys.LoadOrStore(key, waitCh)
	if !present {
		// fetch db data and save in cache
		callback()

		// delete and let other requests take the lock (ideally only 1 per hour per pod)
		c.keys.Delete(key)

		// unblock waiting requests
		close(lockCh.(chan struct{}))
	}

	if block {
		// requests that did not get lock will wait here until the one that reterives the data closes the channel
		<-lockCh.(chan struct{})
	}
}

// GetOrLoad will return value or call function againts key
func (c *AsyncCache) GetOrLoad(key string, dbFunc func()) (interface{}, bool) {
	value, expired, _ := c.GetWithExpiry(key)
	if value != nil {
		if expired {
			//non blocking call for fetching new data
			c.lockAndLoad(key, false, dbFunc)
		}
		return value, true
	}

	//new key: blocking call
	c.lockAndLoad(key, true, dbFunc)
	return c.Get(key)
}

// NewAsyncCache return new object of async map
func NewAsyncCache(defaultExpiration, cleanupInterval, purgeTime time.Duration) *AsyncCache {
	return &AsyncCache{
		Cache: New(defaultExpiration, cleanupInterval, purgeTime),
		keys:  sync.Map{},
	}
}
