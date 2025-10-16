package repository

import (
	"context"

	"github.com/eslsoft/vocnet/internal/entity"
)

// ListLearnedWordQuery holds parameters for listing user words.
type ListLearnedWordQuery struct {
	Pagination
	FilterOrder

	UserID int64
}

// LearnedWordRepository abstracts persistence for user words to keep usecases storage agnostic.
type LearnedWordRepository interface {
	Create(ctx context.Context, LearnedWord *entity.LearnedWord) (*entity.LearnedWord, error)
	Update(ctx context.Context, LearnedWord *entity.LearnedWord) (*entity.LearnedWord, error)
	GetByID(ctx context.Context, userID, id int64) (*entity.LearnedWord, error)
	FindByWord(ctx context.Context, userID int64, word string) (*entity.LearnedWord, error)
	List(ctx context.Context, filter *ListLearnedWordQuery) ([]entity.LearnedWord, int64, error)
	Delete(ctx context.Context, userID, id int64) error
}
