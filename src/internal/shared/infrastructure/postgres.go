package infrastructure

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitiatePostgres() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}
	dialector := postgres.Open(dsn)
	gormdb, err := gorm.Open(dialector)

	if err != nil {
		return nil, err
	}

	return gormdb, nil

}
