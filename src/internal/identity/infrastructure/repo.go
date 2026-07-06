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
	var user User
	tx := postDB.WithContext(ctx).Model(User{}).Where("email = ?", email).Find(&user)

	//TODO!: Convert to domain.User before returning
	return &user, tx.Error
}

func (postDB *PostgresUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := gorm.G[User](postDB.DB).Where("id = ?", id).First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	//TODO!: Convert to domain.User before returning

	return &user, nil
}

func (postDb *PostgresUserRepo) Create(ctx context.Context, user *domain.User) error {
	//TODO!: Convert to user infra layer User struct before passing it to Create()
	if err := gorm.G[User](postDb.DB).Create(ctx, user); err != nil {
		return err
	}

	return nil
}

func (postDb *PostgresUserRepo) Update(ctx context.Context, user *domain.User) error {
	//TODO!: Convert to infra layer user struct before passing to Save()
	tx := postDb.WithContext(ctx).Save(user)
	return tx.Error
}
