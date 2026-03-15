package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api/response"
	"github.com/infrasense/backend/internal/api/validation"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
	"github.com/infrasense/backend/internal/services"
)

type DeviceHandler struct {
	repo              *db.DeviceRepository
	credRepo          *db.DeviceCredentialRepository
	credentialService *services.CredentialService
	redfishService    *services.RedfishService
	ipmiService       *services.IPMIService
	auditService      *services.AuditService
	vmQueryURL        string // base URL for VictoriaMetrics queries (without /api/v1/write)
}

func NewDeviceHandler(repo *db.DeviceRepository, auditService *services.AuditService) *DeviceHandler {
	return &DeviceHandler{
		repo:         repo,
		auditService: auditService,
	}
}

// WithCredentialSupport injects credential dependencies needed for test/sync operations.
func (h *DeviceHandler) WithCredentialSupport(credRepo *db.DeviceCredentialRepository, credSvc *services.CredentialService) *DeviceHandler {
	h.credRepo = credRepo
	h.credentialService = credSvc
	h.redfishService = services.NewRedfishService()
	h.ipmiService = services.NewIPMIService()
	return h
}

// Create handles POST /api/v1/devices
func (h *DeviceHandler) Create(c *gin.Context) {
	var req models.DeviceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := validation.FormatBindingErrors(err)
		log.Printf("validation failed: create device - %v", err)
		response.BadRequest(c, msg, "INVALID_REQUEST")
		return
	}

	// Validate IP addresses
	if err := validation.ValidateIPAddress(req.IPAddress); err != nil {
		log.Printf("validation failed: ip_address - %v", err)
		response.BadRequest(c, fmt.Sprintf("Invalid ip_address: %s", err.Error()), "INVALID_IP_ADDRESS")
		return
	}

	if req.BMCIPAddress != nil && *req.BMCIPAddress != "" {
		if err := validation.ValidateIPAddress(*req.BMCIPAddress); err != nil {
			log.Printf("validation failed: bmc_ip_address - %v", err)
			response.BadRequest(c, fmt.Sprintf("Invalid bmc_ip_address: %s", err.Error()), "INVALID_IP_ADDRESS")
			return
		}
	}

	device, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, "Failed to create device")
		return
	}

	// Log audit event
	userID, _ := c.Get("user_id")
	h.auditService.LogDeviceCreate(c.Request.Context(), userID.(uuid.UUID), device.ID, c.ClientIP(), map[string]interface{}{
		"hostname":    device.Hostname,
		"device_type": device.DeviceType,
	})

	response.Created(c, device)
}

// List handles GET /api/v1/devices
func (h *DeviceHandler) List(c *gin.Context) {
	filter := models.DeviceListFilter{
		Page:     1,
		PageSize: 20,
	}

	// Parse query parameters
	if page := c.Query("page"); page != "" {
		var p int
		if _, err := fmt.Sscanf(page, "%d", &p); err == nil && p > 0 {
			filter.Page = p
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		var ps int
		if _, err := fmt.Sscanf(pageSize, "%d", &ps); err == nil && ps > 0 {
			filter.PageSize = ps
		}
	}

	if deviceType := c.Query("device_type"); deviceType != "" {
		filter.DeviceType = &deviceType
	}

	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	if location := c.Query("location"); location != "" {
		filter.Location = &location
	}

	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}

	devices, total, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "Failed to list devices")
		return
	}

	response.Paginated(c, devices, models.PaginationMeta{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    total,
	})
}

// GetByID handles GET /api/v1/devices/:id
func (h *DeviceHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	device, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "device not found" {
			response.NotFound(c, "Device not found")
			return
		}
		response.InternalError(c, "Failed to get device")
		return
	}

	response.Success(c, device)
}

