// Package smart provides SMART disk health monitoring for the KubeNAS Node Agent.
// It wraps smartctl from smartmontools to collect health data and compute
// a normalized health score used for disk placement decisions.
package smart

import (
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strings"
)

// DiskHealth holds SMART-derived health data for a single disk.
type DiskHealth struct {
	DevicePath     string
	Model          string
	SerialNumber   string
	FirmwareVersion string
	OverallHealth  string // PASSED / FAILED
	Temperature    int    // Celsius
	PowerOnHours   int64
	Rotational     bool
	SmartFailed    bool
	IOErrors       int64

	// Key SMART attributes parsed from the attribute table.
	ReallocatedSectors   int64
	PendingSectors       int64
	UncorrectableErrors  int64
	SpinRetryCount       int64
	CommandTimeout       int64

	// NVMe-specific fields.
	NVMeAvailableSparePercent int
	NVMePercentUsed           int
	NVMeMediaErrors           int64

	// HealthScore is a normalized 0.0 (critical) to 1.0 (perfect) score.
	HealthScore float64
}

// smartctlOutput mirrors the JSON schema from `smartctl -a --json`.
type smartctlOutput struct {
	Device struct {
		Name     string `json:"name"`
		Protocol string `json:"protocol"`
	} `json:"device"`
	ModelName       string `json:"model_name"`
	SerialNumber    string `json:"serial_number"`
	FirmwareVersion string `json:"firmware_version"`
	Rotation        struct {
		Rate int `json:"rate"` // 0 = SSD/NVMe
	} `json:"rotation_rate"`
	SmartStatus struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature struct {
		Current int `json:"current"`
	} `json:"temperature"`
	PowerOnTime struct {
		Hours int64 `json:"hours"`
	} `json:"power_on_time"`
	ATASmartAttributes struct {
		Table []struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Value int    `json:"value"`
			Worst int    `json:"worst"`
			Raw   struct {
				Value int64 `json:"value"`
			} `json:"raw"`
			Flags struct {
				Prefailure bool `json:"prefailure"`
			} `json:"flags"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	NVMeSmartHealthInformationLog struct {
		AvailableSpare       int   `json:"available_spare"`
		AvailableSpareThreshold int `json:"available_spare_threshold"`
		PercentageUsed       int   `json:"percentage_used"`
		MediaErrors          int64 `json:"media_errors"`
		NumErrLogEntries     int64 `json:"num_err_log_entries"`
	} `json:"nvme_smart_health_information_log"`
}

// QueryDisk runs smartctl against a device and returns structured health data.
func QueryDisk(devicePath string) (*DiskHealth, error) {
	cmd := exec.Command("smartctl", "-a", "--json=c", devicePath)
	output, err := cmd.Output()
	// smartctl may exit non-zero even with usable data (exit code is a bitmask).
	// We always attempt to parse the JSON regardless of exit code.
	if err != nil && len(output) == 0 {
		return nil, fmt.Errorf("smartctl failed with no output for %s: %w", devicePath, err)
	}

	var raw smartctlOutput
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("parsing smartctl JSON for %s: %w", devicePath, err)
	}

	health := &DiskHealth{
		DevicePath:      devicePath,
		Model:           raw.ModelName,
		SerialNumber:    raw.SerialNumber,
		FirmwareVersion: raw.FirmwareVersion,
		Temperature:     raw.Temperature.Current,
		PowerOnHours:    raw.PowerOnTime.Hours,
		Rotational:      raw.Rotation.Rate > 0,
		SmartFailed:     !raw.SmartStatus.Passed,
	}

	if raw.SmartStatus.Passed {
		health.OverallHealth = "PASSED"
	} else {
		health.OverallHealth = "FAILED"
	}

	// Parse ATA SMART attributes.
	for _, attr := range raw.ATASmartAttributes.Table {
		switch attr.ID {
		case 5:   // Reallocated Sectors Count
			health.ReallocatedSectors = attr.Raw.Value
		case 187: // Reported Uncorrectable Errors
			health.UncorrectableErrors = attr.Raw.Value
		case 196: // Reallocation Event Count
			// tracked via ID 5
		case 197: // Current Pending Sector Count
			health.PendingSectors = attr.Raw.Value
		case 198: // (Offline) Uncorrectable Sector Count
			if attr.Raw.Value > health.UncorrectableErrors {
				health.UncorrectableErrors = attr.Raw.Value
			}
		case 10: // Spin Retry Count
			health.SpinRetryCount = attr.Raw.Value
		case 188: // Command Timeout
			health.CommandTimeout = attr.Raw.Value
		}
	}

	// Parse NVMe-specific fields.
	nvme := raw.NVMeSmartHealthInformationLog
	health.NVMeAvailableSparePercent = nvme.AvailableSpare
	health.NVMePercentUsed = nvme.PercentageUsed
	health.NVMeMediaErrors = nvme.MediaErrors + nvme.NumErrLogEntries

	health.HealthScore = ComputeHealthScore(health)
	return health, nil
}

// ComputeHealthScore calculates a normalized 0.0-1.0 health score from SMART data.
// A score of 1.0 means the disk is perfectly healthy; 0.0 means critically degraded.
//
// Scoring factors:
//   - SMART overall health (pass/fail): major deduction for failure
//   - Reallocated sectors: penalized exponentially
//   - Pending/uncorrectable sectors: heavily penalized
//   - Temperature: penalty for high temps (>55°C)
//   - NVMe spare/wear: linear degradation
func ComputeHealthScore(h *DiskHealth) float64 {
	score := 1.0

	// Hard failure is critical.
	if h.SmartFailed {
		score -= 0.6
	}

	// Reallocated sectors: deduct up to 0.3 on a log scale.
	if h.ReallocatedSectors > 0 {
		penalty := math.Min(0.3, float64(h.ReallocatedSectors)*0.01)
		score -= penalty
	}

	// Pending sectors: each is a sign of imminent failure.
	if h.PendingSectors > 0 {
		penalty := math.Min(0.4, float64(h.PendingSectors)*0.05)
		score -= penalty
	}

	// Uncorrectable errors: severe degradation.
	if h.UncorrectableErrors > 0 {
		penalty := math.Min(0.5, float64(h.UncorrectableErrors)*0.1)
		score -= penalty
	}

	// Temperature penalty for HDDs above 55°C.
	if h.Rotational && h.Temperature > 55 {
		score -= float64(h.Temperature-55) * 0.02
	}

	// NVMe wear deduction.
	if !h.Rotational && h.NVMePercentUsed > 0 {
		score -= float64(h.NVMePercentUsed) * 0.005
	}

	return math.Max(0.0, math.Min(1.0, score))
}

// FormatSMARTSummary returns a one-line human-readable SMART status.
func FormatSMARTSummary(h *DiskHealth) string {
	parts := []string{h.OverallHealth}
	if h.Temperature > 0 {
		parts = append(parts, fmt.Sprintf("temp=%d°C", h.Temperature))
	}
	if h.ReallocatedSectors > 0 {
		parts = append(parts, fmt.Sprintf("reallocated=%d", h.ReallocatedSectors))
	}
	if h.PendingSectors > 0 {
		parts = append(parts, fmt.Sprintf("pending=%d", h.PendingSectors))
	}
	if h.UncorrectableErrors > 0 {
		parts = append(parts, fmt.Sprintf("uncorrectable=%d", h.UncorrectableErrors))
	}
	if h.NVMeMediaErrors > 0 {
		parts = append(parts, fmt.Sprintf("nvme-errors=%d", h.NVMeMediaErrors))
	}
	return strings.Join(parts, " ")
}

// IsSmartAvailable checks whether smartctl is present on the system.
func IsSmartAvailable() bool {
	_, err := exec.LookPath("smartctl")
	return err == nil
}
