package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection with connection pooling
func NewDB(cfg Config) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(time.Hour)
	conn.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// HealthCheck verifies database connectivity
func (db *DB) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.conn.PingContext(ctx)
}

// RunMigrations applies pending database migrations with validation and rollback support
func (db *DB) RunMigrations(migrationsPath string) error {
	log.Println("Starting database migration process...")

	driver, err := postgres.WithInstance(db.conn, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	// Get current schema version before migration
	currentVersion, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		log.Printf("ERROR: Failed to get current schema version: %v", err)
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	// Check if database is in dirty state (previous migration failed)
	if dirty {
		log.Printf("ERROR: Database schema is in dirty state at version %d", currentVersion)
		log.Printf("Attempting to force schema version to clean state...")

		// Force the version to clean state to allow rollback
		if err := m.Force(int(currentVersion)); err != nil {
			log.Printf("ERROR: Failed to force clean schema version: %v", err)
			return fmt.Errorf("database schema is in dirty state and cannot be recovered: %w", err)
		}

		log.Printf("Schema version forced to clean state, attempting rollback...")

		// Rollback the failed migration
		if err := m.Steps(-1); err != nil {
			log.Printf("ERROR: Failed to rollback failed migration: %v", err)
			return fmt.Errorf("failed to rollback failed migration: %w", err)
		}

		log.Printf("Successfully rolled back failed migration from version %d", currentVersion)

		// Get new current version after rollback
		currentVersion, dirty, err = m.Version()
		if err != nil && err != migrate.ErrNilVersion {
			return fmt.Errorf("failed to get schema version after rollback: %w", err)
		}
	}

	if err == migrate.ErrNilVersion {
		log.Println("Database schema is empty, will apply all migrations")
		currentVersion = 0
	} else {
		log.Printf("Current database schema version: %d", currentVersion)
	}

	// Validate schema version is compatible
	if err := db.validateSchemaVersion(currentVersion); err != nil {
		log.Printf("ERROR: Schema version validation failed: %v", err)
		return fmt.Errorf("schema version incompatible: %w", err)
	}

	log.Println("Schema version validation passed")

	// Apply all pending migrations
	log.Println("Applying pending migrations...")
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No pending migrations to apply")
			return nil
		}

		log.Printf("ERROR: Migration failed: %v", err)
		log.Println("Attempting automatic rollback of failed migration...")

		// Attempt to rollback the failed migration
		if rollbackErr := m.Steps(-1); rollbackErr != nil {
			log.Printf("ERROR: Automatic rollback failed: %v", rollbackErr)
			return fmt.Errorf("migration failed and rollback failed: migration error: %w, rollback error: %v", err, rollbackErr)
		}

		log.Println("Successfully rolled back failed migration")
		return fmt.Errorf("migration failed and was rolled back: %w", err)
	}

	// Get final schema version
	finalVersion, _, err := m.Version()
	if err != nil {
		log.Printf("WARNING: Failed to get final schema version: %v", err)
	} else {
		log.Printf("Database migrations completed successfully. Final schema version: %d", finalVersion)
	}

	return nil
}

// validateSchemaVersion checks if the current schema version is compatible with this application version
func (db *DB) validateSchemaVersion(currentVersion uint) error {
	// Define the expected schema version range for this application version
	// This should be updated when the application requires specific schema versions
	const (
		minSupportedVersion uint = 0  // Minimum schema version this app can work with
		maxSupportedVersion uint = 13 // Maximum schema version (current latest migration)
	)

	// If schema is newer than what we support, it's incompatible
	if currentVersion > maxSupportedVersion {
		return fmt.Errorf(
			"database schema version %d is newer than supported version %d. Please upgrade the application",
			currentVersion,
			maxSupportedVersion,
		)
	}

	// Schema version 0 (empty database) is always acceptable - migrations will bring it up to date
	if currentVersion == 0 {
		return nil
	}

	// If schema is older than minimum supported, it's incompatible
	if currentVersion < minSupportedVersion {
		return fmt.Errorf(
			"database schema version %d is older than minimum supported version %d. Please run migrations",
			currentVersion,
			minSupportedVersion,
		)
	}

	return nil
}

// BeginTx starts a new transaction with retry logic for deadlocks
func (db *DB) BeginTx() (*sql.Tx, error) {
	maxRetries := 3
	var tx *sql.Tx
	var err error

	for i := 0; i < maxRetries; i++ {
		tx, err = db.conn.Begin()
		if err == nil {
			return tx, nil
		}

		// Check if error is a deadlock
		if isDeadlock(err) && i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			continue
		}
		break
	}

	return nil, fmt.Errorf("failed to begin transaction after %d retries: %w", maxRetries, err)
}

// isDeadlock checks if the error is a deadlock error
func isDeadlock(err error) bool {
	// PostgreSQL deadlock error code is 40P01
	return err != nil && (err.Error() == "pq: deadlock detected" ||
		strings.Contains(err.Error(), "40P01"))
}

// Conn returns the underlying sql.DB connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// WithTransaction executes a function within a transaction with retry logic
func (db *DB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	maxRetries := 3
	var err error

	for i := 0; i < maxRetries; i++ {
		tx, err := db.conn.BeginTx(ctx, nil)
		if err != nil {
			if isDeadlock(err) && i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		err = fn(tx)
		if err != nil {
			tx.Rollback()
			if isDeadlock(err) && i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
				continue
			}
			return err
		}

		if err := tx.Commit(); err != nil {
			if isDeadlock(err) && i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	}

	return fmt.Errorf("transaction failed after %d retries: %w", maxRetries, err)
}
