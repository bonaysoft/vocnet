package repository

import (
	"context"

	"github.com/eslsoft/vocnet/internal/entity"
)

// ListLearnedLexemeQuery holds parameters for listing user lexemes.
type ListLearnedLexemeQuery struct {
	Pagination
	FilterOrder

	UserID int64
}

// LearnedLexemeRepository abstracts persistence for user lexemes to keep usecases storage agnostic.
type LearnedLexemeRepository interface {
	Create(ctx context.Context, lexeme *entity.LearnedLexeme) (*entity.LearnedLexeme, error)
	Update(ctx context.Context, lexeme *entity.LearnedLexeme) (*entity.LearnedLexeme, error)
	GetByID(ctx context.Context, userID, id int64) (*entity.LearnedLexeme, error)
	FindByTerm(ctx context.Context, userID int64, term string) (*entity.LearnedLexeme, error)
	List(ctx context.Context, filter *ListLearnedLexemeQuery) ([]entity.LearnedLexeme, int64, error)
	Delete(ctx context.Context, userID, id int64) error
}
