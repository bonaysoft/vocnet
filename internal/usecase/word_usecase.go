package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/eslsoft/vocnet/internal/adapter/repository"
	"github.com/eslsoft/vocnet/internal/entity"
)

// WordUsecase defines business logic for words.
type WordUsecase interface {
	Lookup(ctx context.Context, lemma string, language string) (*entity.Voc, error)
}

const _defaultLanguage = "en"

type wordUsecase struct {
	repo repository.VocRepository
}

func NewWordUsecase(repo repository.VocRepository) WordUsecase {
	return &wordUsecase{repo: repo}
}

func (u *wordUsecase) Lookup(ctx context.Context, lemma string, language string) (*entity.Voc, error) {
	lemma = strings.TrimSpace(lemma)
	if lemma == "" {
		return nil, errors.New("lemma required")
	}
	if language == "" {
		language = _defaultLanguage
	}
	v, err := u.repo.Lookup(ctx, lemma, language)
	if err != nil || v == nil {
		return v, err
	}
	// Populate forms only if this entry itself is a lemma
	if v.VocType == "lemma" {
		forms, ferr := u.repo.ListFormsByLemma(ctx, v.Text, v.Language)
		if ferr == nil {
			v.Forms = forms
		}
	}
	return v, nil
}