// Update handles PUT /api/v1/devices/:id
func (h *DeviceHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	var req models.DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		msg := validation.FormatBindingErrors(err)
		log.Printf("validation failed: update device - %v", err)
		response.BadRequest(c, msg, "INVALID_REQUEST")
		return
	}

	// Validate IP addresses if provided
	if req.IPAddress != nil && *req.IPAddress != "" {
		if err := validation.ValidateIPAddress(*req.IPAddress); err != nil {
			log.Printf("validation failed: ip_address - %v", err)
			response.BadRequest(c, fmt.Sprintf("Invalid ip_address: %s", err.Error()), "INVALID_IP_ADDRESS")
			return
		}
	}

	if req.BMCIPAddress != nil && *req.BMCIPAddress != "" {
		if err := validation.ValidateIPAddress(*req.BMCIPAddress); err != nil {
			log.Printf("validation failed: bmc_ip_address - %v", err)
			response.BadRequest(c, fmt.Sprintf("Invalid bmc_ip_address: %s", err.Error()), "INVALID_IP_ADDRESS")
			return
		}
	}

	device, err := h.repo.Update(c.Request.Context(), id, req)
	if err != nil {
		if err.Error() == "device not found" {
			response.NotFound(c, "Device not found")
			return
		}
		response.InternalError(c, "Failed to update device")
		return
	}

	// Log audit event
	userID, _ := c.Get("user_id")
	h.auditService.LogDeviceUpdate(c.Request.Context(), userID.(uuid.UUID), device.ID, c.ClientIP(), map[string]interface{}{
		"hostname": device.Hostname,
	})

	response.Success(c, device)
}

// Delete handles DELETE /api/v1/devices/:id
func (h *DeviceHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	err = h.repo.Delete(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "device not found" {
			response.NotFound(c, "Device not found")
			return
		}
		response.InternalError(c, "Failed to delete device")
		return
	}

	// Log audit event
	userID, _ := c.Get("user_id")
	h.auditService.LogDeviceDelete(c.Request.Context(), userID.(uuid.UUID), id, c.ClientIP(), nil)

	c.JSON(http.StatusOK, gin.H{"message": "Device deleted successfully"})
}

// TestConnection handles POST /api/v1/devices/:id/test-connection
func (h *DeviceHandler) TestConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	device, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "device not found" {
			response.NotFound(c, "Device not found")
			return
		}
		response.InternalError(c, "Failed to get device")
		return
	}

	if device.BMCIPAddress == nil || *device.BMCIPAddress == "" {
		response.BadRequest(c, "Device has no BMC IP address configured", "NO_BMC_IP")
		return
	}

	if h.credRepo == nil || h.credentialService == nil {
		response.InternalError(c, "Credential service not configured")
		return
	}

	cred, err := h.resolveCredential(c.Request.Context(), id, device.DeviceType)
	if err != nil {
		response.BadRequest(c, "No credentials configured for this device. Please add credentials first.", "NO_CREDENTIALS")
		return
	}

	password := ""
	if len(cred.PasswordEncrypted) > 0 {
		decrypted, err := h.credentialService.Decrypt(cred.PasswordEncrypted)
		if err != nil {
			response.InternalError(c, "Failed to decrypt credentials")
			return
		}
		password = decrypted
	}

	var result models.ConnectionTestResult
	if device.Protocol != nil && *device.Protocol == "ipmi" {
		result = h.ipmiService.TestConnection(c.Request.Context(), *device.BMCIPAddress, cred, password)
	} else {
		result = h.redfishService.TestConnection(c.Request.Context(), *device.BMCIPAddress, cred, password)
	}

	// Persist connection status
	connStatus := "connected"
	connErr := ""
	if !result.Success {
		connStatus = "failed"
		connErr = result.Message
	}
	_ = h.repo.UpdateConnectionStatus(c.Request.Context(), id, connStatus, connErr)

	c.JSON(http.StatusOK, result)
}

