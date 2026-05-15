package infrastructure

import (
	analysisdomain "github.com/Aboody-Studios/ballr/src/internal/match/domain"
	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	identitydomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	progressdomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&identitydomain.User{},
		&analysisdomain.Match{},
		&coachdomain.Conversation{},
		&progressdomain.Progress{},
		&progressdomain.Achievement{},
		&progressdomain.EventLog{},
	)
}
