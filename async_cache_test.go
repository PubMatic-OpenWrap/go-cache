package cache

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

const EXPIRATION_TIME = 30 * time.Minute

func Test_keyStatus_Set(t *testing.T) {

	type fields struct {
		keyMap    map[string]tstatus
		mu        *sync.RWMutex
		purgeTime time.Duration
	}
	type args struct {
		key    string
		Status Status
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantStatus Status
	}{
		{
			name:   "With Valid key and state",
			fields: fields(*NewKeyStatus(EXPIRATION_TIME)),
			args: args{
				key:    "prof_123",
				Status: STATUS_INPROCESS,
			},
			wantStatus: STATUS_INPROCESS,
		},
		{
			name:   "With InValid key and state",
			fields: fields(*NewKeyStatus(EXPIRATION_TIME)),
			args: args{
				key:    "",
				Status: STATUS_INPROCESS,
			},
			wantStatus: STATUS_INVALID_KEY,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ks := NewKeyStatus(EXPIRATION_TIME)
			t.Parallel()
			ks.Set(tt.args.key, tt.args.Status)
			if ks.Get(tt.args.key) != tt.wantStatus {
				t.Log(tt.fields.keyMap)
				t.Errorf("KeyStatys.Set() sets Status %v, want Status %v", tt.args.Status, tt.wantStatus)
			}
		})
	}
}

