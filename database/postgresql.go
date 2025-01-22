package database

import (
	"RoyDental/models"
	"context"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance.
var DB *gorm.DB

// InitDB initializes the database connection and configures it.
func InitDB(ctx context.Context, dsn string) (*gorm.DB, error) {
	var err error

	// Configure logging level based on environment
	logMode := logger.Silent
	if os.Getenv("ENV") == "development" {
		logMode = logger.Info
	}

	// Open the database connection
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: false,
		PrepareStmt:                              true,
		Logger:                                   logger.Default.LogMode(logMode),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to open database connection")
	}

	// Configure connection pool
	if err := configureConnectionPool(); err != nil {
		return nil, err
	}

	// Test the database connection
	if err := testDatabaseConnection(ctx); err != nil {
		return nil, err
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return nil, err
	}

	// Seed initial data
	if err := seedInitialData(); err != nil {
		return nil, err
	}

	log.Println("Database initialized successfully.")
	return DB, nil
}

// configureConnectionPool sets up the connection pool settings for the database.
func configureConnectionPool() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return errors.Wrap(err, "failed to get sql.DB from GORM")
	}
	sqlDB.SetMaxOpenConns(40)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	return nil
}

// testDatabaseConnection verifies that the database connection is functional.
func testDatabaseConnection(ctx context.Context) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return errors.Wrap(err, "failed to get sql.DB from GORM")
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return errors.Wrap(err, "failed to ping database")
	}
	return nil
}

// runMigrations performs database schema migrations.
func runMigrations() error {
	return DB.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.User{},
		&models.Doctor{},
		&models.Patient{},
		&models.EmergencyContact{},
		&models.InsuranceCompany{},
		&models.Examination{},
		&models.Billing{},
		&models.TreatmentPlan{},
		&models.Appointment{},
	)
}

// seedInitialData populates the database with initial data.
func seedInitialData() error {
	if err := models.SeedRoles(DB); err != nil {
		return errors.Wrap(err, "failed to seed roles")
	}
	if err := models.SeedPermissions(DB); err != nil {
		return errors.Wrap(err, "failed to seed permissions")
	}
	if err := models.SeedRolePermissions(DB); err != nil {
		return errors.Wrap(err, "failed to seed role permissions")
	}
	return nil
}

// LoadEnvConfig retrieves configuration values from environment variables.
func LoadEnvConfig() (string, error) {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		return "", errors.New("missing DB_URL environment variable")
	}
	return dsn, nil
}
