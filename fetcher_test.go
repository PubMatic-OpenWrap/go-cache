package cache

import (
	"reflect"
	"testing"
	"time"
)

func TestFetcherRegister(t *testing.T) {
	type fields struct {
		cb        map[string]CallbackExpiry
		prefixLen int
	}
	type args struct {
		keyPrefix string
		cbf       Callback
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "Invalid KeyPrefix",
			fields: fields(*NewFetcher(5)),
			args: args{
				keyPrefix: "AAG0",
				cbf:       nil,
			},
			want: false,
		},
		{
			name: "Empty KeyPrefix Execution",
			fields: fields{
				cb: map[string]CallbackExpiry{
					"AAG00": {nil, 0},
					"AAA00": {nil, 0},
					"AAB00": {nil, 0},
				},
				prefixLen: 5,
			},
			args: args{
				keyPrefix: "",
				cbf:       nil,
			},
			want: false,
		},
		{
			name: "Valid KeyPrefix Callback Registeration",
			fields: fields{
				cb: map[string]CallbackExpiry{
					"AAG00": {nil, 0},
					"AAA00": {nil, 0},
					"AAB00": {nil, 0},
				},
				prefixLen: 5,
			},
			args: args{
				keyPrefix: "AAC00",
				cbf:       CbGetAdUnitConfig,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Fetcher{
				cb:        tt.fields.cb,
				prefixLen: tt.fields.prefixLen,
			}
			if got := f.Register(tt.args.keyPrefix, tt.args.cbf); got != tt.want {
				t.Errorf("fetcher.Register() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetcherExecute(t *testing.T) {
	type fields struct {
		cb        map[string]CallbackExpiry
		prefixLen int
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		want1   time.Duration
		wantErr bool
	}{
		{
			name: "Invalid Key",
			fields: fields{
				cb: map[string]CallbackExpiry{
					"AAG00": {nil, 0},
				},
				prefixLen: 5,
			},
			args: args{
				key: "InvK",
			},
			want:    nil,
			want1:   -1,
			wantErr: true,
		},
		{
			name: "Unexisting Key Execution",
			fields: fields{
				cb: map[string]CallbackExpiry{
					"AAG00": {nil, 0},
					"AAA00": {nil, 0},
					"AAB00": {nil, 0},
				},
				prefixLen: 5,
			},
			args: args{
				key: "UnExisted_Key",
			},
			want:    nil,
			want1:   -1,
			wantErr: true,
		},
		{
			name: "Valid Key Execution",
			fields: fields{
				cb: map[string]CallbackExpiry{
					"AAG00": {CbGetAdUnitConfig, 50},
					"AAA00": {nil, 0},
					"AAB00": {nil, 0},
				},
				prefixLen: 5,
			},
			args: args{
				key: "AAG00_5890",
			},
			want:    "AdUnitConfig",
			want1:   50,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Fetcher{
				cb:        tt.fields.cb,
				prefixLen: tt.fields.prefixLen,
			}
			got, err, got1 := f.Execute(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fetcher.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Fetcher.Execute() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Fetcher.Execute() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func CbGetAdUnitConfig(key string) (interface{}, error) {
	data := "AdUnitConfig"
	return data, nil
}