// SyncDevice handles POST /api/v1/devices/:id/sync
func (h *DeviceHandler) SyncDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	device, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "device not found" {
			response.NotFound(c, "Device not found")
			return
		}
		response.InternalError(c, "Failed to get device")
		return
	}

	if device.BMCIPAddress == nil || *device.BMCIPAddress == "" {
		response.BadRequest(c, "Device has no BMC IP address configured", "NO_BMC_IP")
		return
	}

	if h.credRepo == nil || h.credentialService == nil {
		response.InternalError(c, "Credential service not configured")
		return
	}

	cred, err := h.resolveCredential(c.Request.Context(), id, device.DeviceType)
	if err != nil {
		response.BadRequest(c, "No credentials configured for this device. Please add credentials first.", "NO_CREDENTIALS")
		return
	}

	password := ""
	if len(cred.PasswordEncrypted) > 0 {
		decrypted, err := h.credentialService.Decrypt(cred.PasswordEncrypted)
		if err != nil {
			response.InternalError(c, "Failed to decrypt credentials")
			return
		}
		password = decrypted
	}

	var syncResult models.DeviceSyncResult
	if device.Protocol != nil && *device.Protocol == "ipmi" {
		syncResult, err = h.ipmiService.SyncDevice(c.Request.Context(), *device.BMCIPAddress, cred, password)
	} else {
		syncResult, err = h.redfishService.SyncDevice(c.Request.Context(), *device.BMCIPAddress, cred, password)
	}

	if err != nil {
		_ = h.repo.UpdateSyncResult(c.Request.Context(), id, "failed", err.Error())
		syncResult.Success = false
		syncResult.Message = err.Error()
		c.JSON(http.StatusOK, syncResult)
		return
	}

	_ = h.repo.UpdateSyncResult(c.Request.Context(), id, "connected", "")

	// Persist inventory data collected during sync
	if err := h.repo.UpsertInventory(c.Request.Context(), id, &syncResult); err != nil {
		log.Printf("warn: failed to persist inventory for device %s: %v", id, err)
	}

	// Log audit event
	userID, _ := c.Get("user_id")
	h.auditService.LogDeviceUpdate(c.Request.Context(), userID.(uuid.UUID), device.ID, c.ClientIP(), map[string]interface{}{
		"action": "sync",
	})

	c.JSON(http.StatusOK, syncResult)
}

// resolveCredential looks up credentials for a device, trying the exact device_type first,
// then falling back to the protocol family (redfish, ipmi, snmp_v2c, snmp_v3, proxmox).
// This handles vendor-specific device types like "dell_idrac9_redfish" where credentials
// are stored under the canonical protocol name "redfish".
func (h *DeviceHandler) resolveCredential(ctx context.Context, deviceID uuid.UUID, deviceType string) (*models.DeviceCredential, error) {
	// Try exact match first
	cred, err := h.credRepo.GetByDeviceIDAndProtocol(ctx, deviceID, deviceType)
	if err == nil {
		return cred, nil
	}

	// Derive protocol family from device_type suffix
	protocolFamilies := []string{"redfish", "ipmi", "snmp_v3", "snmp_v2c", "proxmox"}
	for _, proto := range protocolFamilies {
		if strings.HasSuffix(deviceType, proto) || strings.Contains(deviceType, proto) {
			cred, err = h.credRepo.GetByDeviceIDAndProtocol(ctx, deviceID, proto)
			if err == nil {
				return cred, nil
			}
		}
	}

	return nil, fmt.Errorf("no credentials found for device %s (type: %s)", deviceID, deviceType)
}

// WithMetricsSupport injects the VictoriaMetrics query URL for the metrics endpoint.
func (h *DeviceHandler) WithMetricsSupport(vmQueryURL string) *DeviceHandler {
	h.vmQueryURL = vmQueryURL
	return h
}

