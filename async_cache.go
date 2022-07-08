package cache

import (
	"sync"
)

//status for keys/process declared in constants
const (
	STATUS_INPROCESS status = iota
	STATUS_DONE
	STATUS_INVALID
	STATUS_NOTPRESENT
	STATUS_INVALID_KEY
)

type keyStatus struct {
	keys map[string]status //status{INPROCESS,DONE,INVALID,NOPRESENT,INVALID_KEY}
	mu   *sync.RWMutex
}

type status int

//To Create A New keyStatus
func NewKeyStatus() *keyStatus {
	return &keyStatus{
		keys: make(map[string]status),
		mu:   &sync.RWMutex{},
	}
}

//Set status/status with respective key in keyStatus
func (ks *keyStatus) Set(key string, status status) bool {

	if len(key) > 0 {
		ks.mu.Lock()
		ks.keys[key] = status
		ks.mu.Unlock()
	} else {
		return false
	}
	return true
}

//Get status/status of respective key in keyStatus
func (ks *keyStatus) Get(key string) status {
	if len(key) == 0 {
		return STATUS_INVALID_KEY
	} else {
		status, found := ks.keys[key]
		if !found {
			return STATUS_NOTPRESENT
		}
		return status
	}
}
