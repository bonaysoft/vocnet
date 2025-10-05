package repository

import (
	"context"

	"github.com/eslsoft/vocnet/internal/entity"
)

type ListWordQuery struct {
	Pagination
	FilterOrder
}

// WordRepository defines data access for word entries.
type WordRepository interface {
	Create(ctx context.Context, word *entity.Word) (*entity.Word, error)
	Update(ctx context.Context, word *entity.Word) (*entity.Word, error)
	GetByID(ctx context.Context, id int64) (*entity.Word, error)
	Lookup(ctx context.Context, text string, language entity.Language) (*entity.Word, error)
	List(ctx context.Context, filter *ListWordQuery) ([]*entity.Word, int64, error)
	Delete(ctx context.Context, id int64) error
	ListFormsByLemma(ctx context.Context, lemma string, language entity.Language) ([]entity.WordFormRef, error)
}
