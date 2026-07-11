package infrastructure

import (
	identityinfra "github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	analysisinfra "github.com/Aboody-Studios/ballr/src/internal/match/infrastructure"
	progressinfra "github.com/Aboody-Studios/ballr/src/internal/progress/infrastructure"
	coachinfra "github.com/Aboody-Studios/ballr/src/internal/coach/infrastructure"
	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&identityinfra.User{},
		&analysisinfra.Match{},
		&progressinfra.Progress{},
		&progressinfra.Achievement{},
		&progressinfra.EventLog{},
		&coachinfra.ChatMessage{},
		
	)
}
