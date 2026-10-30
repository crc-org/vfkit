package test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	vfkithelpers "github.com/crc-org/crc/v2/pkg/drivers/vfkit"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const handshakeTimeout = 10 * time.Second

func sshDial(ctx context.Context, scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	var d net.Dialer

	// Bound a connection timeout to 10 seconds for each attempt
	// spread over the ctx timeout of 3 minutes.
	d.Timeout = handshakeTimeout
	conn, err := d.DialContext(ctx, scheme, address)
	if err != nil {
		return nil, err
	}

	// Bound the handshake timeout to 10 seconds for each attempt
	if err := conn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to set handshake deadline: %w", err)
	}

	// Abort the handshake as soon as ctx is done (i.e. 3 minutes are up).
	// If a handshake is blocked in Read, the close will interrupt it immediately.
	stopCancelWatch := context.AfterFunc(ctx, func() { _ = conn.Close() })

	c, chans, reqs, err := ssh.NewClientConn(conn, address, sshConfig)

	// The attempt is over (successfully or not). Disarm the watcher to
	// avoid closing the conn of an already-completed handshake.
	cancelWatchStopped := stopCancelWatch()

	// Clear the deadline to allow the connection to persist after the handshake.
	_ = conn.SetDeadline(time.Time{})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !cancelWatchStopped {
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
	var lastErr error
	// Log the last dial error once when giving up on the overall timeout
	// This will help debug why the ssh dial failed instead of a bare
	// context-deadline error
	timeoutErr := func() error {
		if lastErr != nil {
			log.WithError(lastErr).Error("ssh dial failed; giving up")
		}
		return fmt.Errorf("timeout waiting for SSH: %v", ctx.Err())
	}
	for {
		select {
		case err := <-processDoneCh:
			return nil, vfkitExitedErr(err)
		case <-ctx.Done():
			return nil, timeoutErr()
		default:
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