// GetMetrics handles GET /api/v1/devices/:id/metrics
// Queries VictoriaMetrics for the last 24 hours of data for the given device.
func (h *DeviceHandler) GetMetrics(c *gin.Context) {
	idStr := c.Param("id")
	if _, err := uuid.Parse(idStr); err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	if h.vmQueryURL == "" {
		response.InternalError(c, "Metrics backend not configured")
		return
	}

	end := time.Now()
	start := end.Add(-24 * time.Hour)

	metricNames := []string{
		"infrasense_redfish_temperature_celsius",
		"infrasense_redfish_fan_speed_rpm",
		"infrasense_redfish_psu_status",
		"infrasense_redfish_psu_power_watts",
		"infrasense_redfish_system_health",
		"infrasense_redfish_cpu_health",
		"infrasense_redfish_memory_health",
		"infrasense_redfish_raid_status",
		"infrasense_redfish_disk_health",
	}

	result := make(map[string]interface{})
	client := &http.Client{Timeout: 10 * time.Second}

	for _, metric := range metricNames {
		query := fmt.Sprintf(`%s{device_id="%s"}`, metric, idStr)
		reqURL := fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%d&end=%d&step=60",
			h.vmQueryURL, url.QueryEscape(query), start.Unix(), end.Unix())

		resp, err := client.Get(reqURL)
		if err != nil {
			log.Printf("metrics query error for %s: %v", metric, err)
			continue
		}
		defer resp.Body.Close()

		var vmResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&vmResp); err != nil {
			continue
		}
		if data, ok := vmResp["data"]; ok {
			result[metric] = data
		}
	}

	response.Success(c, result)
}

// PowerControl handles POST /api/v1/devices/:id/power
// Body: {"reset_type": "On|ForceOff|GracefulShutdown|ForceRestart|PowerCycle|GracefulRestart"}
func (h *DeviceHandler) PowerControl(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	var req models.PowerControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body: reset_type is required", "INVALID_REQUEST")
		return
	}

	validResetTypes := map[string]bool{
		"On": true, "ForceOff": true, "GracefulShutdown": true,
		"ForceRestart": true, "PowerCycle": true, "GracefulRestart": true,
	}
	if !validResetTypes[req.ResetType] {
		response.BadRequest(c, fmt.Sprintf("Invalid reset_type '%s'. Valid values: On, ForceOff, GracefulShutdown, ForceRestart, PowerCycle, GracefulRestart", req.ResetType), "INVALID_RESET_TYPE")
		return
	}

	device, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "device not found" {
			response.NotFound(c, "Device not found")
			return
		}
		response.InternalError(c, "Failed to get device")
		return
	}

	if device.BMCIPAddress == nil || *device.BMCIPAddress == "" {
		response.BadRequest(c, "Device has no BMC IP address configured", "NO_BMC_IP")
		return
	}

	cred, err := h.resolveCredential(c.Request.Context(), id, device.DeviceType)
	if err != nil {
		response.BadRequest(c, "No credentials configured for this device", "NO_CREDENTIALS")
		return
	}

	password := ""
	if len(cred.PasswordEncrypted) > 0 {
		decrypted, err := h.credentialService.Decrypt(cred.PasswordEncrypted)
		if err != nil {
			response.InternalError(c, "Failed to decrypt credentials")
			return
		}
		password = decrypted
	}

	result := h.redfishService.PowerControl(c.Request.Context(), *device.BMCIPAddress, cred, password, req.ResetType)

	userID, _ := c.Get("user_id")
	h.auditService.LogDeviceUpdate(c.Request.Context(), userID.(uuid.UUID), device.ID, c.ClientIP(), map[string]interface{}{
		"action":     "power_control",
		"reset_type": req.ResetType,
		"success":    result.Success,
	})

	c.JSON(http.StatusOK, result)
}

// BootControl handles POST /api/v1/devices/:id/boot
// Body: {"target": "Pxe|Cd|Hdd|BiosSetup|None", "once": true}
func (h *DeviceHandler) BootControl(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	var req models.BootControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body: target is required", "INVALID_REQUEST")
		return
	}

	validTargets := map[string]bool{
		"Pxe": true, "Cd": true, "Hdd": true, "BiosSetup": true, "None": true,
	}
	if !validTargets[req.Target] {
		response.BadRequest(c, fmt.Sprintf("Invalid boot target '%s'. Valid values: Pxe, Cd, Hdd, BiosSetup, None", req.Target), "INVALID_BOOT_TARGET")
		return
	}

	device, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "device not found" {
			response.NotFound(c, "Device not found")
			return
		}
		response.InternalError(c, "Failed to get device")
		return
	}

	if device.BMCIPAddress == nil || *device.BMCIPAddress == "" {
		response.BadRequest(c, "Device has no BMC IP address configured", "NO_BMC_IP")
		return
	}

	cred, err := h.resolveCredential(c.Request.Context(), id, device.DeviceType)
	if err != nil {
		response.BadRequest(c, "No credentials configured for this device", "NO_CREDENTIALS")
		return
	}

	password := ""
	if len(cred.PasswordEncrypted) > 0 {
		decrypted, err := h.credentialService.Decrypt(cred.PasswordEncrypted)
		if err != nil {
			response.InternalError(c, "Failed to decrypt credentials")
			return
		}
		password = decrypted
	}

	result := h.redfishService.BootControl(c.Request.Context(), *device.BMCIPAddress, cred, password, req.Target, req.Once)

	userID, _ := c.Get("user_id")
	h.auditService.LogDeviceUpdate(c.Request.Context(), userID.(uuid.UUID), device.ID, c.ClientIP(), map[string]interface{}{
		"action":  "boot_control",
		"target":  req.Target,
		"once":    req.Once,
		"success": result.Success,
	})

	c.JSON(http.StatusOK, result)
}

