package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/infrasense/backend/internal/auth"
)

// AuthMiddleware validates JWT tokens.
// Supports both "Authorization: Bearer <token>" header and "?token=<token>" query param
// (the query param is needed for SSE streams where browsers cannot set custom headers).
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Prefer Authorization header; fall back to ?token= query param (for SSE)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format", "code": "UNAUTHORIZED"})
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else if q := c.Query("token"); q != "" {
			tokenString = q
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required", "code": "UNAUTHORIZED"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}
