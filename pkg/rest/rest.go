package rest

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/crc-org/vfkit/pkg/util"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// see `man unix`:
// UNIX-domain addresses are variable-length filesystem pathnames of at most 104 characters.
func maxSocketPathLen() int {
	var sockaddr syscall.RawSockaddrUnix
	// sockaddr.Path must end with '\0', it's not relevant for go strings
	return len(sockaddr.Path) - 1
}

type Endpoint struct {
	Host   string
	Path   string
	Scheme ServiceScheme
}

func NewEndpoint(input string) (*Endpoint, error) {
	uri, err := parseRestfulURI(input)
	if err != nil {
		return nil, err
	}
	scheme, err := toRestScheme(uri.Scheme)
	if err != nil {
		return nil, err
	}
	return &Endpoint{
		Host:   uri.Host,
		Path:   uri.Path,
		Scheme: scheme,
	}, nil
}

func (ep *Endpoint) ToCmdLine() ([]string, error) {
	args := []string{"--restful-uri"}
	switch ep.Scheme {
	case Unix:
		args = append(args, fmt.Sprintf("unix://%s", ep.Path))
	case TCP:
		args = append(args, fmt.Sprintf("tcp://%s%s", ep.Host, ep.Path))
	case None:
		return []string{}, nil
	default:
		return []string{}, errors.New("invalid endpoint scheme")
	}
	return args, nil
}

// VFKitService is used for the restful service; it describes
// the variables of the service like host/path but also has
// the router object
type VFKitService struct {
	*Endpoint
	router *gin.Engine
}

// Start initiates the already configured gin service
func (v *VFKitService) Start() {
	var listener net.Listener
	if v.Scheme == Unix {
		// Bind synchronously, before the VM starts creating files, because
		// the umask change in newUnixListener is process-wide.
		unixListener, err := newUnixListener(v.Path)
		if err != nil {
			logrus.Fatal(err)
		}
		listener = unixListener
		util.RegisterExitHandler(func() { os.Remove(v.Path) })
	}
	go func() {
		var err error
		switch v.Scheme {
		case TCP:
			err = v.router.Run(v.Host)
		case Unix:
			err = v.serveUnix(listener)
		}
		logrus.Fatal(err)
	}()
}

// newUnixListener binds the REST socket with owner-only permissions. The
// umask is narrowed while the socket is bound so that it is never reachable
// with wider permissions; the explicit chmod covers filesystems that do not
// apply the umask to sockets.
func newUnixListener(path string) (net.Listener, error) {
	oldMask := syscall.Umask(0077)
	listener, err := net.Listen("unix", path)
	syscall.Umask(oldMask)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("could not restrict unix socket permissions: %w", err)
	}
	return listener, nil
}

func (v *VFKitService) serveUnix(listener net.Listener) error {
	defer listener.Close()
	defer os.Remove(v.Path)
	server := &http.Server{
		Handler:           v.router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.Serve(listener)
}

// NewServer creates a new restful service. storageHandler may be nil; storage
// endpoints are only served on unix endpoints, so it is ignored with a warning
// for TCP endpoints.
func NewServer(
	inspector VirtualMachineInspector,
	stateHandler VirtualMachineStateHandler,
	endpoint string,
	storageHandler VirtualMachineStorageHandler,
) (*VFKitService, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	ep, err := NewEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	err = r.SetTrustedProxies(nil)
	if err != nil {
		return nil, err
	}
	s := VFKitService{
		router:   r,
		Endpoint: ep,
	}

	// Handlers for the restful service.  This is where endpoints are defined.
	r.GET("/vm/state", stateHandler.GetVMState)
	r.POST("/vm/state", stateHandler.SetVMState)
	r.GET("/vm/inspect", inspector.Inspect)
	if storageHandler != nil {
		if ep.Scheme == Unix {
			r.GET("/vm/storage", storageHandler.ListStorageDevices)
			r.PUT("/vm/storage/:id", storageHandler.AttachStorageDevice)
			r.DELETE("/vm/storage/:id", storageHandler.DetachStorageDevice)
		} else {
			logrus.Warn("storage hotplug endpoints are only served on unix REST endpoints, not registering /vm/storage")
		}
	}
	return &s, nil
}

type VirtualMachineInspector interface {
	Inspect(c *gin.Context)
}

type VirtualMachineStateHandler interface {
	GetVMState(c *gin.Context)
	SetVMState(c *gin.Context)
}

type VirtualMachineStorageHandler interface {
	ListStorageDevices(c *gin.Context)
	AttachStorageDevice(c *gin.Context)
	DetachStorageDevice(c *gin.Context)
}

// parseRestfulURI validates the input URI and returns an URL object
func parseRestfulURI(inputURI string) (*url.URL, error) {
	restURI, err := url.ParseRequestURI(inputURI)
	if err != nil {
		return nil, err
	}
	scheme, err := toRestScheme(restURI.Scheme)
	if err != nil {
		return nil, err
	}
	if scheme == TCP && len(restURI.Host) < 1 {
		return nil, errors.New("invalid TCP uri: missing host")
	}
	if scheme == TCP && len(restURI.Path) > 0 {
		return nil, errors.New("invalid TCP uri: path is forbidden")
	}
	if scheme == TCP && restURI.Port() == "" {
		return nil, errors.New("invalid TCP uri: missing port")
	}
	if scheme == Unix && len(restURI.Path) < 1 {
		return nil, errors.New("invalid unix uri: missing path")
	}
	if scheme == Unix && len(restURI.Host) > 0 {
		return nil, errors.New("invalid unix uri: host is forbidden")
	}
	if scheme == Unix && len(restURI.Path) > maxSocketPathLen() {
		return nil, fmt.Errorf("invalid unix uri: socket path length exceeds macOS limits")
	}
	return restURI, err
}

// toRestScheme converts a string to a [ServiceScheme].
func toRestScheme(s string) (ServiceScheme, error) {
	switch s {
	case "none":
		return None, nil
	case "unix":
		return Unix, nil
	case "tcp", "http":
		return TCP, nil
	}
	return None, fmt.Errorf("invalid scheme %s", s)
}
