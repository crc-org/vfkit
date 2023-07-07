package define

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// InspectResponse is used when responding to a request for
// information about the virtual machine
type InspectResponse struct {
	CPUs   uint   `json:"cpus"`
	Memory uint64 `json:"memory"`
	// Devices []config.VirtioDevice `json:"devices"`
}

// VMState can be used to describe the current state of a VM
// as well as used to request a state change
type VMState struct {
	State string `json:"state"`
}

type StateChange string

const (
	Resume   StateChange = "Resume"
	Pause    StateChange = "Pause"
	Stop     StateChange = "Stop"
	HardStop StateChange = "HardStop"
)

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
	return restURI, err
}

// toRestScheme converts a string to a ServiceScheme
func toRestScheme(s string) (ServiceScheme, error) {
	switch strings.ToUpper(s) {
	case "NONE":
		return None, nil
	case "UNIX":
		return Unix, nil
	case "TCP", "HTTP":
		return TCP, nil
	}
	return None, fmt.Errorf("invalid scheme %s", s)
}

type ServiceScheme int

const (
	TCP ServiceScheme = iota
	Unix
	None
	HTTP
)
