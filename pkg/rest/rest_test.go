package rest

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRestfulURI(t *testing.T) {
	type args struct {
		inputURI string
	}
	tests := []struct {
		name    string
		args    args
		want    *url.URL
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid tcp",
			args: args{
				inputURI: "tcp://localhost:8080",
			},
			want: &url.URL{
				Scheme: "tcp",
				Host:   "localhost:8080",
			},
			wantErr: assert.NoError,
		},
		{
			name: "valid unix",
			args: args{
				inputURI: "unix:///var/tmp/socket.sock",
			},
			want: &url.URL{
				Scheme: "unix",
				Path:   "/var/tmp/socket.sock",
			},
			wantErr: assert.NoError,
		},
		{
			name: "tcp - no host information",
			args: args{
				inputURI: "tcp://",
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "tcp - path information",
			args: args{
				inputURI: "tcp:///some/path",
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "tcp - no port",
			args: args{
				inputURI: "tcp://localhost",
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "unix - no path",
			args: args{
				inputURI: "unix://",
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "unix - host",
			args: args{
				inputURI: "unix://host",
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "unix - host and path",
			args: args{
				inputURI: "unix://host/and/path",
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRestfulURI(tt.args.inputURI)
			if !tt.wantErr(t, err, fmt.Sprintf("parseRestfulURI(%v)", tt.args.inputURI)) {
				return
			}
			assert.Equalf(t, tt.want, got, "parseRestfulURI(%v)", tt.args.inputURI)
		})
	}
}

func TestToRestScheme(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    ServiceScheme
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid none",
			args: args{
				s: "none",
			},
			want:    None,
			wantErr: assert.NoError,
		},
		{
			name: "valid unix",
			args: args{
				s: "unix",
			},
			want:    Unix,
			wantErr: assert.NoError,
		},
		{
			name: "valid tcp",
			args: args{
				s: "tcp",
			},
			want:    TCP,
			wantErr: assert.NoError,
		},
		{
			name: "invalid input",
			args: args{
				s: "foobar",
			},
			want:    2,
			wantErr: assert.Error,
		},
		{
			name: "case doesnt matter",
			args: args{
				s: "UnIx",
			},
			want:    Unix,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toRestScheme(tt.args.s)
			if !tt.wantErr(t, err, fmt.Sprintf("toRestScheme(%v)", tt.args.s)) {
				return
			}
			assert.Equalf(t, tt.want, got, "toRestScheme(%v)", tt.args.s)
		})
	}
}

func Test_validateRestfulURI(t *testing.T) {
	type args struct {
		inputURI string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid tcp",
			args: args{
				inputURI: "tcp://localhost:8080",
			},
			wantErr: false,
		},
		{
			name: "invalid tcp - no host",
			args: args{
				inputURI: "tcp://",
			},
			wantErr: true,
		},
		{
			name: "invalid scheme",
			args: args{
				inputURI: "http://localhost",
			},
			wantErr: true,
		},
		{
			name: "valid uds",
			args: args{
				inputURI: "unix:///my/socket/goes/here/vfkit.sock",
			},
			wantErr: false,
		},
		{
			name: "invalid uds - no path",
			args: args{
				inputURI: "unix://",
			},
			wantErr: true,
		},
		{
			name: "none",
			args: args{
				inputURI: "none://",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateRestfulURI(tt.args.inputURI); (err != nil) != tt.wantErr {
				t.Errorf("validateRestfulURI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
