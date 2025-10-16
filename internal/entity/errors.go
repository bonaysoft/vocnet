package entity

import "errors"

// Domain errors for user entity and related aggregates.
var (
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidUserName        = errors.New("invalid user name")
	ErrInvalidUserEmail       = errors.New("invalid user email")
	ErrUserAlreadyExists      = errors.New("user already exists")
	ErrInvalidUserID          = errors.New("invalid user ID")
	ErrLearnedWordNotFound    = errors.New("user word not found")
	ErrDuplicateLearnedWord   = errors.New("user word already exists")
	ErrInvalidLearnedWordText = errors.New("invalid user word text")
	ErrVocNotFound            = errors.New("word not found")
	ErrInvalidVocID           = errors.New("invalid word id")
	ErrInvalidVocText         = errors.New("invalid word text")
	ErrDuplicateWord          = errors.New("word already exists")
)
