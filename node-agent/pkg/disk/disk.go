// Package disk provides disk discovery, formatting, and mount management
// for the KubeNAS Node Agent running on Kubernetes/OpenShift nodes.
package disk

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DeviceInfo contains metadata for a block device discovered on the host.
type Device struct {
	Name       string   `json:"name"`
	DevicePath string   `json:"devicePath"`
	SizeBytes  int64    `json:"sizeBytes"`
	Rotational bool     `json:"rotational"`
	Model      string   `json:"model"`
	Serial     string   `json:"serial"`
	Filesystem string   `json:"filesystem"`
	MountPoint string   `json:"mountPoint"`
	Children   []string `json:"children"`
}

// lsblkOutput mirrors the JSON structure from `lsblk --json`.
type lsblkOutput struct {
	Blockdevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name       string        `json:"name"`
	Size       string        `json:"size"`
	Rota       bool          `json:"rota"`
	Model      string        `json:"model,omitempty"`
	Serial     string        `json:"serial,omitempty"`
	Fstype     string        `json:"fstype,omitempty"`
	Mountpoint string        `json:"mountpoint,omitempty"`
	Children   []lsblkDevice `json:"children,omitempty"`
}

// DiscoverDevices runs lsblk and returns all block devices on the host.
func DiscoverDevices() ([]Device, error) {
	cmd := exec.Command("lsblk", "--json", "--output",
		"NAME,SIZE,ROTA,MODEL,SERIAL,FSTYPE,MOUNTPOINT", "--bytes", "--nodeps")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk failed: %w", err)
	}

	var lsblk lsblkOutput
	if err := json.Unmarshal(output, &lsblk); err != nil {
		return nil, fmt.Errorf("parsing lsblk output: %w", err)
	}

	var devices []Device
	for _, d := range lsblk.Blockdevices {
		// Skip loop devices, CD-ROM, etc.
		if strings.HasPrefix(d.Name, "loop") || strings.HasPrefix(d.Name, "sr") {
			continue
		}

		sizeBytes, _ := strconv.ParseInt(d.Size, 10, 64)
		devices = append(devices, Device{
			Name:       d.Name,
			DevicePath: "/dev/" + d.Name,
			SizeBytes:  sizeBytes,
			Rotational: d.Rota,
			Model:      strings.TrimSpace(d.Model),
			Serial:     strings.TrimSpace(d.Serial),
			Filesystem: d.Fstype,
			MountPoint: d.Mountpoint,
		})
	}

	return devices, nil
}

// GetDeviceFilesystem returns the filesystem type on a device using blkid.
func GetDeviceFilesystem(devicePath string) (string, error) {
	cmd := exec.Command("blkid", "-s", "TYPE", "-o", "value", devicePath)
	output, err := cmd.Output()
	if err != nil {
		// blkid returns exit code 2 for no filesystem found.
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

// FormatDevice formats a block device with the given filesystem.
// This is destructive — caller must ensure no data exists or is intended.
func FormatDevice(devicePath, filesystem string) error {
	var cmd *exec.Cmd
	switch filesystem {
	case "xfs":
		cmd = exec.Command("mkfs.xfs", "-f", devicePath)
	case "ext4":
		cmd = exec.Command("mkfs.ext4", "-F", devicePath)
	case "btrfs":
		cmd = exec.Command("mkfs.btrfs", "-f", devicePath)
	default:
		return fmt.Errorf("unsupported filesystem %q", filesystem)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mkfs %s on %s: %w\nstderr: %s", filesystem, devicePath, err, stderr.String())
	}
	return nil
}

// MountDevice mounts a block device at the given path.
// Creates the mount point directory if it does not exist.
func MountDevice(devicePath, mountPoint, filesystem string) error {
	if err := os.MkdirAll(mountPoint, 0750); err != nil {
		return fmt.Errorf("creating mount point %s: %w", mountPoint, err)
	}

	if IsMounted(mountPoint) {
		return nil // Already mounted, idempotent.
	}

	args := []string{"-t", filesystem}
	if filesystem == "xfs" {
		args = append(args, "-o", "noatime,nodiratime")
	}
	args = append(args, devicePath, mountPoint)

	cmd := exec.Command("mount", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mounting %s at %s: %w\nstderr: %s", devicePath, mountPoint, err, stderr.String())
	}
	return nil
}

// UnmountDevice unmounts the filesystem at the given mount point.
func UnmountDevice(mountPoint string) error {
	if !IsMounted(mountPoint) {
		return nil // Not mounted, nothing to do.
	}

	cmd := exec.Command("umount", "-l", mountPoint)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unmounting %s: %w\nstderr: %s", mountPoint, err, stderr.String())
	}
	return nil
}

// IsMounted returns true if the given path has an active mount.
func IsMounted(mountPoint string) bool {
	absPath, err := filepath.Abs(mountPoint)
	if err != nil {
		return false
	}

	f, err := os.Open("/proc/mounts")
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == absPath {
			return true
		}
	}
	return false
}

// GetDiskUsage returns used and available bytes for a mount point.
func GetDiskUsage(mountPoint string) (usedBytes, availableBytes int64, err error) {
	cmd := exec.Command("df", "--block-size=1", "--output=used,avail", mountPoint)
	output, runErr := cmd.Output()
	if runErr != nil {
		return 0, 0, fmt.Errorf("df on %s: %w", mountPoint, runErr)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("unexpected df output: %s", string(output))
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("unexpected df fields: %s", lines[1])
	}

	used, _ := strconv.ParseInt(fields[0], 10, 64)
	avail, _ := strconv.ParseInt(fields[1], 10, 64)
	return used, avail, nil
}

// MountMergerFS creates or updates a mergerfs union mount.
// Branches are joined with ':' as per mergerfs convention.
func MountMergerFS(branches []string, mountPoint, createPolicy, minFreeSpace, extraOpts string) error {
	if err := os.MkdirAll(mountPoint, 0750); err != nil {
		return fmt.Errorf("creating pool mount point %s: %w", mountPoint, err)
	}

	// Unmount existing mergerfs at this path before remounting (for updates).
	if IsMounted(mountPoint) {
		_ = UnmountDevice(mountPoint)
	}

	branchStr := strings.Join(branches, ":")
	opts := fmt.Sprintf("defaults,allow_other,category.create=%s,minfreespace=%s,fsname=mergerfs",
		createPolicy, minFreeSpace)
	if extraOpts != "" {
		opts += "," + extraOpts
	}

	cmd := exec.Command("mergerfs", "-o", opts, branchStr, mountPoint)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mergerfs mount failed: %w\nstderr: %s", err, stderr.String())
	}
	return nil
}

// EnsureFstabEntry writes or updates a /etc/fstab entry for persistence across reboots.
// On CoreOS/FCOS nodes with immutable root, this should write to /etc/fstab.d/ if available.
func EnsureFstabEntry(devicePath, mountPoint, filesystem, options string) error {
	entry := fmt.Sprintf("%s\t%s\t%s\t%s\t0\t0\n", devicePath, mountPoint, filesystem, options)

	// Read existing fstab.
	content, err := os.ReadFile("/etc/fstab")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading /etc/fstab: %w", err)
	}

	// Check if entry already exists.
	if strings.Contains(string(content), devicePath) {
		return nil
	}

	f, err := os.OpenFile("/etc/fstab", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening /etc/fstab for writing: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(entry)
	return err
}
