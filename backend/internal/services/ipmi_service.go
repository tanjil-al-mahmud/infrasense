package services

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/infrasense/backend/internal/models"
)

// IPMIService handles connections to Generic IPMI/Legacy BMC devices
type IPMIService struct{}

func NewIPMIService() *IPMIService {
	return &IPMIService{}
}

// TestConnection tests IPMI connectivity using lanplus
func (s *IPMIService) TestConnection(ctx context.Context, host string, cred *models.DeviceCredential, password string) models.ConnectionTestResult {
	if cred == nil || cred.Username == nil || *cred.Username == "" {
		return models.ConnectionTestResult{
			Success: false,
			Message: "IPMI requires username and password",
		}
	}

	// Just run an ipmitool chassis status command to verify login
	cmdArgs := []string{
		"-I", "lanplus",
		"-H", host,
		"-U", *cred.Username,
		"-P", password,
		"chassis", "status",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	if err := cmd.Run(); err != nil {
		slog.Error("IPMI test connection failed", "event", "ipmi_test_failed", "host", host, "error", err)
		return models.ConnectionTestResult{
			Success: false,
			Message: fmt.Sprintf("IPMI connection failed: %v", err),
		}
	}

	return models.ConnectionTestResult{
		Success: true,
		Message: "Connected successfully via IPMI 2.0 (lanplus)",
	}
}

// SyncDevice pulls FRU data to populate the device inventory automatically
func (s *IPMIService) SyncDevice(ctx context.Context, host string, cred *models.DeviceCredential, password string) (models.DeviceSyncResult, error) {
	result := models.DeviceSyncResult{
		Success: true,
		Message: "Sync completed via IPMI 2.0",
	}

	cmdArgs := []string{
		"-I", "lanplus",
		"-H", host,
		"-U", *cred.Username,
		"-P", password,
		"fru",
	}

	cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Log error but don't fail entirely, some older BMCs might not return FRU
		slog.Error("IPMI sync fru failed", "event", "ipmi_sync_fru_error", "host", host, "error", err, "output", string(out))
		result.Message = fmt.Sprintf("Sync connected, but FRU read failed: %v", err)
		return result, nil
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Product Manufacturer", "Board Mfg":
			if result.Manufacturer == nil || *result.Manufacturer == "" {
				m := val
				result.Manufacturer = &m
			}
		case "Product Name":
			m := val
			result.Model = &m
		case "Product Serial":
			s := val
			result.SerialNumber = &s
		}
	}

	// Additional attempt to get Firmware Version via BMC info if needed:
	mcCmd := exec.CommandContext(ctx, "ipmitool", "-I", "lanplus", "-H", host, "-U", *cred.Username, "-P", password, "mc", "info")
	if mcOut, mcErr := mcCmd.CombinedOutput(); mcErr == nil {
		for _, line := range strings.Split(string(mcOut), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "Firmware Revision") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					// Add to system info or log it (FirmwareVersion is not a raw field on DeviceSyncResult)
					slog.Info("IPMI found firmware", "host", host, "version", strings.TrimSpace(parts[1]))
				}
				break
			}
		}
	}

	return result, nil
}
