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
	tx := postDB.WithContext(ctx).Model(User{}).Where("email = ?", email).First(&user)

	userDomain := FromUserInfraToDomain(user)
	return userDomain, tx.Error
}

func (postDB *PostgresUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := gorm.G[User](postDB.DB).Where("id = ?", id).First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	userDomain := FromUserInfraToDomain(user)

	return userDomain, nil
}

func (postDb *PostgresUserRepo) Create(ctx context.Context, user *domain.User) error {
	userInfra := FromUserDomainToInfra(user)
	if err := gorm.G[User](postDb.DB).Create(ctx, userInfra); err != nil {
		return err
	}

	return nil
}

func (postDb *PostgresUserRepo) Update(ctx context.Context, user *domain.User) error {
	userInfra := FromUserDomainToInfra(user)
	tx := postDb.WithContext(ctx).Model(userInfra).Save(userInfra)

	return tx.Error
}
