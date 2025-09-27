package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eslsoft/vocnet/internal/entity"
	db "github.com/eslsoft/vocnet/internal/infrastructure/database/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// VocRepository defines data access for vocs.
type VocRepository interface {
	Lookup(ctx context.Context, text string, language string) (*entity.Voc, error)
	ListFormsByLemma(ctx context.Context, lemma string, language string) ([]entity.VocFormRef, error)
}

type vocRepository struct{ q *db.Queries }

func NewVocRepository(q *db.Queries) VocRepository { return &vocRepository{q: q} }

func (r *vocRepository) Lookup(ctx context.Context, text string, language string) (*entity.Voc, error) {
	rec, err := r.q.LookupVoc(ctx, db.LookupVocParams{Lower: text, Language: language})
	if err != nil {
		if fmt.Sprintf("%v", err) == "no rows in result set" { // pgx/v5 style error compare
			return nil, nil
		}
		return nil, fmt.Errorf("lookup voc: %w", err)
	}

	voc := &entity.Voc{
		ID:        rec.ID,
		Text:      rec.Text,
		Language:  rec.Language,
		VocType:   rec.VocType,
		Phonetic:  rec.Phonetic.String,
		Tags:      rec.Tags,
		CreatedAt: rec.CreatedAt.Time,
	}
	if rec.Lemma.Valid {
		lemma := rec.Lemma.String
		voc.Lemma = &lemma
	}

	if len(rec.Meanings) > 0 {
		var ms []entity.VocMeaning
		if err := json.Unmarshal(rec.Meanings, &ms); err == nil {
			voc.Meanings = ms
		}
	}

	return voc, nil
}

// ListFormsByLemma returns all non-lemma forms (text + voc_type) for a lemma.
func (r *vocRepository) ListFormsByLemma(ctx context.Context, lemma string, language string) ([]entity.VocFormRef, error) {
	// Reuse ListInflections (which currently returns all rows where lemma = $2)
	// We filter out any protective self rows (shouldn't exist since lemma field is null/empty on lemma row).
	rows, err := r.q.ListInflections(ctx, db.ListInflectionsParams{Language: language, Lemma: pgtype.Text{String: lemma, Valid: true}})
	if err != nil {
		return nil, fmt.Errorf("list forms: %w", err)
	}
	forms := make([]entity.VocFormRef, 0, len(rows))
	for _, rrow := range rows {
		// Skip if voc_type == 'lemma' just in case
		if rrow.VocType == "lemma" {
			continue
		}
		forms = append(forms, entity.VocFormRef{Text: rrow.Text, VocType: rrow.VocType})
	}
	return forms, nil
}
