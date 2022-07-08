package cache

import (
	"sync"
	"testing"
)

func Test_keyStatus_Set(t *testing.T) {
	type fields struct {
		keys map[string]status
		mu   *sync.RWMutex
	}
	type args struct {
		key    string
		status status
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       bool
		wantStatus status
	}{
		{
			name:   "With Valid key and state",
			fields: fields(*NewKeyStatus()),
			args: args{
				key:    "prof_123",
				status: STATUS_INPROCESS,
			},
			want:       true,
			wantStatus: STATUS_INPROCESS,
		},
		{
			name:   "With InValid key and state",
			fields: fields(*NewKeyStatus()),
			args: args{
				key:    "",
				status: STATUS_INPROCESS,
			},
			want:       false,
			wantStatus: STATUS_INVALID_KEY,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := &keyStatus{
				keys: tt.fields.keys,
				mu:   tt.fields.mu,
			}
			if got := ks.Set(tt.args.key, tt.args.status); got != tt.want {
				t.Errorf("keyStatus.Set() = %v, want %v", got, tt.want)
				t.Errorf("KeyStatys.Set() sets status %v, want status %v", tt.args.status, tt.wantStatus)
			}
		})
	}
}

func Test_keyStatus_Get(t *testing.T) {
	type fields struct {
		keys map[string]status
		mu   *sync.RWMutex
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   status
	}{
		{
			name: "With Valid key and state as DONE",
			fields: fields{
				keys: map[string]status{"prof_123": STATUS_DONE},
			},
			args: args{
				key: "prof_123",
			},
			want: STATUS_DONE,
		},
		{
			name: "With Valid key and state as INPROCESS",
			fields: fields{
				keys: map[string]status{"prof_123": STATUS_INPROCESS},
			},
			args: args{
				key: "prof_123",
			},
			want: STATUS_INPROCESS,
		},
		{
			name: "With Valid key but not present in keys",
			fields: fields{
				keys: map[string]status{"prof_123": STATUS_INPROCESS},
			},
			args: args{
				key: "getAdUnit_5890",
			},
			want: STATUS_NOTPRESENT,
		},
		{
			name: "With Invalid key and state as INPROCESS",
			fields: fields{
				keys: map[string]status{"prof_123": STATUS_INPROCESS},
			},
			args: args{
				key: "",
			},
			want: STATUS_INVALID_KEY,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := &keyStatus{
				keys: tt.fields.keys,
				mu:   tt.fields.mu,
			}
			if got := ks.Get(tt.args.key); got != tt.want {
				t.Errorf("keyStatus.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}
