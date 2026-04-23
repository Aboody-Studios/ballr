package application

import (
	"github.com/Aboody-Studios/ballr/src/internal/identity/domain"
)

// This struct is here because adding it to identity/http causes an import cycle.

func (s *Service) RegisterUser(user *SignupRequest) (*domain.User, error) {
	hashedPass, err := hashPass(user.Password)
	if err != nil {
		return nil, err
	}

	//TODO!: Use FindByEmail here to check for email duplicates before registration before creating the user.

	domainUser, newUserError := domain.NewUser("123", user.Email, hashedPass, user.FullName, user.BirthDate, user.Position, user.Footedness, user.Goals)

	if newUserError != nil {
		return nil, newUserError
	}

	// TODO!: Persist user via repository when database is configured.

	return domainUser, nil
}
