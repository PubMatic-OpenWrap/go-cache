package cache

import (
	"errors"
	"log"
	"sync"
	"time"
)

//Status is signal which helps to identify in which state the current process with a specific key is
type Status int
type tstatus struct {
	expirationTime time.Time
	value          Status
}

//Status for keys/process declared in constants
const (
	STATUS_NOTPRESENT     Status = iota //current key is not present in KeyMap
	STATUS_INPROCESS                    // current key/process is already INPROCESS to fetch data from data source
	STATUS_DONE                         //current key/process have DONE fetching data and updated in cache
	STATUS_INTERNAL_ERROR               //current key/process recieved internal_error while fetching data
	STATUS_INVALID_KEY                  // current key is invalid to be fetched
)

type keyStatus struct {
	keyMap        map[string]tstatus //Status{INPROCESS,DONE,INVALID,NOPRESENT,INVALID_KEY}
	mu            *sync.RWMutex
	purgeInterval time.Duration
}

type funcDef func(key string, err error)

var (
	ErrRoutineBlocked = errors.New("Blocked Time Exceeded")
)

//Async Cache
type AsyncCache struct {
	*Cache
	Fetcher   *Fetcher
	keystatus *keyStatus
	errorFunc funcDef
}

type Config struct {
	Fetcher       *Fetcher
	PurgeInterval time.Duration
	ExpiryTime    time.Duration
	ErrorFunc     funcDef
}

//To Create A New keyStatus
func NewKeyStatus(purge_time time.Duration, errorFunc funcDef) *keyStatus {
	//if purge time is less than or zero, skipping purge for keyStatus
	ks := &keyStatus{
		keyMap:        make(map[string]tstatus),
		mu:            &sync.RWMutex{},
		purgeInterval: purge_time,
	}
	ks.purge(errorFunc)
	return ks
}

//default ErrorHandler Function
func DefaultErrorHandler(key string, err error) {
	log.Println("ERROR: _Key:", key, "ErrorInformation: ", err)
}

//Init NewAsyncCache
// PurgeInterval should be greater than 0, for enable purging
// Expiry Time can be -> -1 - No Expiration OR 0 - Default Expiration OR > 0 (Manual Data Expiry Time)
// Fetcher and ErrorFunc if nil, will have default initializers
func NewAsyncCache(aConfig Config) *AsyncCache {
	if aConfig.Fetcher == nil {
		aConfig.Fetcher = NewFetcher(0)
	}
	if aConfig.ErrorFunc == nil {
		aConfig.ErrorFunc = DefaultErrorHandler
	}
	return &AsyncCache{
		Fetcher:   aConfig.Fetcher,
		keystatus: NewKeyStatus(aConfig.PurgeInterval, aConfig.ErrorFunc),
		Cache:     New(aConfig.ExpiryTime, aConfig.PurgeInterval),
		errorFunc: aConfig.ErrorFunc,
	}
}

//Set Status/Status with respective key in keyStatus
func (ks *keyStatus) Set(key string, Status Status) {

	if len(key) > 0 {
		ks.mu.Lock()
		/* Setting Expiry time as current time --> (now + keyState.Purgetime) and Value for tstatus.value as Updated Status*/
		ks.keyMap[key] = tstatus{expirationTime: time.Now().Add(ks.purgeInterval),
			value: Status}
		ks.mu.Unlock()
	}
}

//Get Status/Status of respective key in keyStatus
func (ks *keyStatus) Get(key string) Status {
	if len(key) == 0 {
		return STATUS_INVALID_KEY
	}
	ks.mu.RLock()
	Status, found := ks.keyMap[key]
	ks.mu.RUnlock()
	if !found {
		return STATUS_NOTPRESENT
	}
	return Status.value
}

// To check if associated key is expired/alive
func (ts tstatus) isExpired() bool {
	if ts.expirationTime.Unix() == 0 {
		return false
	}
	return time.Now().Unix() > ts.expirationTime.Unix()
}

//Deleting expired keys for keyStatus.KeyMap based on keyStatus.expireTime
func (ks *keyStatus) deleteExpiredKeys(errorFunc funcDef) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	for k, t := range ks.keyMap {
		if t.isExpired() {
			delete(ks.keyMap, k)
			// INVALID_CASE: where process is_expired and process status is still in-process
			if t.value == STATUS_INPROCESS && errorFunc != nil {
				errorFunc(k, ErrRoutineBlocked)
			}
		}

	}
}

func (ks *keyStatus) lock() {
	ks.mu.Lock()
}

func (ks *keyStatus) unlock() {
	ks.mu.Unlock()
}

//time-based triggering for purging based on keyStatus.tstatus.purgeTime
func (ks *keyStatus) purge(errorFunc funcDef) {
	//validating purgeTime to be non-negative and Zero
	if ks.purgeInterval <= 0 {
		return
	}
	ticker := time.NewTicker(ks.purgeInterval)
	go func() {
		for range ticker.C {
			ks.deleteExpiredKeys(errorFunc)
		}
	}()
}

func (ac *AsyncCache) AsyncGet(key string) (interface{}, Status) {
	//Fetching from cache
	data, found := ac.Get(key)
	if found {
		return data, STATUS_DONE
	}
	ac.keystatus.lock()
	currStatus := ac.keystatus.get(key)
	if currStatus == STATUS_INPROCESS {
		//data not present in cache and Status INPROCESS
		ac.keystatus.unlock()
		return data, STATUS_INPROCESS
	}

	//New Call for the key, updating KeyStatus for key to Status INPROCESS
	ac.keystatus.set(key, STATUS_INPROCESS)
	ac.keystatus.unlock()
	//asyncCall to DB/dataSource
	go ac.asyncUpdate(key)

	//returning stale data instead of empty with STATUS-DONE
	if data != nil {
		return data, STATUS_DONE
	}
	// return empty data and in-process for ideal first call
	return data, STATUS_INPROCESS
}

func (ac *AsyncCache) asyncUpdate(key string) {
	fetchedData, err, keyprefExpiry := ac.Fetcher.Execute(key)
	// fetching and returning data
	if err != nil {
		// Response Error from DB/Fetcher error
		ac.keystatus.Set(key, STATUS_INTERNAL_ERROR)
		ac.errorFunc(key, err)
		return
	}
	ac.Set(key, fetchedData, keyprefExpiry)
	ac.keystatus.Set(key, STATUS_DONE)
}

func (ks *keyStatus) get(key string) Status {
	if len(key) == 0 {
		return STATUS_INVALID_KEY
	}
	Status, found := ks.keyMap[key]
	if !found {
		return STATUS_NOTPRESENT
	}
	return Status.value
}

func (ks *keyStatus) set(key string, Status Status) {
	if len(key) > 0 {
		/* Setting Expiry time as current time --> (now + keyState.Purgetime) and Value for tstatus.value as Updated Status*/
		ks.keyMap[key] = tstatus{expirationTime: time.Now().Add(ks.purgeInterval),
			value: Status}
	}
}
