package test

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/crc-org/vfkit/pkg/config"

	"github.com/cavaliergopher/grab/v3"
	"github.com/crc-org/crc/v2/pkg/extract"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func kernelArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return "invalid"
	}
}

func downloadPuipui(destDir string) ([]string, error) {
	const puipuiVersion = "0.0.1"
	var puipuiURL = fmt.Sprintf("https://github.com/Code-Hex/puipui-linux/releases/download/v%s/puipui_linux_v%s_%s.tar.gz", puipuiVersion, puipuiVersion, kernelArch())

	// https://github.com/cavaliergopher/grab/issues/104
	grab.DefaultClient.UserAgent = "vfkit"
	resp, err := grab.Get(destDir, puipuiURL)
	if err != nil {
		return nil, err
	}
	return extract.Uncompress(resp.Filename, destDir)
}

type OsProvider interface {
	Fetch(destDir string) error
	ToVirtualMachine() (*config.VirtualMachine, error)
	SSHConfig() *ssh.ClientConfig
	SSHAccessMethods() []SSHAccessMethod
}

type SSHAccessMethod struct {
	network string
	port    int
}

type PuiPuiProvider struct {
	vmlinuz    string
	initramfs  string
	kernelArgs string
}

func NewPuipuiProvider() *PuiPuiProvider {
	return &PuiPuiProvider{}
}

func findFile(files []string, filename string) (string, error) {
	for _, f := range files {
		if filepath.Base(f) == filename {
			return f, nil
		}
	}

	return "", fmt.Errorf("could not find %s", filename)
}

func ungz(gzFile string) (string, error) {
	reader, err := os.Open(gzFile)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()
	destFile, _ := strings.CutSuffix(gzFile, ".gz")
	writer, err := os.OpenFile(destFile, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	defer writer.Close()

	// https://stackoverflow.com/questions/67327323/g110-potential-dos-vulnerability-via-decompression-bomb-gosec
	for {
		_, err = io.CopyN(writer, gzReader, 1024*1024)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
	}
	return destFile, nil
}

func findKernel(files []string) (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return findFile(files, "bzImage")
	case "arm64":
		compressed, err := findFile(files, "Image.gz")
		if err != nil {
			return "", err
		}
		return ungz(compressed)
	default:
		return "", fmt.Errorf("unsupported architecture '%s'", runtime.GOARCH)
	}
}

func (puipui *PuiPuiProvider) Fetch(destDir string) error {
	log.Infof("downloading puipui to %s", destDir)
	files, err := downloadPuipui(destDir)
	if err != nil {
		return err
	}

	puipui.vmlinuz, err = findKernel(files)
	if err != nil {
		return err
	}
	log.Infof("puipui kernel: %s", puipui.vmlinuz)
	puipui.initramfs, err = findFile(files, "initramfs.cpio.gz")
	if err != nil {
		return err
	}
	log.Infof("puipui initramfs: %s", puipui.initramfs)
	puipui.kernelArgs = "console=hvc0"
	log.Infof("puipui kernel command line: %s", puipui.kernelArgs)

	return nil
}

const puipuiMemoryMiB = 1 * 1024
const puipuiCPUs = 2

func (puipui *PuiPuiProvider) ToVirtualMachine() (*config.VirtualMachine, error) {
	bootloader := config.NewLinuxBootloader(puipui.vmlinuz, puipui.kernelArgs, puipui.initramfs)
	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)

	return vm, nil
}

func (puipui *PuiPuiProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{ssh.Password("passwd")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func (puipui *PuiPuiProvider) SSHAccessMethods() []SSHAccessMethod {
	return []SSHAccessMethod{
		{
			network: "tcp",
			port:    22,
		},
		{
			network: "vsock",
			port:    2222,
		},
	}
}
