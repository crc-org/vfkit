package vf

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/crc-org/vfkit/pkg/config"

	"github.com/Code-Hex/vz/v3"
	log "github.com/sirupsen/logrus"
)

type VirtioNet config.VirtioNet

func localUnixSocketPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(homeDir, "Library", "Application Support", "vfkit")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	tmpFile, err := os.CreateTemp(dir, fmt.Sprintf("net-%d-*.sock", os.Getpid()))
	if err != nil {
		return "", err
	}
	// slightly racy, but this is in a directory only user-writable
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	return tmpFile.Name(), nil
}

func (dev *VirtioNet) connectUnixPath() error {
	remoteAddr := net.UnixAddr{
		Name: dev.UnixSocketPath,
		Net:  "unixgram",
	}
	localSocketPath, err := localUnixSocketPath()
	if err != nil {
		return err
	}
	// FIXME: need to remove localSocketPath at process  exit
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
		if err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 1*1024*1024); err != nil {
			return
		}
		if err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024); err != nil {
			return
		}
	})
	if err != nil {
		return err
	}

	/* send vfkit magic so that the remote end can identify our connection attempt */
	if _, err := conn.Write([]byte("VFKT")); err != nil {
		return err
	}
	log.Infof("local: %v remote: %v", conn.LocalAddr(), conn.RemoteAddr())

	fd, err := conn.File()
	if err != nil {
		return err
	}

	dev.Socket = fd
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

func (dev *VirtioNet) AddToVirtualMachineConfig(vmConfig *vzVirtualMachineConfiguration) error {
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
	netConfig, err := dev.toVz()
	if err != nil {
		return err
	}

	vmConfig.networkDevicesConfiguration = append(vmConfig.networkDevicesConfiguration, netConfig)

	return nil
}
