package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api/response"
	"github.com/infrasense/backend/internal/auth"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
)

type UserHandler struct {
	userRepo *db.UserRepository
}

func NewUserHandler(userRepo *db.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// userResponse strips the password hash before returning a user.
func userResponse(u *models.User) *models.User {
	u.PasswordHash = ""
	return u
}

// ListUsers handles GET /api/v1/users (admin only)
func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.List(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to list users")
		return
	}
	for i := range users {
		users[i].PasswordHash = ""
	}
	response.Success(c, users)
}

// CreateUser handles POST /api/v1/users (admin only)
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	validRoles := map[string]bool{
		models.RoleAdmin:    true,
		models.RoleOperator: true,
		models.RoleViewer:   true,
	}
	if !validRoles[req.Role] {
		response.BadRequest(c, "Invalid role", "INVALID_ROLE")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		response.InternalError(c, "Failed to hash password")
		return
	}

	callerID, _ := c.Get("user_id")
	callerUUID, _ := callerID.(uuid.UUID)

	email := req.Email
	user := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		PasswordHash: passwordHash,
		Email:        &email,
		Role:         req.Role,
		FullName:     req.FullName,
		Enabled:      req.Enabled,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    &callerUUID,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		response.InternalError(c, "Failed to create user")
		return
	}

	response.Created(c, userResponse(user))
}

// GetUser handles GET /api/v1/users/:id (admin only)
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", "INVALID_ID")
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, userResponse(user))
}

// UpdateUser handles PUT /api/v1/users/:id (admin only)
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", "INVALID_ID")
		return
	}

	var req models.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = &req.Email
	}
	if req.Role != "" {
		validRoles := map[string]bool{
			models.RoleAdmin:    true,
			models.RoleOperator: true,
			models.RoleViewer:   true,
		}
		if !validRoles[req.Role] {
			response.BadRequest(c, "Invalid role", "INVALID_ROLE")
			return
		}
		user.Role = req.Role
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		response.InternalError(c, "Failed to update user")
		return
	}

	response.Success(c, userResponse(user))
}

// DeleteUser handles DELETE /api/v1/users/:id (admin only)
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", "INVALID_ID")
		return
	}

	// Prevent deleting own account
	callerID, _ := c.Get("user_id")
	if callerID.(uuid.UUID) == id {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete your own account", "code": "FORBIDDEN"})
		return
	}

	if err := h.userRepo.Delete(c.Request.Context(), id); err != nil {
		response.NotFound(c, "User not found")
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ChangePassword handles PUT /api/v1/users/:id/password (admin or self)
func (h *UserHandler) ChangePassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user ID", "INVALID_ID")
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	callerID, _ := c.Get("user_id")
	callerUUID := callerID.(uuid.UUID)
	callerRole, _ := c.Get("user_role")

	isSelf := callerUUID == id
	isAdmin := callerRole.(string) == models.RoleAdmin

	if !isSelf && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions", "code": "FORBIDDEN"})
		return
	}

	// If changing own password, verify current password
	if isSelf {
		if req.CurrentPassword == "" {
			response.BadRequest(c, "current_password is required when changing your own password", "INVALID_REQUEST")
			return
		}
		user, err := h.userRepo.GetByID(c.Request.Context(), id)
		if err != nil {
			response.NotFound(c, "User not found")
			return
		}
		if err := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect", "code": "INVALID_PASSWORD"})
			return
		}
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		response.InternalError(c, "Failed to hash password")
		return
	}

	if err := h.userRepo.UpdatePassword(c.Request.Context(), id, newHash); err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, gin.H{"message": "Password updated successfully"})
}

// ChangeOwnPassword handles PUT /api/v1/users/me/password
func (h *UserHandler) ChangeOwnPassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error(), "INVALID_REQUEST")
		return
	}

	if req.CurrentPassword == "" {
		response.BadRequest(c, "current_password is required", "INVALID_REQUEST")
		return
	}

	callerID, _ := c.Get("user_id")
	id := callerID.(uuid.UUID)

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	if err := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect", "code": "INVALID_PASSWORD"})
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		response.InternalError(c, "Failed to hash password")
		return
	}

	if err := h.userRepo.UpdatePassword(c.Request.Context(), id, newHash); err != nil {
		response.InternalError(c, "Failed to update password")
		return
	}

	response.Success(c, gin.H{"message": "Password updated successfully"})
}

// GetMe handles GET /api/v1/users/me
func (h *UserHandler) GetMe(c *gin.Context) {
	callerID, _ := c.Get("user_id")
	id := callerID.(uuid.UUID)

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	response.Success(c, userResponse(user))
}
