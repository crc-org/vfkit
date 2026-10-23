package rest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingVMHandler struct {
	storageCalls []string
}

func (h *recordingVMHandler) Inspect(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *recordingVMHandler) GetVMState(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *recordingVMHandler) SetVMState(c *gin.Context) {
	c.Status(http.StatusAccepted)
}

func (h *recordingVMHandler) ListStorageDevices(c *gin.Context) {
	h.storageCalls = append(h.storageCalls, "list")
	c.Status(http.StatusOK)
}

func (h *recordingVMHandler) AttachStorageDevice(c *gin.Context) {
	h.storageCalls = append(h.storageCalls, "attach:"+c.Param("id"))
	c.Status(http.StatusCreated)
}

func (h *recordingVMHandler) DetachStorageDevice(c *gin.Context) {
	h.storageCalls = append(h.storageCalls, "detach:"+c.Param("id"))
	c.Status(http.StatusNoContent)
}

func TestNewServerRegistersStorageRoutes(t *testing.T) {
	handler := &recordingVMHandler{}
	server, err := NewServer(handler, handler, "unix:///tmp/vfkit-test.sock", handler)
	require.NoError(t, err)

	tests := []struct {
		method string
		path   string
		status int
	}{
		{method: http.MethodGet, path: "/vm/storage", status: http.StatusOK},
		{method: http.MethodPut, path: "/vm/storage/cache", status: http.StatusCreated},
		{method: http.MethodDelete, path: "/vm/storage/cache", status: http.StatusNoContent},
	}

	for _, tt := range tests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(tt.method, tt.path, nil)
		server.router.ServeHTTP(recorder, request)
		assert.Equal(t, tt.status, recorder.Code)
	}
	assert.Equal(t, []string{"list", "attach:cache", "detach:cache"}, handler.storageCalls)
}

func TestNewServerSkipsStorageRoutesOnTCP(t *testing.T) {
	handler := &recordingVMHandler{}
	server, err := NewServer(handler, handler, "tcp://127.0.0.1:8080", handler)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/vm/storage", nil))
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Empty(t, handler.storageCalls)
}

func TestNewServerWithoutStorageHandler(t *testing.T) {
	handler := &recordingVMHandler{}
	server, err := NewServer(handler, handler, "unix:///tmp/vfkit-test.sock", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/vm/storage/cache", nil))
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Empty(t, handler.storageCalls)
}

func TestNewUnixListenerIsOwnerOnly(t *testing.T) {
	socketFile, err := os.CreateTemp("/tmp", "vfkit-rest-")
	require.NoError(t, err)
	socketPath := socketFile.Name()
	require.NoError(t, socketFile.Close())
	require.NoError(t, os.Remove(socketPath))
	t.Cleanup(func() { _ = os.Remove(socketPath) })

	listener, err := newUnixListener(socketPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, listener.Close())
	})

	info, err := os.Stat(socketPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

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
