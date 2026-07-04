package infrastructure

import (
	"time"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

type User struct {
	ID            string
	Email         string `gorm:"uniqueIndex"`
	Username      string
	AvatarURL     string
	OAuthProvider string
	FullName      string
	BirthDate     time.Time
	Footedness    domain.Footedness
	CreatedAt     time.Time
	TrainingDays  []domain.TrainingDay `gorm:"type:jsonb;serializer:json"`
	Timezone      string
}
