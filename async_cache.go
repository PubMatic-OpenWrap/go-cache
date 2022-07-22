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
	STATUS_NOTPRESENT            status = iota //current key is not present in KeyMap
	STATUS_INPROCESS                           // current key/process is already INPROCESS to fetch data from data source
	STATUS_DONE                                //current key/process have DONE fetching data and updated in cache
	STATUS_STATUS_INTERNAL_ERROR               //current key/process recieved internal_error while fetching data
	STATUS_INVALID_KEY                         // current key is invalid to be fetched
)

type keyStatus struct {
	keyMap    map[string]tstatus //status{INPROCESS,DONE,INVALID,NOPRESENT,INVALID_KEY}
	mu        *sync.RWMutex
	purgeTime time.Duration
}

//Initiating DefaulTime and Time-related contants
const (
	DEFAULT_EXPIRATION = 0
)

//To Create A New keyStatus
func NewKeyStatus() *keyStatus {
	return &keyStatus{
		keyMap:    make(map[string]tstatus),
		mu:        &sync.RWMutex{},
		purgeTime: DEFAULT_EXPIRATION,
	}
}

//utility function to Update Keystatus.purgeTime
func (ks *keyStatus) updatePurgetime(td time.Duration) {
	ks.mu.Lock()
	ks.purgeTime = td
	ks.mu.Unlock()
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
func (ks *keyStatus) Purge() {
	ticker := time.NewTicker(ks.purgeTime)
	go func() {
		for range ticker.C {
			ks.deleteExpiredKeys()
		}
	}()
}
