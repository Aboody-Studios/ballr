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
	obj := postDb.Save(user)
	if obj.Error != nil {
		return obj.Error
	}

	return nil
}

func (postDb *PostgresUserRepo) GetUsernames(ctx context.Context, userIDs []string) (map[string]string, error) {
	type userRow struct {
		ID       string
		Username string
	}
	var userRowSlice = make([]userRow, 0, len(userIDs))

	obj := postDb.WithContext(ctx).Model(&domain.User{}).Where("id IN ?", userIDs).Select("id, username").Find(&userRowSlice)
	if obj.Error != nil {
		return nil, obj.Error
	}

	IDtoUsername := make(map[string]string, len(userIDs))

	for _, currUserRow := range userRowSlice {
		IDtoUsername[currUserRow.ID] = currUserRow.Username
	}

	return IDtoUsername, nil
}
