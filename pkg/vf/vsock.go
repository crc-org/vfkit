package vf

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/Code-Hex/vz/v3"
	"inet.af/tcpproxy"
)

func ExposeVsock(vm *vz.VirtualMachine, port uint, vsockPath string, listen bool) error {
	if listen {
		return listenVsock(vm, port, vsockPath)
	} else {
		return connectVsock(vm, port, vsockPath)
	}
}

func ConnectVsockSync(vm *vz.VirtualMachine, port uint) (net.Conn, error) {
	socketDevices := vm.SocketDevices()
	if len(socketDevices) != 1 {
		return nil, fmt.Errorf("VM has too many/not enough virtio-vsock devices (%d)", len(socketDevices))
	}
	vsockDevice := socketDevices[0]

	return vsockDevice.Connect(uint32(port))
}

// connectVsock proxies connections from a host unix socket to a vsock port
// This allows the host to initiate connections to the guest over vsock
func connectVsock(vm *vz.VirtualMachine, port uint, vsockPath string) error {

	var proxy tcpproxy.Proxy
	// listen for connections on the host unix socket
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		parsed, err := url.Parse(laddr)
		if err != nil {
			return nil, err
		}
		switch parsed.Scheme {
		case "unix":
			addr := net.UnixAddr{Net: "unix", Name: parsed.EscapedPath()}
			return net.ListenUnix("unix", &addr)
		default:
			return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
		}
	}

	proxy.AddRoute(fmt.Sprintf("unix://:%s", vsockPath), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("vsock:%d", port),
		// when there's a connection to the unix socket listener, connect to the specified vsock port
		DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, e error) {
			parsed, err := url.Parse(addr)
			if err != nil {
				return nil, err
			}
			switch parsed.Scheme {
			case "vsock":
				return ConnectVsockSync(vm, port)
			default:
				return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
			}
		},
	})
	return proxy.Start()
}

// listenVsock proxies connections from a vsock port to a host unix socket.
// This allows the guest to initiate connections to the host over vsock
func listenVsock(vm *vz.VirtualMachine, port uint, vsockPath string) error {
	var proxy tcpproxy.Proxy
	// listen for connections on the vsock port
	proxy.ListenFunc = func(_, laddr string) (net.Listener, error) {
		parsed, err := url.Parse(laddr)
		if err != nil {
			return nil, err
		}
		switch parsed.Scheme {
		case "vsock":
			port, err := strconv.Atoi(parsed.Port())
			if err != nil {
				return nil, err
			}
			socketDevices := vm.SocketDevices()
			if len(socketDevices) != 1 {
				return nil, fmt.Errorf("VM has too many/not enough virtio-vsock devices (%d)", len(socketDevices))
			}
			return socketDevices[0].Listen(uint32(port))
		default:
			return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
		}
	}

	proxy.AddRoute(fmt.Sprintf("vsock://:%d", port), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("unix:%s", vsockPath),
		// when there's a connection to the vsock listener, connect to the provided unix socket
		DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, e error) {
			parsed, err := url.Parse(addr)
			if err != nil {
				return nil, err
			}
			switch parsed.Scheme {
			case "unix":
				var d net.Dialer
				return d.DialContext(ctx, parsed.Scheme, parsed.Path)
			default:
				return nil, fmt.Errorf("unexpected scheme '%s'", parsed.Scheme)
			}
		},
	})
	//defer proxy.Close()
	return proxy.Start()
}
