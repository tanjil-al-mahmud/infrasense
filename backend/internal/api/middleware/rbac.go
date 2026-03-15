package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/infrasense/backend/internal/models"
)

// RequireRole middleware checks if user has required role
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userRole := role.(string)

		// Check if user has allowed role
		allowed := false
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions", "code": "FORBIDDEN"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin middleware requires admin role
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}

// RequireAdminOrOperator middleware requires admin or operator role
func RequireAdminOrOperator() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin, models.RoleOperator)
}
