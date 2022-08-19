package cache

import (
	"sync"
	"time"
)

//status is signal which helps to identify in which state the current process with a specific key is
type status int
type tstatus struct {
	expirationTime time.Time
	value          status
}

//status for keys/process declared in constants
const (
	STATUS_NOTPRESENT     status = iota //current key is not present in KeyMap
	STATUS_INPROCESS                    // current key/process is already INPROCESS to fetch data from data source
	STATUS_DONE                         //current key/process have DONE fetching data and updated in cache
	STATUS_INTERNAL_ERROR               //current key/process recieved internal_error while fetching data
	STATUS_INVALID_KEY                  // current key is invalid to be fetched
)

type keyStatus struct {
	keyMap    map[string]tstatus //status{INPROCESS,DONE,INVALID,NOPRESENT,INVALID_KEY}
	mu        *sync.RWMutex
	purgeTime time.Duration
}

//Async Cache
type AsyncCache struct {
	gCache    *Cache
	fetcher   *fetcher
	keystatus *keyStatus
}

//Initiating DefaulTime and Time-related contants
const (
	EXPIRATION_TIME = 30 * time.Minute
)

//To Create A New keyStatus
func NewKeyStatus(purge_time time.Duration) *keyStatus {
	return &keyStatus{
		keyMap:    make(map[string]tstatus),
		mu:        &sync.RWMutex{},
		purgeTime: purge_time,
	}
}

//Set status/status with respective key in keyStatus
func (ks *keyStatus) Set(key string, status status) {

	if len(key) > 0 {
		ks.mu.Lock()
		/* Setting Expiry time as current time --> (now + keyState.Purgetime) and Value for tstatus.value as Updated status*/
		ks.keyMap[key] = tstatus{expirationTime: time.Now().Add(ks.purgeTime),
			value: status}
		ks.mu.Unlock()
	}
}

//Get status/status of respective key in keyStatus
func (ks *keyStatus) Get(key string) status {
	if len(key) == 0 {
		return STATUS_INVALID_KEY
	}
	ks.mu.RLock()
	status, found := ks.keyMap[key]
	ks.mu.RUnlock()
	if !found {
		return STATUS_NOTPRESENT
	}
	return status.value
}

// To check if associated key is expired/alive
func (ts tstatus) isExpired() bool {
	if ts.expirationTime.Unix() == 0 {
		return false
	}
	return time.Now().Unix() > ts.expirationTime.Unix()
}

//Deleting expired keys for keyStatus.KeyMap based on keyStatus.expireTime
func (ks *keyStatus) deleteExpiredKeys() {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	for k, t := range ks.keyMap {
		if t.isExpired() {
			delete(ks.keyMap, k)
		}
	}
}

//time-based triggering for purging based on keyStatus.tstatus.purgeTime
func (ks *keyStatus) purge() {
	ticker := time.NewTicker(ks.purgeTime)
	go func() {
		for range ticker.C {
			ks.deleteExpiredKeys()
		}
	}()
}

//Init NewAsyncCache
// set cache_expiration to 0 , to use default EXPIRATION_TIME --> 30 Minutes
func NewAsyncCache(prefix_len int, purge_time time.Duration, cache_expiration time.Duration) *AsyncCache {
	if cache_expiration == 0 {
		cache_expiration = EXPIRATION_TIME
	}
	return &AsyncCache{
		fetcher:   NewFetcher(prefix_len),
		keystatus: NewKeyStatus(purge_time),
		gCache:    New(cache_expiration, purge_time),
	}
}

func (ac *AsyncCache) AsyncGet(key string) (interface{}, status, error) {
	//Fetching from cache
	cache_status := STATUS_INPROCESS
	data, ok := ac.gCache.Get(key)
	if ok {
		return data, STATUS_DONE, nil
	} else if !ok && ac.keystatus.Get(key) == STATUS_INPROCESS {
		//data not present in cache and status INPROCESS
		return data, cache_status, nil
	} else if !ok && ac.keystatus.Get(key) != STATUS_INPROCESS {
		//New Call for the key, updating KeyStatus for key to Status INPROCESS
		ac.keystatus.Set(key, cache_status)
		//asyncCall to DB/dataSource
		go asyncUpdate(ac, key)
	}
	//returning data and cache/keystatus
	return data, cache_status, nil
}

func asyncUpdate(ac *AsyncCache, key string) {
	done := make(chan bool, 1)
	go func(done_status chan bool) {
		dataStatus := STATUS_INPROCESS
		fetchedData, err := ac.fetcher.Execute(key)
		// fetching and returning data
		if err != nil {
			// Response Error from DB/Fetcher error
			dataStatus = STATUS_INTERNAL_ERROR
		}
		ac.gCache.Set(key, fetchedData, ac.keystatus.purgeTime)
		dataStatus = STATUS_DONE
		ac.keystatus.Set(key, dataStatus)

		//updating async-update process status
		done_status <- true

	}(done)
	<-done
	close(done)
}
