package infrastructure

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitiatePostgres() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=myuser password=mypassword dbname=mydb port=5432 sslmode=disable"
	}
	dialector := postgres.Open(dsn)
	gormdb, err := gorm.Open(dialector)

	if err != nil {
		return nil, err
	}

	return gormdb, nil

}
