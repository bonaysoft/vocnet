package entity

import "errors"

// Domain errors for user entity and related aggregates.
var (
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidUserName     = errors.New("invalid user name")
	ErrInvalidUserEmail    = errors.New("invalid user email")
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidUserID       = errors.New("invalid user ID")
	ErrUserWordNotFound    = errors.New("user word not found")
	ErrDuplicateUserWord   = errors.New("user word already exists")
	ErrInvalidUserWordText = errors.New("invalid user word text")
)
