package test

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/crc-org/vfkit/pkg/config"
	"github.com/xi2/xz"

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

const puipuiVersion = "1.0.3"

func downloadPuipui(destDir string) ([]string, error) {
	var puipuiURL = fmt.Sprintf("https://github.com/Code-Hex/puipui-linux/releases/download/v%s/puipui_linux_v%s_%s.tar.gz", puipuiVersion, puipuiVersion, kernelArch())

	// https://github.com/cavaliergopher/grab/issues/104
	grab.DefaultClient.UserAgent = "vfkit"
	resp, err := grab.Get(destDir, puipuiURL)
	if err != nil {
		return nil, err
	}
	return extract.Uncompress(context.Background(), resp.Filename, destDir)
}

func downloadFedora(destDir string) (string, error) {
	const fedoraVersion = "42"
	arch := kernelArch()
	release := "1.1"

	baseURL := fmt.Sprintf("https://download.fedoraproject.org/pub/fedora/linux/releases/%s/Cloud/%s/images", fedoraVersion, arch)
	fileName := fmt.Sprintf("Fedora-Cloud-Base-AmazonEC2-%s-%s.%s.raw.xz", fedoraVersion, release, arch)
	fedoraURL := fmt.Sprintf("%s/%s", baseURL, fileName)
	log.Infof("downloading %s", fedoraURL)

	// https://github.com/cavaliergopher/grab/issues/104
	grab.DefaultClient.UserAgent = "vfkit"
	resp, err := grab.Get(destDir, fedoraURL)
	if err != nil {
		return "", err
	}
	return uncompressFedora(resp.Filename, destDir)
}

func uncompressFedora(fileName string, targetDir string) (string, error) {
	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader, err := xz.NewReader(file, 0)
	if err != nil {
		return "", err
	}

	xzCutName, _ := strings.CutSuffix(filepath.Base(file.Name()), ".xz")
	outPath := filepath.Join(targetDir, xzCutName)
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(out, reader)
	if err != nil {
		return "", err
	}

	return outPath, nil
}

type OsProvider interface {
	Fetch(destDir string) error
	ToVirtualMachine() (*config.VirtualMachine, error)
	SSHConfig() *ssh.ClientConfig
	SSHAccessMethods() []SSHAccessMethod
}

type SSHAccessMethod struct {
	network string
	port    uint
}

type PuiPuiProvider struct {
	vmlinuz    string
	initramfs  string
	kernelArgs string
}

func NewPuipuiProvider() *PuiPuiProvider {
	return &PuiPuiProvider{}
}

type FedoraProvider struct {
	diskImage            string
	efiVariableStorePath string
	createVariableStore  bool
}

func NewFedoraProvider() *FedoraProvider {
	return &FedoraProvider{}
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
	log.Infof("downloading puipui v%s to %s", puipuiVersion, destDir)
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
	puipui.kernelArgs = "quiet"
	log.Infof("puipui kernel command line: %s", puipui.kernelArgs)

	return nil
}

func (fedora *FedoraProvider) Fetch(destDir string) error {
	log.Infof("downloading fedora to %s", destDir)
	file, err := downloadFedora(destDir)
	if err != nil {
		return err
	}

	fedora.diskImage = file

	return nil
}

const puipuiMemoryMiB = 1 * 1024
const puipuiCPUs = 2

func (puipui *PuiPuiProvider) ToVirtualMachine() (*config.VirtualMachine, error) {
	bootloader := config.NewLinuxBootloader(puipui.vmlinuz, puipui.kernelArgs, puipui.initramfs)
	vm := config.NewVirtualMachine(puipuiCPUs, puipuiMemoryMiB, bootloader)

	return vm, nil
}

func (fedora *FedoraProvider) ToVirtualMachine() (*config.VirtualMachine, error) {
	bootloader := config.NewEFIBootloader(fedora.efiVariableStorePath, fedora.createVariableStore)
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

func (fedora *FedoraProvider) SSHConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: "vfkituser",
		Auth: []ssh.AuthMethod{ssh.Password("vfkittest")},
		// #nosec 106 -- the host SSH key of the VM will change each time it boots
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

}

func (fedora *FedoraProvider) SSHAccessMethods() []SSHAccessMethod {
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
