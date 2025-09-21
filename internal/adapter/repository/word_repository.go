package repository

import (
    "context"
    "fmt"

    "github.com/eslsoft/vocnet/internal/entity"
    db "github.com/eslsoft/vocnet/internal/infrastructure/database/db"
)

// WordRepository defines data access for words.
type WordRepository interface {
	Lookup(ctx context.Context, lemma string, language string) (*entity.Word, error)
}

type wordRepository struct{ q *db.Queries }

func NewWordRepository(q *db.Queries) WordRepository { return &wordRepository{q: q} }

func (r *wordRepository) Lookup(ctx context.Context, lemma string, language string) (*entity.Word, error) {
	rec, err := r.q.LookupWord(ctx, db.LookupWordParams{Lower: lemma, Language: language})
	if err != nil {
		if fmt.Sprintf("%v", err) == "no rows in result set" { // pgx/v5 style error compare
			return nil, nil
		}
		return nil, fmt.Errorf("lookup word: %w", err)
	}
    return &entity.Word{
        ID:          rec.ID,
        Lemma:       rec.Lemma,
        Language:    rec.Language,
        Phonetic:    rec.Phonetic.String,
        POS:         rec.Pos.String,
        Definition:  rec.Definition.String,
        Translation: rec.Translation.String,
        Exchange:    rec.Exchange.String,
        Tags:        rec.Tags,
        CreatedAt:   rec.CreatedAt.Time,
    }, nil
}
