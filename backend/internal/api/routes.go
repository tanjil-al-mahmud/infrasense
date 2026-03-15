package api

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/infrasense/backend/internal/api/handlers"
	"github.com/infrasense/backend/internal/api/middleware"
	"github.com/infrasense/backend/internal/auth"
	"github.com/infrasense/backend/internal/config"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/services"
)

type Server struct {
	router     *gin.Engine
	db         *db.DB
	jwtService *auth.JWTService
	config     *config.Config
}

func NewServer(database *db.DB, jwtService *auth.JWTService, cfg *config.Config) *Server {
	return &Server{
		router:     gin.Default(),
		db:         database,
		jwtService: jwtService,
		config:     cfg,
	}
}

// startCollectorHealthMonitor runs a background goroutine that marks stale collectors
// as unhealthy every minute. A collector is considered stale if its updated_at timestamp
// is older than 10 minutes.
func (s *Server) startCollectorHealthMonitor(collectorRepo *db.CollectorStatusRepository) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			count, err := collectorRepo.MarkStaleCollectorsUnhealthy(context.Background())
			if err != nil {
				log.Printf("collector health monitor: error marking stale collectors: %v", err)
			} else if count > 0 {
				log.Printf("collector health monitor: marked %d collector(s) as unhealthy", count)
			}
		}
	}()
}

