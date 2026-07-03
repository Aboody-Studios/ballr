package infrastructure

import (
	identitydomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	analysisdomain "github.com/Aboody-Studios/ballr/src/internal/match/domain"
	progressdomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&identitydomain.User{},
		&analysisdomain.Match{},
		&progressdomain.Progress{},
		&progressdomain.Achievement{},
		&progressdomain.EventLog{},
	)
}
