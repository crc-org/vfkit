package rest

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Code-Hex/vz/v4"
	"github.com/crc-org/vfkit/pkg/rest/define"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SetVirtioFsShare replaces the directories exposed by the running virtio-fs device whose mount
// tag is in the URL. The guest keeps its mount and sees the new directories.
func (vm *VzVirtualMachine) SetVirtioFsShare(c *gin.Context) {
	var req define.VirtioFsShare
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateVirtioFsShare(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tag := c.Param("mountTag")
	var device *vz.VirtioFileSystemDevice
	for _, candidate := range vm.DirectorySharingDevices() {
		if fs, ok := candidate.(*vz.VirtioFileSystemDevice); ok && fs.Tag() == tag {
			device = fs
			break
		}
	}
	if device == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("no virtio-fs device with mount tag %q", tag)})
		return
	}
	share, err := virtioFsShareToVz(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := device.SetDirectoryShare(share); err != nil {
		logrus.Errorf("failed to replace virtio-fs share %q: %v", tag, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusAccepted)
}

func validateVirtioFsShare(req define.VirtioFsShare) error {
	if (req.SharedDir == "") == (req.Shares == nil) {
		return errors.New("set exactly one of sharedDir or shares")
	}
	if req.Shares == nil {
		return validateSharedDir(req.SharedDir)
	}
	for name, dir := range req.Shares {
		if name == "" || name == "." || name == ".." || strings.ContainsRune(name, '/') {
			return fmt.Errorf("invalid share name %q", name)
		}
		if err := validateSharedDir(dir.SharedDir); err != nil {
			return err
		}
	}
	return nil
}

func validateSharedDir(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("sharedDir must be an absolute path: %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("sharedDir is not a directory: %q", path)
	}
	return nil
}

func virtioFsShareToVz(req define.VirtioFsShare) (vz.DirectoryShare, error) {
	if req.Shares == nil {
		dir, err := vz.NewSharedDirectory(req.SharedDir, req.ReadOnly)
		if err != nil {
			return nil, err
		}
		return vz.NewSingleDirectoryShare(dir)
	}
	dirs := make(map[string]*vz.SharedDirectory, len(req.Shares))
	for name, share := range req.Shares {
		dir, err := vz.NewSharedDirectory(share.SharedDir, share.ReadOnly)
		if err != nil {
			return nil, err
		}
		dirs[name] = dir
	}
	return vz.NewMultipleDirectoryShare(dirs)
}