func (s *Server) SetupRoutes() {
	// Apply global middleware
	s.router.Use(middleware.LoggingMiddleware())

	allowedOrigins := []string{"http://localhost:3000", "http://localhost", "http://127.0.0.1"}
	if originsStr := os.Getenv("CORS_ALLOWED_ORIGINS"); originsStr != "" {
		allowedOrigins = append(allowedOrigins, strings.Split(originsStr, ",")...)
	}
	s.router.Use(middleware.CORSMiddleware(allowedOrigins))

	s.router.Use(middleware.RateLimitMiddleware(100, time.Minute))

	s.router.Use(middleware.TimeoutMiddleware(30 * time.Second))
	s.router.Use(middleware.SecurityHeadersMiddleware())

	// Add GZIP compression
	s.router.Use(gzip.Gzip(gzip.DefaultCompression))

	// Health check endpoints (no auth required)
	s.router.GET("/health", func(c *gin.Context) {
		if err := s.db.HealthCheck(); err != nil {
			c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "healthy"})
	})
	s.router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "1.0.0"})
	})

	// API v1 routes
	v1 := s.router.Group("/api/v1")

	// Initialize repositories
	deviceRepo := db.NewDeviceRepository(s.db)
	userRepo := db.NewUserRepository(s.db)
	auditRepo := db.NewAuditRepository(s.db)
	credentialRepo := db.NewDeviceCredentialRepository(s.db)
	groupRepo := db.NewDeviceGroupRepository(s.db)
	alertRuleRepo := db.NewAlertRuleRepository(s.db)
	maintenanceWindowRepo := db.NewMaintenanceWindowRepository(s.db)
	alertAckRepo := db.NewAlertAcknowledgmentRepository(s.db)
	collectorRepo := db.NewCollectorStatusRepository(s.db)

	// Initialize services
	auditService := services.NewAuditService(auditRepo)
	credentialService, _ := services.NewCredentialService(s.config.Auth.EncryptionKey)

	// Derive VictoriaMetrics query base URL from the write URL.
	// Strip any trailing /api/v1/write or /api/v1/import path so we get the base URL.
	vmQueryURL := s.config.Metrics.VictoriaMetricsURL
	for _, suffix := range []string{"/api/v1/write", "/api/v1/import", "/api/v1"} {
		if strings.HasSuffix(vmQueryURL, suffix) {
			vmQueryURL = strings.TrimSuffix(vmQueryURL, suffix)
			break
		}
	}

	// Initialize cache
	apiCache := middleware.NewCache()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, s.jwtService, auditService)
	userHandler := handlers.NewUserHandler(userRepo)
	deviceHandler := handlers.NewDeviceHandler(deviceRepo, auditService).
		WithCredentialSupport(credentialRepo, credentialService).
		WithMetricsSupport(vmQueryURL)
	credentialHandler := handlers.NewDeviceCredentialHandler(credentialRepo, credentialService)
	groupHandler := handlers.NewDeviceGroupHandler(groupRepo)
	streamHandler := handlers.NewStreamHandler(deviceRepo, credentialRepo, credentialService)
	protocolHandler := handlers.NewProtocolHandler()
	alertRuleHandler := handlers.NewAlertRuleHandler(alertRuleRepo, auditService)
	maintenanceWindowHandler := handlers.NewMaintenanceWindowHandler(maintenanceWindowRepo, auditService)
	alertHandler := handlers.NewAlertHandler(alertAckRepo, auditService)
	collectorHandler := handlers.NewCollectorHandler(collectorRepo)

	// Start background collector health monitor
	s.startCollectorHealthMonitor(collectorRepo)

	// Public routes (no authentication required)
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
	}

	// Protected routes (authentication required)
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(s.jwtService))
	{
		// Auth routes
		protected.GET("/auth/me", authHandler.Me)
		protected.POST("/auth/logout", authHandler.Logout)

		// User management routes
		users := protected.Group("/users")
		{
			users.GET("/me", userHandler.GetMe)
			users.PUT("/me/password", userHandler.ChangeOwnPassword)
			users.GET("", middleware.RequireAdmin(), userHandler.ListUsers)
			users.POST("", middleware.RequireAdmin(), userHandler.CreateUser)
			users.GET("/:id", middleware.RequireAdmin(), userHandler.GetUser)
			users.PUT("/:id", middleware.RequireAdmin(), userHandler.UpdateUser)
			users.DELETE("/:id", middleware.RequireAdmin(), userHandler.DeleteUser)
			users.PUT("/:id/password", userHandler.ChangePassword)
		}

		// Device routes
		devices := protected.Group("/devices")
		{
			devices.GET("", middleware.CacheMiddleware(apiCache, 1*time.Minute), deviceHandler.List)
			devices.GET("/:id", deviceHandler.GetByID)
			devices.POST("", middleware.RequireAdminOrOperator(), deviceHandler.Create)
			devices.PUT("/:id", middleware.RequireAdminOrOperator(), deviceHandler.Update)
			devices.DELETE("/:id", middleware.RequireAdminOrOperator(), deviceHandler.Delete)

			// Device credentials routes
			devices.POST("/:id/credentials", middleware.RequireAdminOrOperator(), credentialHandler.Create)
			devices.PUT("/:id/credentials", middleware.RequireAdminOrOperator(), credentialHandler.Update)
			devices.DELETE("/:id/credentials", middleware.RequireAdminOrOperator(), credentialHandler.Delete)

			// Device BMC operations
			devices.POST("/:id/test-connection", middleware.RequireAdminOrOperator(), deviceHandler.TestConnection)
			devices.POST("/:id/sync", middleware.RequireAdminOrOperator(), deviceHandler.SyncDevice)
			devices.POST("/:id/power", middleware.RequireAdminOrOperator(), deviceHandler.PowerControl)
			devices.POST("/:id/boot", middleware.RequireAdminOrOperator(), deviceHandler.BootControl)

			// Device metrics, logs and inventory
			devices.GET("/:id/metrics", deviceHandler.GetMetrics)
			devices.GET("/:id/logs", deviceHandler.GetDeviceLogs)
			devices.GET("/:id/inventory", deviceHandler.GetInventory)
			devices.GET("/:id/stream", streamHandler.StreamTelemetry)
			devices.POST("/detect-protocol", middleware.RequireAdminOrOperator(), protocolHandler.DetectProtocol)
		}

		// Device groups routes
		groups := protected.Group("/device-groups")
		{
			groups.GET("", middleware.CacheMiddleware(apiCache, 1*time.Minute), groupHandler.List)
			groups.GET("/:id", groupHandler.GetByID)
			groups.POST("", middleware.RequireAdminOrOperator(), groupHandler.Create)
			groups.PUT("/:id", middleware.RequireAdminOrOperator(), groupHandler.Update)
			groups.DELETE("/:id", middleware.RequireAdminOrOperator(), groupHandler.Delete)
			groups.POST("/:id/devices", middleware.RequireAdminOrOperator(), groupHandler.AddDevice)
			groups.DELETE("/:id/devices/:deviceId", middleware.RequireAdminOrOperator(), groupHandler.RemoveDevice)
		}

		// Alert rules routes
		alertRules := protected.Group("/alert-rules")
		{
			alertRules.GET("", middleware.CacheMiddleware(apiCache, 1*time.Minute), alertRuleHandler.List)
			alertRules.GET("/:id", alertRuleHandler.GetByID)
			alertRules.POST("", middleware.RequireAdminOrOperator(), alertRuleHandler.Create)
			alertRules.PUT("/:id", middleware.RequireAdminOrOperator(), alertRuleHandler.Update)
			alertRules.DELETE("/:id", middleware.RequireAdminOrOperator(), alertRuleHandler.Delete)
		}

		// Maintenance windows routes
		maintenanceWindows := protected.Group("/maintenance-windows")
		{
			maintenanceWindows.GET("", maintenanceWindowHandler.List)
			maintenanceWindows.POST("", middleware.RequireAdminOrOperator(), maintenanceWindowHandler.Create)
			maintenanceWindows.DELETE("/:id", middleware.RequireAdminOrOperator(), maintenanceWindowHandler.Delete)
		}

		// Alert routes
		alerts := protected.Group("/alerts")
		{
			alerts.GET("", alertHandler.ListActive)
			alerts.GET("/history", alertHandler.ListHistory)
			alerts.POST("/:id/acknowledge", alertHandler.Acknowledge)
		}

		// Collector routes
		collectors := protected.Group("/collectors")
		{
			collectors.GET("", collectorHandler.List)
			collectors.GET("/:id", collectorHandler.GetByID)
		}
	}
}

func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
