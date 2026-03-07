package csi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Driver struct {
	PoolRoot string
}

func NewDriver() *Driver {
	return &Driver{PoolRoot: "/mnt/kubenas/pools"}
}

func (d *Driver) CreateVolume(pool, path string) (string, error) {
	volPath := filepath.Join(d.PoolRoot, pool, path)
	if err := os.MkdirAll(volPath, 0o755); err != nil {
		return "", err
	}
	return volPath, nil
}

func (d *Driver) DeleteVolume(pool, path string) error {
	return os.RemoveAll(filepath.Join(d.PoolRoot, pool, path))
}

func (d *Driver) NodePublishVolume(pool, path, targetPath string) error {
	source := filepath.Join(d.PoolRoot, pool, path)
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return err
	}
	cmd := exec.Command("mount", "--bind", source, targetPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("bind mount failed: %w (%s)", err, string(out))
	}
	return nil
}

func (d *Driver) NodeUnpublishVolume(targetPath string) error {
	cmd := exec.Command("umount", targetPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unmount failed: %w (%s)", err, string(out))
	}
	return nil
}
