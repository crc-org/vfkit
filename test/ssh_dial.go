package test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	vfkithelpers "github.com/crc-org/crc/v2/pkg/drivers/vfkit"
	"github.com/crc-org/vfkit/pkg/sshnc"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const handshakeTimeout = 10 * time.Second

func sshDial(ctx context.Context, scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	var conn net.Conn
	var deadline = handshakeTimeout
	switch scheme {
	case "unix":
		var err error
		// Bound a connection timeout to 10 seconds for each attempt
		// spread over the ctx timeout of 3 minutes.
		d := net.Dialer{
			Timeout: handshakeTimeout,
		}
		conn, err = d.DialContext(ctx, scheme, address)
		if err != nil {
			return nil, err
		}
	case "tcp":
		var err error
		d := sshnc.Dialer{}
		conn, err = d.DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, err
		}
		// give tcp connect + handshake two windows
		deadline += handshakeTimeout
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}

	// Bound the handshake timeout for each attempt
	if err := conn.SetDeadline(time.Now().Add(deadline)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to set handshake deadline: %w", err)
	}

	// Abort the handshake as soon as ctx is done (i.e. 3 minutes are up).
	// If a handshake is blocked in Read, the close will interrupt it immediately.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })

	c, chans, reqs, sshErr := ssh.NewClientConn(conn, address, sshConfig)

	// The attempt is over (successfully or not). Disarm the watcher to
	// avoid closing the conn of an already-completed handshake.
	cancelStopped := stop()

	// Clear the deadline to allow the connection to persist after the handshake.
	_ = conn.SetDeadline(time.Time{})

	if sshErr != nil {
		if errors.Is(sshErr, os.ErrDeadlineExceeded) {
			sshErr = fmt.Errorf("ssh handshake with %s timed out after %s: %w", address, deadline, sshErr)
		}
		_ = conn.Close()
		if c, ok := conn.(sshnc.NcError); ok {
			if msg := c.NcError(); msg != "" {
				sshErr = fmt.Errorf("%w (nc: %s)", sshErr, msg)
			}
		}
		return nil, sshErr
	}

	if !cancelStopped {
		// ctx was already cancelled and close was already started
		// via the AfterFunc path. Report the cancellation rather than
		// returning a client on a connection that is being torn down.
		return nil, ctx.Err()
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func retrySSHDial(processDoneCh chan error, scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	const sshDialTimeout = 3 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), sshDialTimeout)
	defer cancel()
	return sshDialContext(ctx, processDoneCh, scheme, address, sshConfig)
}

func sshDialContext(ctx context.Context, processDoneCh chan error, scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {

	// lastErr is the last error that occurred during the dial process
	// This variable is only used to log the error once when giving up
	// on the overall timeout
	var lastErr error

	// Log the last dial error once when giving up on the overall timeout
	// This will help debug why the ssh dial failed instead of a bare
	// context-deadline error
	timeoutErr := func() error {
		if lastErr != nil {
			log.WithError(lastErr).Error("ssh dial failed; giving up")
		}
		return fmt.Errorf("timeout waiting for SSH: %w", ctx.Err())
	}
	for {
		select {
		case err := <-processDoneCh:
			return nil, vfkitExitedErr(err)
		case <-ctx.Done():
			return nil, timeoutErr()
		default:
			select {
			case err := <-processDoneCh:
				return nil, vfkitExitedErr(err)
			case <-ctx.Done():
				return nil, timeoutErr()
			default:
			}
			client, err := sshDial(ctx, scheme, address, sshConfig)
			if err == nil {
				log.WithFields(log.Fields{"scheme": scheme, "address": address}).Info("ssh dial succeeded")
				return client, nil
			}
			// Only record failures that happened while the budget was still
			// live
			if ctx.Err() == nil {
				lastErr = err
			}
			log.WithError(err).Debug("ssh dial failed; will retry")
		}

		// Back off 1s before the next attempt, but still watch for vfkit exit
		// or overall timeout.
		select {
		case err := <-processDoneCh:
			return nil, vfkitExitedErr(err)
		case <-ctx.Done():
			return nil, timeoutErr()
		case <-time.After(1 * time.Second):
		}
	}
}

func retryIPFromMAC(errCh chan error, macAddress string) (string, error) {
	var (
		err error
		ip  string
	)

	timeout := time.After(10 * time.Second)

	for {
		select {
		case err := <-errCh:
			return "", vfkitExitedErr(err)
		case <-time.After(1 * time.Second):
			ip, err = vfkithelpers.GetIPAddressByMACAddress(macAddress)
			if err == nil {
				log.Infof("found IP address %s for MAC %s", ip, macAddress)
				return ip, nil
			}
		case <-timeout:
			return "", fmt.Errorf("timeout getting IP from MAC: %w", err)
		}
	}
}

func vfkitExitedErr(err error) error {
	if err == nil {
		return errors.New("vfkit process exited before SSH became reachable")
	}
	return fmt.Errorf("vfkit process exited before SSH became reachable: %w", err)
}
