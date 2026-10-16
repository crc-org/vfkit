package test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// retrySSHDial repeatedly dials the VM over SSH, once per second, until a
// connection succeeds or the 60s timeout elapses. SSH may not be reachable
// immediately after boot, so retries are expected during normal startup.
// If errCh reports that the vfkit process itself exited, retrySSHDial
// returns that error immediately instead of waiting out the full timeout.
func retrySSHDial(errCh chan error, scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	var (
		sshClient *ssh.Client
		err       error
	)

	timeout := time.After(60 * time.Second)

	for {
		select {
		case err := <-errCh:
			return nil, err
		case <-time.After(1 * time.Second):
			log.Debugf("trying ssh dial")
			sshClient, err = sshDial(scheme, address, sshConfig)
			if err == nil {
				log.Infof("established SSH connection to %s over %s", address, scheme)
				return sshClient, nil
			}
			log.Debugf("ssh failed: %v", err)
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for SSH: %w", err)
		}
	}
}

// sshDial dials the VM over the given scheme ("unix" for vsock, "tcp" for
// virtio-net). The two schemes are dialed differently: unix-domain sockets
// are dialed directly, since they aren't subject to macOS Local Network
// Privacy (LNP) checks, but tcp connections are routed through
// sshDialViaNetcat instead of a direct net.Dial.
func sshDial(scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	switch scheme {
	case "unix":
		return ssh.Dial("unix", address, sshConfig)
	case "tcp":
		return sshDialViaNetcat(address, sshConfig)
	}
	return nil, fmt.Errorf("unknown scheme: %s", scheme)
}

// sshHandshakeTimeout bounds how long sshDialViaNetcat waits for nc to
// connect and for the SSH handshake over that connection to complete. It
// A watchdog timer below is used to close the connection after the timeout.
const sshHandshakeTimeout = 10 * time.Second

// sshDialViaNetcat dials address by piping the SSH byte stream through the
// macOS /usr/bin/nc binary instead of calling net.Dial directly from the
// test binary.
//
// This exists to work around macOS Local Network Privacy (LNP): LNP
// intercepts outbound connections to local-network addresses (such as the
// VM's 192.168.64.x address) and requires the user to approve a one-time
// permission prompt before the connection is allowed. On an unattended CI
// runner there is no user to approve that prompt, so a direct dial from the
// test binary is silently blocked or hangs. /usr/bin/nc is a separately
// pre-approved system binary, so letting it perform the actual connect()
// call avoids triggering the LNP prompt entirely.
func sshDialViaNetcat(address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", address, err)
	}

	cmd := exec.Command("/usr/bin/nc", host, port)
	// WaitDelay bounds how long cmd.Wait can block in Close cleaning up nc's
	// pipes after it's killed; without it, Wait can hang indefinitely.
	cmd.WaitDelay = 5 * time.Second
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open nc stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open nc stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start /usr/bin/nc: %w", err)
	}

	conn := &ncConn{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}

	// If nc never connects or the handshake takes too long, the watchdog closes
	// the connection.
	watchdog := time.AfterFunc(sshHandshakeTimeout, func() {
		_ = conn.Close()
	})
	defer watchdog.Stop()

	c, chans, reqs, err := ssh.NewClientConn(conn, address, sshConfig)
	if err != nil {
		_ = conn.Close()
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%w (nc: %s)", err, msg)
		}
		return nil, err
	}

	return ssh.NewClient(c, chans, reqs), nil
}

// ncConn wraps a running netcat subprocess to satisfy the net.Conn interface
type ncConn struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	closeOne sync.Once
}

func (c *ncConn) Read(b []byte) (int, error)  { return c.stdout.Read(b) }
func (c *ncConn) Write(b []byte) (int, error) { return c.stdin.Write(b) }

// Close terminates the underlying nc process and closes its pipes. It can be
// called more than once, including concurrently, since both the watchdog and
// the normal error path in sshDialViaNetcat may call it
func (c *ncConn) Close() error {
	c.closeOne.Do(func() {
		_ = c.stdin.Close()
		_ = c.stdout.Close()
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
			_ = c.cmd.Wait()
		}
	})
	return nil
}

// Boilerplate stubs to fulfill the rest of net.Conn
func (c *ncConn) LocalAddr() net.Addr                { return &net.IPAddr{} }
func (c *ncConn) RemoteAddr() net.Addr               { return &net.IPAddr{} }
func (c *ncConn) SetDeadline(t time.Time) error      { return nil }
func (c *ncConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *ncConn) SetWriteDeadline(t time.Time) error { return nil }
