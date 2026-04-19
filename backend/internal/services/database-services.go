package services

import (
	"github.com/Aboody-Studios/ballr/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func PersistToDatabase(user *models.User) error {
	// Move the hashing to service layer
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	//TODO: save user to database
}
