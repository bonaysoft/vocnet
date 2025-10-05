package repository

import (
	"context"

	"github.com/eslsoft/vocnet/internal/entity"
)

// ListUserWordQuery holds parameters for listing user words.
type ListUserWordQuery struct {
	Pagination
	FilterOrder

	UserID int64
}

// UserWordRepository abstracts persistence for user words to keep usecases storage agnostic.
type UserWordRepository interface {
	Create(ctx context.Context, userWord *entity.UserWord) (*entity.UserWord, error)
	Update(ctx context.Context, userWord *entity.UserWord) (*entity.UserWord, error)
	GetByID(ctx context.Context, userID, id int64) (*entity.UserWord, error)
	FindByWord(ctx context.Context, userID int64, word string) (*entity.UserWord, error)
	List(ctx context.Context, filter *ListUserWordQuery) ([]entity.UserWord, int64, error)
	Delete(ctx context.Context, userID, id int64) error
}
