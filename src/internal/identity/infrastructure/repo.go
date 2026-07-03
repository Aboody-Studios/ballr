package infrastructure

import (
	"context"
	"errors"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"gorm.io/gorm"
)

//TODO!: Replace Generic usage with traditional API

type PostgresUserRepo struct {
	*gorm.DB
}

func (postDB *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	tx := postDB.WithContext(ctx).Model(User{}).Where("email = ?", email).Find(&user)

	return &user, tx.Error
}

func (postDB *PostgresUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// TODO!: Change to to accept result as domain.User and then fill infrastracture.User variable
	user, err := gorm.G[User](postDB.DB).Where("id = ?", id).First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (postDb *PostgresUserRepo) Create(ctx context.Context, user *User) error {
	if err := gorm.G[User](postDb.DB).Create(ctx, user); err != nil {
		return err
	}

	return nil
}

func (postDb *PostgresUserRepo) Update(ctx context.Context, user *User) error {
	tx := postDb.WithContext(ctx).Save(user)
	return tx.Error
}
