package application

import (
	"context"
	"errors"

	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	"gorm.io/gorm"
)

var ErrEmailAlreadyExists = errors.New("Email already exists")

func (s *Service) RegisterUser(user *UserDTO, ctx context.Context) (*domain.User, error) {
	hashedPass, err := hashPass(user.Password)
	if err != nil {
		return nil, err
	}

	userfound, findErr := s.UserRepo.FindByEmail(ctx, user.Email)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return nil, findErr
	}

	if userfound != nil {
		return nil, ErrEmailAlreadyExists
	}

	domainUser, newUserError := domain.NewUser("123", user.Email, hashedPass, user.FullName, user.BirthDate, user.Position, user.Footedness, user.Goals)

	if newUserError != nil {
		return nil, newUserError
	}

	// TODO!: Persist user via repository when database is configured.

	return domainUser, nil
}
