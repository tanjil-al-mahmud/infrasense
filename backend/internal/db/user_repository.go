package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/models"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Conn returns the underlying *sql.DB for direct query access.
func (r *UserRepository) Conn() *sql.DB {
	return r.db.conn
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, password_hash, email, role, enabled, created_at, updated_at, full_name, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.conn.ExecContext(
		ctx, query,
		user.ID, user.Username, user.PasswordHash, user.Email,
		user.Role, user.Enabled, user.CreatedAt, user.UpdatedAt,
		user.FullName, user.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, username, password_hash, email, role, enabled, created_at, updated_at,
		       full_name, last_login_at, created_by
		FROM users
		WHERE username = $1
	`

	var fullName sql.NullString
	var lastLoginAt sql.NullTime
	var createdBy uuid.NullUUID

	err := r.db.conn.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
		&fullName, &lastLoginAt, &createdBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.FullName = fullName.String
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if createdBy.Valid {
		user.CreatedBy = &createdBy.UUID
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, username, password_hash, email, role, enabled, created_at, updated_at,
		       full_name, last_login_at, created_by
		FROM users
		WHERE id = $1
	`

	var fullName sql.NullString
	var lastLoginAt sql.NullTime
	var createdBy uuid.NullUUID

	err := r.db.conn.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
		&fullName, &lastLoginAt, &createdBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.FullName = fullName.String
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if createdBy.Valid {
		user.CreatedBy = &createdBy.UUID
	}

	return user, nil
}

// List retrieves all users
func (r *UserRepository) List(ctx context.Context) ([]models.User, error) {
	query := `
		SELECT id, username, password_hash, email, role, enabled, created_at, updated_at,
		       full_name, last_login_at, created_by
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		user := models.User{}
		var fullName sql.NullString
		var lastLoginAt sql.NullTime
		var createdBy uuid.NullUUID
		err := rows.Scan(
			&user.ID, &user.Username, &user.PasswordHash, &user.Email,
			&user.Role, &user.Enabled, &user.CreatedAt, &user.UpdatedAt,
			&fullName, &lastLoginAt, &createdBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		user.FullName = fullName.String
		if lastLoginAt.Valid {
			user.LastLoginAt = &lastLoginAt.Time
		}
		if createdBy.Valid {
			user.CreatedBy = &createdBy.UUID
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET username = $1, password_hash = $2, email = $3, role = $4, enabled = $5, updated_at = $6,
		    full_name = $7
		WHERE id = $8
	`

	result, err := r.db.conn.ExecContext(
		ctx, query,
		user.Username, user.PasswordHash, user.Email, user.Role, user.Enabled, user.UpdatedAt,
		user.FullName, user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateLastLoginAt updates the last_login_at timestamp for a user.
// It silently ignores errors if the column does not exist yet (migration pending).
func (r *UserRepository) UpdateLastLoginAt(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err := r.db.conn.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		// Gracefully ignore if the column doesn't exist yet (migration pending)
		if strings.Contains(err.Error(), "last_login_at") || strings.Contains(err.Error(), "column") {
			return nil
		}
		return fmt.Errorf("failed to update last_login_at: %w", err)
	}
	return nil
}

// UpdatePassword updates the password hash for a user
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	result, err := r.db.conn.ExecContext(ctx, query, passwordHash, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM users WHERE id = $1"

	result, err := r.db.conn.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
