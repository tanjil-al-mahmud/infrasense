package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/infrasense/backend/internal/services"
)

// CollectorHandler handles protocol detection requests.
type ProtocolHandler struct {
	detector *services.ProtocolDetector
}

func NewProtocolHandler() *ProtocolHandler {
	return &ProtocolHandler{detector: services.NewProtocolDetector()}
}

// DetectProtocol handles POST /api/v1/devices/detect-protocol
// Body: {"bmc_ip": "192.168.1.100", "timeout_seconds": 5}
func (h *ProtocolHandler) DetectProtocol(c *gin.Context) {
	var req struct {
		BMCIP          string `json:"bmc_ip" binding:"required"`
		TimeoutSeconds int    `json:"timeout_seconds"`
		Username       string `json:"username,omitempty"`
		Password       string `json:"password,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bmc_ip is required"})
		return
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 5
	}

	result := h.detector.Detect(c.Request.Context(), req.BMCIP, req.Username, req.Password)
	c.JSON(http.StatusOK, result)
}