func Test_keyStatus_Get(t *testing.T) {

	type args struct {
		key string
	}
	tests := []struct {
		name string
		KeyS keyStatus
		args args
		want Status
	}{
		{
			name: "With Valid key and state as DONE",
			KeyS: func() keyStatus {
				ks := NewKeyStatus(EXPIRATION_TIME)
				ks.Set("prof_123", STATUS_DONE)
				return *ks
			}(),
			args: args{
				key: "prof_123",
			},
			want: STATUS_DONE,
		},
		{
			name: "With Valid key and state as INPROCESS",
			KeyS: func() keyStatus {
				ks := NewKeyStatus(EXPIRATION_TIME)
				ks.Set("prof_123", STATUS_INPROCESS)
				return *ks
			}(),

			args: args{
				key: "prof_123",
			},
			want: STATUS_INPROCESS,
		},
		{
			name: "With Valid key but not present in keyMap",
			KeyS: *NewKeyStatus(EXPIRATION_TIME),
			args: args{
				key: "getAdUnit_5890",
			},
			want: STATUS_NOTPRESENT,
		},
		{
			name: "With Invalid key and state as INPROCESS",
			KeyS: *NewKeyStatus(EXPIRATION_TIME),
			args: args{
				key: "",
			},
			want: STATUS_INVALID_KEY,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ks := NewKeyStatus(EXPIRATION_TIME)
			ks.keyMap = tt.KeyS.keyMap
			t.Parallel()
			if got := ks.Get(tt.args.key); got != tt.want {
				t.Errorf("keyStatus.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testFillKeyCache() map[string]tstatus {
	ks := NewKeyStatus(25 * time.Millisecond)
	ks.Set("prof_5890", STATUS_INPROCESS)
	ks.Set("aucf_739", STATUS_INVALID_KEY)
	ks.Set("aucf_111", STATUS_INTERNAL_ERROR)
	ks.Set("prof_202", STATUS_NOTPRESENT)

	ks.keyMap["prof_Alive2"] = tstatus{
		value:          STATUS_DONE,
		expirationTime: time.Now().Add(3 * time.Minute),
	}
	return ks.keyMap
}

func Test_keyStatus_Purge(t *testing.T) {
	// type KeyAndstatus = map[string]Status
	tests := []struct {
		name   string
		fields *keyStatus
		want   map[string]Status
	}{
		{
			name: "Purging with valid expired keys",
			fields: &keyStatus{
				keyMap:    testFillKeyCache(),
				mu:        &sync.RWMutex{},
				purgeTime: 25 * time.Millisecond,
			},
			want: map[string]Status{
				"prof_Alive2": STATUS_DONE,
			},
		},
		{
			name: "Purging with Empty KeyMap",
			fields: &keyStatus{
				keyMap:    make(map[string]tstatus),
				mu:        &sync.RWMutex{},
				purgeTime: 250 * time.Millisecond,
			},
			want: map[string]Status{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := tt.fields
			ks.purge()
			if tt.name == "Purging with valid expired keys" {
				time.Sleep(1000 * time.Millisecond)
			}
			mapper := ks.keyMap
			kV := make(map[string]Status)
			for k, v := range mapper {
				kV[k] = v.value
			}
			if !reflect.DeepEqual(kV, tt.want) {
				t.Errorf("%v", reflect.DeepEqual(kV, tt.want))
			}

		})
	}
}

func Benchmark_keyStatus_DeleteKeysWithInbuiltDelete(b *testing.B) {
	b.StopTimer()
	ks := NewKeyStatus(EXPIRATION_TIME)
	ks.keyMap = testFillKeyCache()
	b.StartTimer()
	for n := 0; n < b.N; n++ {
		ks.deleteExpiredKeys()
	}
}

func TestAsyncCache_AsyncGet(t *testing.T) {

	type args struct {
		key string
	}
	tests := []struct {
		name       string
		args       args
		want       interface{}
		wantStatus Status
		as         AsyncCache
	}{
		{name: "Hitting first call request",
			args: args{
				key: "PROF_5890",
			},
			want:       nil,
			wantStatus: STATUS_INPROCESS,
			as:         *InitAsyncCache(),
		},
		{name: "getting data if data is already present in AsyncCache with Status DONE",
			args: args{
				key: "PROF_5890",
			},
			want:       "profile_5890_201",
			wantStatus: STATUS_DONE,
			as: func() AsyncCache {
				as1 := InitAsyncCache()
				as1.Set("PROF_5890", "profile_5890_201", as1.keystatus.purgeTime)
				as1.keystatus.Set("PROF_5890", STATUS_DONE)
				return *as1
			}(),
		},
		{name: "With an Invalid datasource response with internal error ",
			args: args{
				key: "CONF_5890",
			},
			want:       nil,
			wantStatus: STATUS_INPROCESS,
			as: func() AsyncCache {
				as1 := InitAsyncCache()
				return *as1
			}(),
		},
		{name: "Demonstrating asyncUpdate call",
			args: args{
				key: "PROF_5890",
			},
			want:       "profile_5890_201",
			wantStatus: STATUS_DONE,
			as: func() AsyncCache {
				f := NewFetcher(4)
				f.Register("PROF", getProf)
				f.Register("CONF", getConf)
				config := Config{
					Fetcher:             f,
					PurgeTime:           8 * time.Second,
					ExpiryTime:          0,
					ErrorFuncDefination: ErrorHandler,
				}
				ac := NewAsyncCache(&config)
				return *ac
			}(),
		},
		{name: "request with already INPROCESS Status",
			args: args{
				key: "PROF_5890",
			},
			want:       nil,
			wantStatus: STATUS_INPROCESS,
			as: func() AsyncCache {
				f := NewFetcher(4)
				f.Register("PROF", getProf)
				f.Register("CONF", getConf)
				config := Config{
					Fetcher:             f,
					PurgeTime:           8 * time.Second,
					ExpiryTime:          0,
					ErrorFuncDefination: ErrorHandler,
				}
				ac := NewAsyncCache(&config)
				ac.keystatus.Set("PROF_5890", STATUS_INPROCESS)
				return *ac
			}(),
		},
		{name: "Returning Expired Stale data instead of Empty",
			args: args{
				key: "PROF_5890",
			},
			want:       "Expired-Stale-Data",
			wantStatus: STATUS_DONE,
			as: func() AsyncCache {
				f := NewFetcher(4)
				f.Register("PROF", getProf)
				f.Register("CONF", getConf)
				config := Config{
					Fetcher:             f,
					PurgeTime:           10 * time.Millisecond,
					ExpiryTime:          1 * time.Microsecond,
					ErrorFuncDefination: ErrorHandler,
				}
				ac := NewAsyncCache(&config)
				ac.keystatus.Set("PROF_5890", STATUS_DONE)
				ac.Set("PROF_5890", "Expired-Stale-Data", ac.defaultExpiration)
				return *ac
			}(),
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			ac := tt.as
			//we will be calling AsyncGet to explicitly set the data with key
			if tt.name == "Demonstrating asyncUpdate call" {
				ac.AsyncGet(tt.args.key)
			}
			time.Sleep(1 * time.Millisecond)
			got, got1 := ac.AsyncGet(tt.args.key)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AsyncCache.AsyncGet() got data = %v, want data %v", got, tt.want)
			}
			if got1 != tt.wantStatus {
				t.Errorf("AsyncCache.AsyncGet() got Status = %v, want Status %v", got1, tt.wantStatus)
			}
		})
	}
}
func ErrorHandler(key string, err error) {
	fmt.Print("Error for ", key, "is ", err)
}
func InitAsyncCache() *AsyncCache {
	f := NewFetcher(4)
	f.Register("PROF", getProf)
	f.Register("CONF", getConf)
	config := Config{
		Fetcher:             f,
		PurgeTime:           8 * time.Second,
		ExpiryTime:          1000 * time.Millisecond,
		ErrorFuncDefination: nil,
	}
	ac := NewAsyncCache(&config)
	return ac
}

func getProf(key string) (interface{}, error) {
	data := "profile_5890_201"
	return data, nil
}
func getConf(key string) (interface{}, error) {
	return nil, errors.New("err:getting configs")
}

func TestAsyncCache_SetInCache(t *testing.T) {

	type args struct {
		key  string
		data interface{}
	}
	tests := []struct {
		name string
		ac   AsyncCache
		args args
	}{
		{
			name: "Setting data in ac.Cache",
			ac:   *InitAsyncCache(),
			args: args{
				key:  "PROF_541",
				data: "dummy 123",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac1 := tt.ac
			ac1.Set(tt.args.key, tt.args.data, 0)
			data, ok := ac1.Get(tt.args.key)
			if !ok {
				t.Errorf("data found %v", data)
			}

		})
	}
}

func TestAsyncCache_SetWithExpiry(t *testing.T) {

	type args struct {
		key  string
		data interface{}
		t    time.Duration
	}
	tests := []struct {
		name string
		ac   AsyncCache
		args args
	}{
		{
			name: "Setting data with expiry in asyncache.Cache",
			args: args{
				key:  "PROF_2022",
				data: "DummyData",
				t:    500 * time.Millisecond,
			},
			ac: *InitAsyncCache(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac1 := tt.ac
			ac1.Set(tt.args.key, tt.args.data, tt.args.t)
			data, ok := ac1.Get(tt.args.key)
			if !ok {
				t.Errorf("data found %v", data)
			}

		})
	}
}

func TestNewAsyncCache(t *testing.T) {
	type args struct {
		aConfig *Config
	}
	tests := []struct {
		name string
		args args
		want *AsyncCache
	}{
		{
			name: "Check for Validation",
			args: args{
				aConfig: &Config{
					Fetcher:             nil,
					PurgeTime:           0,
					ExpiryTime:          0,
					ErrorFuncDefination: nil,
				},
			},
			want: &AsyncCache{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAsyncCache(tt.args.aConfig); got == nil || reflect.TypeOf(got) != reflect.TypeOf(tt.want) {
				t.Errorf("NewAsyncCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
