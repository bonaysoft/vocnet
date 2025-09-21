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
	Lookup(ctx context.Context, lemma string, language string) (*entity.Word, error)
}

type wordUsecase struct {
	repo            repository.WordRepository
	defaultLanguage string
}

func NewWordUsecase(repo repository.WordRepository, defaultLanguage string) WordUsecase {
	if defaultLanguage == "" {
		defaultLanguage = "en"
	}
	return &wordUsecase{repo: repo, defaultLanguage: defaultLanguage}
}

func (u *wordUsecase) Lookup(ctx context.Context, lemma string, language string) (*entity.Word, error) {
	lemma = strings.TrimSpace(lemma)
	if lemma == "" {
		return nil, errors.New("lemma required")
	}
	if language == "" {
		language = u.defaultLanguage
	}
	return u.repo.Lookup(ctx, lemma, language)
}
