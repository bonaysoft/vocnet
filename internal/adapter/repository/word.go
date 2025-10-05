package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	db "github.com/eslsoft/vocnet/internal/infrastructure/database/db"
	"github.com/eslsoft/vocnet/internal/repository"
	"github.com/eslsoft/vocnet/pkg/filterexpr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type wordRepository struct{ q *db.Queries }

func NewWordRepository(q *db.Queries) repository.WordRepository { return &wordRepository{q: q} }

func (r *wordRepository) Create(ctx context.Context, word *entity.Word) (*entity.Word, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	params, err := toCreateWordParams(word)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateWord(ctx, params)
	if err != nil {
		return nil, translateWordError(err)
	}
	return mapDBWord(row), nil
}

func (r *wordRepository) Update(ctx context.Context, word *entity.Word) (*entity.Word, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	params, err := toUpdateWordParams(word)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateWord(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrVocNotFound
		}
		return nil, translateWordError(err)
	}
	return mapDBWord(row), nil
}

func (r *wordRepository) GetByID(ctx context.Context, id int64) (*entity.Word, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row, err := r.q.GetWordByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrVocNotFound
		}
		return nil, fmt.Errorf("get word: %w", err)
	}
	return mapDBWord(row), nil
}

func (r *wordRepository) Lookup(ctx context.Context, text string, language entity.Language) (*entity.Word, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rec, err := r.q.LookupWord(ctx, db.LookupWordParams{Lower: text, Language: entity.NormalizeLanguage(language).Code()})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup word: %w", err)
	}
	return mapDBWord(rec), nil
}

func (r *wordRepository) List(ctx context.Context, query *repository.ListWordQuery) ([]*entity.Word, int64, error) {
	var p db.ListWordsParams
	if err := filterexpr.Bind(query, &p, listWordsSchema); err != nil {
		return nil, 0, err
	}

	p.Offset = query.Offset()
	p.Limit = query.PageSize
	rows, err := r.q.ListWords(ctx, p)
	if err != nil {
		return nil, 0, fmt.Errorf("list words: %w", err)
	}
	words := make([]*entity.Word, 0, len(rows))
	for _, row := range rows {
		words = append(words, mapDBWord(row))
	}
	total, err := r.q.CountWords(ctx, db.CountWordsParams{
		Language: p.Language,
		Keyword:  p.Keyword,
		WordType: p.WordType,
		Words:    p.Words,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("count words: %w", err)
	}
	return words, total, nil
}

func (r *wordRepository) Delete(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	affected, err := r.q.DeleteWord(ctx, id)
	if err != nil {
		return fmt.Errorf("delete word: %w", err)
	}
	if affected == 0 {
		return entity.ErrVocNotFound
	}
	return nil
}

// ListFormsByLemma returns all non-lemma forms (text + voc_type) for a lemma.
func (r *wordRepository) ListFormsByLemma(ctx context.Context, lemma string, language entity.Language) ([]entity.WordFormRef, error) {
	rows, err := r.q.ListInflections(ctx, db.ListInflectionsParams{Language: entity.NormalizeLanguage(language).Code(), Lemma: pgtype.Text{String: lemma, Valid: lemma != ""}})
	if err != nil {
		return nil, fmt.Errorf("list forms: %w", err)
	}
	forms := make([]entity.WordFormRef, 0, len(rows))
	for _, row := range rows {
		if row.WordType == "lemma" {
			continue
		}
		forms = append(forms, entity.WordFormRef{Text: row.Text, WordType: row.WordType})
	}
	return forms, nil
}

func mapDBWord(row db.Word) *entity.Word {
	word := &entity.Word{
		ID:          row.ID,
		Text:        row.Text,
		Language:    entity.ParseLanguage(row.Language),
		WordType:    row.WordType,
		Phonetics:   row.Phonetics,
		Definitions: row.Meanings,
		Tags:        row.Tags,
		Phrases:     row.Phrases,
		Sentences:   row.Sentences,
		Relations:   row.Relations,
		CreatedAt:   timeValue(row.CreatedAt),
		UpdatedAt:   timeValue(row.UpdatedAt),
	}
	if row.Lemma.Valid {
		lemma := row.Lemma.String
		word.Lemma = &lemma
	}
	return word
}

func toCreateWordParams(word *entity.Word) (db.CreateWordParams, error) {
	return db.CreateWordParams{
		Text:      word.Text,
		Language:  entity.NormalizeLanguage(word.Language).Code(),
		WordType:  defaultWordType(word.WordType),
		Lemma:     stringPtrToPgText(word.Lemma),
		Phonetics: word.Phonetics,
		Meanings:  word.Definitions,
		Tags:      word.Tags,
		Phrases:   word.Phrases,
		Sentences: word.Sentences,
		Relations: word.Relations,
	}, nil
}

func toUpdateWordParams(word *entity.Word) (db.UpdateWordParams, error) {
	return db.UpdateWordParams{
		ID:        word.ID,
		Text:      word.Text,
		Language:  entity.NormalizeLanguage(word.Language).Code(),
		WordType:  defaultWordType(word.WordType),
		Lemma:     stringPtrToPgText(word.Lemma),
		Phonetics: word.Phonetics,
		Meanings:  word.Definitions,
		Tags:      word.Tags,
		Phrases:   word.Phrases,
		Sentences: word.Sentences,
		Relations: word.Relations,
	}, nil
}

func defaultWordType(vt string) string {
	if vt == "" {
		return "lemma"
	}
	return vt
}

func stringPtrToPgText(val *string) pgtype.Text {
	if val == nil {
		return pgtype.Text{Valid: false}
	}
	return stringToPgText(*val)
}

func stringToPgText(val string) pgtype.Text {
	if val == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: val, Valid: true}
}

func timeValue(ts pgtype.Timestamptz) (t time.Time) {
	if ts.Valid {
		return ts.Time
	}
	return
}

func translateWordError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return entity.ErrDuplicateWord
		case "23503":
			return entity.ErrVocNotFound
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.ErrVocNotFound
	}
	return err
}
