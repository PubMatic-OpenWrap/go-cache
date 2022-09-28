package cache

import (
	"fmt"
)

const (
	errInvalidKey = "type:[invalid_key] key:[%s]"
)

type Callback func(key string) (interface{}, error)

type fetcher struct {
	// "cb" stores keyPrefix Length and Static Callback w. r. t. KeyPrefix
	cb        map[string]Callback
	prefixLen int
}

//Initiazing Fetcher
func NewFetcher(prefixLen int) *fetcher {
	return &fetcher{
		cb:        make(map[string]Callback),
		prefixLen: prefixLen,
	}
}

//Registering Fetcher with Each Callback Func w. r. t. keyPrefix
func (f *fetcher) Register(keyPrefix string, cbf Callback) bool {

	//Case of InValid KeyPrefix
	if len(keyPrefix) != f.prefixLen {
		return false
	}

	f.cb[keyPrefix] = cbf
	return true
}

func (f *fetcher) Execute(key string) (interface{}, error) {

	//Case of InValid Key
	if len(key) < f.prefixLen {
		return nil, fmt.Errorf(errInvalidKey, key)
	}
	keyPrefix := key[:f.prefixLen]

	//keyPrefix Not present in cb
	cbf, ok := f.cb[keyPrefix]
	if !ok {
		return nil, fmt.Errorf(errInvalidKey, key)
	}

	return cbf(key)
}
