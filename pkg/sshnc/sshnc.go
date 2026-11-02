package sshnc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

type ncAddr string

func (a ncAddr) Network() string {
	return "nc"
}

func (a ncAddr) String() string {
	return string(a)
}

type Dialer struct {
	// LocalAddr, if non-nil, is the source address nc binds for the
	// connection: its IP is passed to nc -s, and its port, if nonzero, to
	// nc -p. It is also what the connection reports from LocalAddr since
	// nc owns the actual socket.
	LocalAddr *net.TCPAddr
}

// DialContext spawns nc and returns immediately. The returned conn is not
// tied to ctx which is only checked for cancellation before nc is spawned,
// and the nc process runs under its own lifetime until the conn is closed.
func (d *Dialer) DialContext(ctx context.Context, scheme, address string) (net.Conn, error) {
	// Don't dial if the ctx has already errored.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if scheme != "tcp" {
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", address, err)
	}

	ncArgs := []string{}
	if d.LocalAddr != nil {
		if d.LocalAddr.IP == nil {
			return nil, errors.New("sshnc: Dialer.LocalAddr has no IP")
		}
		ncArgs = append(ncArgs, "-s", d.LocalAddr.IP.String())
		if d.LocalAddr.Port != 0 {
			ncArgs = append(ncArgs, "-p", strconv.Itoa(d.LocalAddr.Port))
		}
	}
	ncArgs = append(ncArgs, host, port)

	// Start a new context for nc process management
	// which is independent of the ctx passed to DialContext
	processCtx, processCtxCancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(processCtx, "/usr/bin/nc", ncArgs...)
	// WaitDelay bounds how long Wait can block on pipe cleanup after cmd exits
	cmd.WaitDelay = 5 * time.Second

	stdin, err := cmd.StdinPipe()
	if err != nil {
		processCtxCancel()
		return nil, fmt.Errorf("failed to open nc stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		processCtxCancel()
		return nil, fmt.Errorf("failed to open nc stdout pipe: %w", err)
	}

	conn := &ncConn{
		cancel: processCtxCancel,
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: bytes.Buffer{},
		raddr:  address,
		laddr:  d.LocalAddr,
	}
	cmd.Stderr = &conn.stderr

	if err := conn.cmd.Start(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// This implements the net.Conn interface for an ssh.ClientConn
// that enwraps a reference to the "nc" system binary
// to allow connections to be made via nc to avoid LNP errors
// on CI
type ncConn struct {
	raddr     string
	laddr     *net.TCPAddr
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    bytes.Buffer
	closeOnce sync.Once
	cancel    context.CancelFunc
	ncErrMsg  string // snapshot of stderr, taken in Close
}

func (c *ncConn) Read(b []byte) (int, error)  { return c.stdout.Read(b) }
func (c *ncConn) Write(b []byte) (int, error) { return c.stdin.Write(b) }

// Close terminates the underlying nc process and closes its pipes. It is safe
// to call more than once, including concurrently.
func (c *ncConn) Close() error {
	c.closeOnce.Do(func() {
		c.cancel()
		_ = c.stdin.Close()
		_ = c.stdout.Close()
		if c.cmd.Process != nil {
			_ = c.cmd.Wait()
		}
		c.ncErrMsg = c.stderr.String()
	})
	return nil
}

// SetDeadline sets the read and write deadlines.
func (c *ncConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

// SetReadDeadline sets the read deadline.
// It falls back to ErrUnsupported if the stdout is not a *os.File.
func (c *ncConn) SetReadDeadline(t time.Time) error {
	if f, ok := c.stdout.(*os.File); ok {
		return f.SetReadDeadline(t)
	}
	return errors.ErrUnsupported
}

// SetWriteDeadline sets the write deadline.
// It falls back to ErrUnsupported if the stdin is not a *os.File.
func (c *ncConn) SetWriteDeadline(t time.Time) error {
	if f, ok := c.stdin.(*os.File); ok {
		return f.SetWriteDeadline(t)
	}
	return errors.ErrUnsupported
}

// LocalAddr/RemoteAddr fulfill the rest of net.Conn.
func (c *ncConn) LocalAddr() net.Addr {
	if c.laddr != nil {
		return c.laddr
	}
	return ncAddr("nc")
}
func (c *ncConn) RemoteAddr() net.Addr { return ncAddr(c.raddr) }

type NcError interface {
	NcError() string
}

func (c *ncConn) NcError() string {
	return c.ncErrMsg
}
