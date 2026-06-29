package infrastructure

import (
	"context"
	"errors"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"gorm.io/gorm"
)

type PostgresUserRepo struct {
	*gorm.DB
}

func (postDB *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := gorm.G[domain.User](postDB.DB).Where("email = ?", email).First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (postDB *PostgresUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := gorm.G[domain.User](postDB.DB).Where("id = ?", id).First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (postDb *PostgresUserRepo) Create(ctx context.Context, user *domain.User) error {
	if err := gorm.G[domain.User](postDb.DB).Create(ctx, user); err != nil {
		return err
	}

	return nil
}

func (postDb *PostgresUserRepo) Update(ctx context.Context, user *domain.User) error {
	obj := postDb.WithContext(ctx).Save(user)
	if obj.Error != nil {
		return obj.Error
	}

	return nil
}
