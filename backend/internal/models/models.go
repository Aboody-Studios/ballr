package models

import (
	"github.com/golang-jwt/jwt/v5"
)

type Video struct {
	Name string `json:"name"`
	Size uint64 `json:"size"`
}

type User struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type JWTCustomClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}