// DeviceLogEntry is the unified log entry returned by GetDeviceLogs.
type DeviceLogEntry struct {
	ID        string `json:"id"`
	Source    string `json:"source"` // "sel" or "lifecycle"
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// GetDeviceLogs handles GET /api/v1/devices/:id/logs
// Query params: severity (critical|warning|all, default "all"), limit (default 50)
func (h *DeviceHandler) GetDeviceLogs(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	severity := c.DefaultQuery("severity", "all")
	limitStr := c.DefaultQuery("limit", "50")

	limit := 50
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || limit < 1 {
		limit = 50
	}

	// Verify device exists
	if _, err := h.repo.GetByID(c.Request.Context(), id); err != nil {
		if err.Error() == "device not found" {
			response.NotFound(c, "Device not found")
			return
		}
		response.InternalError(c, "Failed to get device")
		return
	}

	// Query sel_entries_json and lifecycle_logs_json from device_inventory
	var selJSON, lcJSON []byte
	err = h.repo.DB().Conn().QueryRowContext(c.Request.Context(),
		`SELECT COALESCE(sel_entries_json, '[]'::jsonb), COALESCE(lifecycle_logs_json, '[]'::jsonb)
		 FROM device_inventory WHERE device_id = $1`, id,
	).Scan(&selJSON, &lcJSON)
	if err != nil {
		// No inventory row — return empty logs
		c.JSON(http.StatusOK, gin.H{"logs": []DeviceLogEntry{}, "total": 0})
		return
	}

	var selEntries []models.SELEntry
	var lcEntries []models.LifecycleLogEntry
	_ = json.Unmarshal(selJSON, &selEntries)
	_ = json.Unmarshal(lcJSON, &lcEntries)

	// Merge into unified log entries
	var logs []DeviceLogEntry
	for _, e := range selEntries {
		logs = append(logs, DeviceLogEntry{
			ID:        e.ID,
			Source:    "sel",
			Severity:  e.Severity,
			Message:   e.Message,
			Timestamp: e.Created,
		})
	}
	for _, e := range lcEntries {
		logs = append(logs, DeviceLogEntry{
			ID:        e.ID,
			Source:    "lifecycle",
			Severity:  e.Severity,
			Message:   e.Message,
			Timestamp: e.Created,
		})
	}

	// Filter by severity if not "all"
	if severity != "all" {
		filtered := logs[:0]
		for _, l := range logs {
			if strings.EqualFold(l.Severity, severity) {
				filtered = append(filtered, l)
			}
		}
		logs = filtered
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp > logs[j].Timestamp
	})

	// Apply limit
	total := len(logs)
	if limit < total {
		logs = logs[:limit]
	}

	if logs == nil {
		logs = []DeviceLogEntry{}
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total})
}

// GetInventory handles GET /api/v1/devices/:id/inventory
func (h *DeviceHandler) GetInventory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid device ID", "INVALID_ID")
		return
	}

	inv, err := h.repo.GetInventory(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "inventory not found" {
			response.NotFound(c, "No inventory data found for this device")
			return
		}
		response.InternalError(c, "Failed to get inventory")
		return
	}

	response.Success(c, inv)
}

