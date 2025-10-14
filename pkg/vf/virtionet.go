package vf

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/crc-org/vfkit/pkg/config"
	"github.com/crc-org/vfkit/pkg/util"

	"github.com/Code-Hex/vz/v3"
	log "github.com/sirupsen/logrus"
)

type VirtioNet struct {
	*config.VirtioNet
	localAddr *net.UnixAddr
}

func localUnixSocketPath(dir string) (string, error) {
	// unix socket endpoints are filesystem paths, but their max length is
	// quite small (a bit over 100 bytes).
	// In this function we try to build a filename which is relatively
	// unique, not easily guessable (to prevent hostile collisions), and
	// short (`os.CreateTemp` filenames are a bit too long)
	//
	// os.Getpid() is unique but guessable. We append a short 16 bit random
	// number to it. We only use hex values to make the representation more
	// compact
	filename := filepath.Join(dir, fmt.Sprintf("vfkit-%x-%x.sock", os.Getpid(), rand.Int31n(0xffff))) //#nosec G404 -- no need for crypto/rand here

	tmpFile, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	// slightly racy, but hopefully this is in a directory only user-writable
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	return tmpFile.Name(), nil
}

// path for unixgram sockets must be less than 104 bytes on macOS
const maxUnixgramPathLen = 104

func (dev *VirtioNet) connectUnixPath() error {

	remoteAddr := net.UnixAddr{
		Name: dev.UnixSocketPath,
		Net:  "unixgram",
	}
	localSocketPath, err := localUnixSocketPath(filepath.Dir(dev.UnixSocketPath))
	if err != nil {
		return err
	}
	if len(localSocketPath) >= maxUnixgramPathLen {
		return fmt.Errorf("unixgram path '%s' is too long: %d >= %d bytes", localSocketPath, len(localSocketPath), maxUnixgramPathLen)
	}
	localAddr := net.UnixAddr{
		Name: localSocketPath,
		Net:  "unixgram",
	}
	conn, err := net.DialUnix("unixgram", &localAddr, &remoteAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	err = rawConn.Control(func(fd uintptr) {
		err := syscall.SetsockoptInt(unixFd(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 1*1024*1024)
		if err != nil {
			return
		}
		err = syscall.SetsockoptInt(unixFd(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024)
		if err != nil {
			return
		}
	})
	if err != nil {
		return err
	}

	/* send vfkit magic so that the remote end can identify our connection attempt */
	if dev.VfkitMagic {
		if _, err := conn.Write([]byte("VFKT")); err != nil {
			return err
		}
		log.Debugf("sent vfkit magic packet (VFKT)")
	} else {
		log.Debugf("skipping vfkit magic packet (disabled)")
	}
	log.Infof("local: %v remote: %v", conn.LocalAddr(), conn.RemoteAddr())

	// This duplicates the connection fd, so we have to close the connection to
	// ensure the network proxy detect when we close dupFd.
	dupFd, err := conn.File()
	if err != nil {
		return err
	}
	if err := conn.Close(); err != nil {
		return err
	}

	dev.Socket = dupFd
	dev.localAddr = &localAddr
	dev.UnixSocketPath = ""
	return nil
}

func (dev *VirtioNet) toVz() (*vz.VirtioNetworkDeviceConfiguration, error) {
	var (
		mac *vz.MACAddress
		err error
	)

	if len(dev.MacAddress) == 0 {
		mac, err = vz.NewRandomLocallyAdministeredMACAddress()
	} else {
		mac, err = vz.NewMACAddress(dev.MacAddress)
	}
	if err != nil {
		return nil, err
	}
	var attachment vz.NetworkDeviceAttachment
	if dev.Socket != nil {
		attachment, err = vz.NewFileHandleNetworkDeviceAttachment(dev.Socket)
	} else {
		attachment, err = vz.NewNATNetworkDeviceAttachment()
	}
	if err != nil {
		return nil, err
	}
	networkConfig, err := vz.NewVirtioNetworkDeviceConfiguration(attachment)
	if err != nil {
		return nil, err
	}
	networkConfig.SetMACAddress(mac)

	return networkConfig, nil
}

func (dev *VirtioNet) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	log.Infof("Adding virtio-net device (nat: %t macAddress: [%s])", dev.Nat, dev.MacAddress)
	if dev.Socket != nil {
		log.Infof("Using fd %d", dev.Socket.Fd())
	}
	if dev.UnixSocketPath != "" {
		log.Infof("Using unix socket %s", dev.UnixSocketPath)
		if err := dev.connectUnixPath(); err != nil {
			return err
		}
	}

	util.RegisterExitHandler(dev.Shutdown)

	netConfig, err := dev.toVz()
	if err != nil {
		return err
	}

	vmConfig.networkDevicesConfiguration = append(vmConfig.networkDevicesConfiguration, netConfig)

	return nil
}

func (dev *VirtioNet) Shutdown() {
	if dev.localAddr != nil {
		log.Debugf("Removing %s", dev.localAddr.Name)
		if err := os.Remove(dev.localAddr.Name); err != nil {
			log.Errorf("failed to remove %s: %v", dev.localAddr.Name, err)
		}
	}
	if dev.Socket != nil {
		log.Debugf("Closing fd %v", dev.Socket.Fd())
		if err := dev.Socket.Close(); err != nil {
			log.Errorf("failed to close fd %d: %v", dev.Socket.Fd(), err)
		}
	}
}
