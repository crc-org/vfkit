package vf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/Code-Hex/vz"
	"golang.org/x/sys/unix"
	"inet.af/tcpproxy"
)

func ExposeVsock(vm *vz.VirtualMachine, port uint, vsockPath string) error {
	var proxy tcpproxy.Proxy
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
			return Listen(socketDevices[0], uint32(port))
		default:
			return nil, errors.New(fmt.Sprintf("unexpected scheme '%s'", parsed.Scheme))
		}
	}

	proxy.AddRoute(fmt.Sprintf("vsock://:%d", port), &tcpproxy.DialProxy{
		Addr: fmt.Sprintf("unix:%s", vsockPath),
		DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, e error) {
			fmt.Println("DialContext:", network, addr)
			parsed, err := url.Parse(addr)
			if err != nil {
				return nil, err
			}
			switch parsed.Scheme {
			case "unix":
				var d net.Dialer
				return d.DialContext(ctx, parsed.Scheme, parsed.Path)
			default:
				return nil, errors.New(fmt.Sprintf("unexpected scheme '%s'", parsed.Scheme))
			}
		},
	})
	//defer proxy.Close()
	return proxy.Start()
}

type dup struct {
	conn *vz.VirtioSocketConnection
	err  error
}

type Listener struct {
	port            uint32
	incomingConnsCh chan dup
}

func Listen(v *vz.VirtioSocketDevice, port uint32) (net.Listener, error) {
	// for a given device, we should only use one instance of *VirtioSocketListener
	listener := &Listener{
		port:            port,
		incomingConnsCh: make(chan dup, 1),
	}
	shouldAcceptConn := func(conn *vz.VirtioSocketConnection, err error) {
		listener.incomingConnsCh <- dup{conn, err}
	}

	virtioSocketListener := vz.NewVirtioSocketListener(shouldAcceptConn)
	v.SetSocketListenerForPort(virtioSocketListener, port)
	return listener, nil
}

func (l *Listener) Accept() (net.Conn, error) {
	dup := <-l.incomingConnsCh
	return dup.conn, dup.err
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return &vz.Addr{
		CID:  unix.VMADDR_CID_HOST,
		Port: l.port,
	}
}

func (l *Listener) Close() error {
	// need to close incomingConns and cleanly exit the associated go func when this happens
	// also need to disconnect from port
	return nil
}
