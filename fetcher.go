package cache

import (
	"fmt"
	"time"
)

const (
	errInvalidKey = "type:[invalid_key] key:[%s]"
)

type Callback func(key string) (interface{}, error)

type CallbackExpiry struct {
	Callback
	PrefixExpiry time.Duration
}

type Fetcher struct {
	// "cb" stores keyPrefix Length and Static Callback w. r. t. KeyPrefix
	cb        map[string]CallbackExpiry
	prefixLen int
}

//Initiazing Fetcher
func NewFetcher(prefixLen int) *Fetcher {
	return &Fetcher{
		cb:        make(map[string]CallbackExpiry),
		prefixLen: prefixLen,
	}
}

//Registering Fetcher with Each Callback Func w. r. t. keyPrefix
func (f *Fetcher) Register(keyPrefix string, cbf Callback) bool {

	//Case of InValid KeyPrefix
	if len(keyPrefix) != f.prefixLen {
		return false
	}

	expiry := 0
	cbStruct := CallbackExpiry{cbf, time.Duration(expiry)}
	f.cb[keyPrefix] = cbStruct

	return true
}

func (f *Fetcher) Execute(key string) (interface{}, error, time.Duration) {

	var invalidExpiry time.Duration = -1
	//Case of InValid Key
	if len(key) < f.prefixLen {
		return nil, fmt.Errorf(errInvalidKey, key), invalidExpiry
	}
	keyPrefix := key[:f.prefixLen]

	//keyPrefix Not present in cb
	cbStruct, ok := f.cb[keyPrefix]
	if !ok {
		return nil, fmt.Errorf(errInvalidKey, key), invalidExpiry
	}

	data, err := cbStruct.Callback(key)
	return data, err, cbStruct.PrefixExpiry
}
