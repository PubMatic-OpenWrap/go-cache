package cache

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func Test_keyStatus_Set(t *testing.T) {

	type fields struct {
		keyMap    map[string]tstatus
		mu        *sync.RWMutex
		purgeTime time.Duration
	}
	type args struct {
		key    string
		status status
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantStatus status
	}{
		{
			name:   "With Valid key and state",
			fields: fields(*NewKeyStatus(EXPIRATION_TIME)),
			args: args{
				key:    "prof_123",
				status: STATUS_INPROCESS,
			},
			wantStatus: STATUS_INPROCESS,
		},
		{
			name:   "With InValid key and state",
			fields: fields(*NewKeyStatus(EXPIRATION_TIME)),
			args: args{
				key:    "",
				status: STATUS_INPROCESS,
			},
			wantStatus: STATUS_INVALID_KEY,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ks := NewKeyStatus(EXPIRATION_TIME)
			t.Parallel()
			ks.Set(tt.args.key, tt.args.status)
			if ks.Get(tt.args.key) != tt.wantStatus {
				t.Log(tt.fields.keyMap)
				t.Errorf("KeyStatys.Set() sets status %v, want status %v", tt.args.status, tt.wantStatus)
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
		want status
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
	ks := NewKeyStatus(2 * time.Second)
	ks.Set("prof_5890", STATUS_INPROCESS)
	ks.Set("aucf_739", STATUS_INVALID_KEY)
	ks.Set("aucf_111", STATUS_INTERNAL_ERROR)
	ks.Set("prof_202", STATUS_NOTPRESENT)
	ks.keyMap["prof_Alive1"] = tstatus{
		value:          STATUS_INPROCESS,
		expirationTime: time.Now().Add(5 * time.Second),
	}
	ks.keyMap["prof_Alive2"] = tstatus{
		value:          STATUS_DONE,
		expirationTime: time.Now().Add(3 * time.Minute),
	}
	return ks.keyMap
}

func Test_keyStatus_Purge(t *testing.T) {
	// type KeyAndstatus = map[string]status
	tests := []struct {
		name   string
		fields *keyStatus
		want   map[string]status
	}{
		{
			name: "Purging with valid expired keys",
			fields: &keyStatus{
				keyMap:    testFillKeyCache(),
				mu:        &sync.RWMutex{},
				purgeTime: 2 * time.Second,
			},
			want: map[string]status{
				"prof_Alive2": STATUS_DONE,
			},
		},
		{
			name: "Purging with Empty KeyMap",
			fields: &keyStatus{
				keyMap:    make(map[string]tstatus),
				mu:        &sync.RWMutex{},
				purgeTime: 2 * time.Second,
			},
			want: map[string]status{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := tt.fields
			ks.purge()
			time.Sleep(10 * time.Second)
			mapper := ks.keyMap
			kV := make(map[string]status)
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
