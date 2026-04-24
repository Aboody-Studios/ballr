package domain

import (
	"context"
)

// UserRepository defines the contract for user persistence operations.
// This interface is implemented in the infrastructure layer to maintain
// dependency inversion - the domain knows what it needs, not how it's stored.
type UserRepository interface {
	// Save persists a new user to the database.
	Save(ctx context.Context, user *User) error

	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindByID retrieves a user by their UUID.
	FindByID(ctx context.Context, id string) (*User, error)

	// Update modifies an existing user's data.
	Update(ctx context.Context, user *User) error
}

// ErrUserNotFound is returned when a user lookup fails.
// Defined here to avoid infrastructure dependencies in domain.
// Circular shit yk.
var ErrUserNotFound = errUserNotFound{}

type errUserNotFound struct{}

func (errUserNotFound) Error() string { return "user not found" }
