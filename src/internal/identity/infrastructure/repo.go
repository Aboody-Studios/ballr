package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"gorm.io/gorm"
)

type PostgresUserRepo struct {
	*gorm.DB
}

func (postDB *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := gorm.G[domain.User](postDB.DB).Where("email = ?", email).First(ctx)

	if err != nil {
		return nil, err
	}

	return &user, nil
}


