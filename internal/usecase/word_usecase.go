package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
)

// WordUsecase defines business logic for words.
type WordUsecase interface {
	Create(ctx context.Context, word *entity.Word) (*entity.Word, error)
	Update(ctx context.Context, word *entity.Word) (*entity.Word, error)
	Get(ctx context.Context, id int64) (*entity.Word, error)
	Lookup(ctx context.Context, lemma string, language entity.Language) (*entity.Word, error)
	List(ctx context.Context, filter *repository.ListWordQuery) ([]*entity.Word, int64, error)
	Delete(ctx context.Context, id int64) error
}

const (
	_defaultLanguage = entity.LanguageEnglish
	_defaultLimit    = int32(20)
	_maxLimit        = int32(10000)
)

type wordUsecase struct {
	repo repository.WordRepository
}

func NewWordUsecase(repo repository.WordRepository) WordUsecase {
	return &wordUsecase{repo: repo}
}

func (u *wordUsecase) Create(ctx context.Context, word *entity.Word) (*entity.Word, error) {
	norm, err := normalizeVocForUpsert(word)
	if err != nil {
		return nil, err
	}
	return u.repo.Create(ctx, norm)
}

func (u *wordUsecase) Update(ctx context.Context, word *entity.Word) (*entity.Word, error) {
	norm, err := normalizeVocForUpsert(word)
	if err != nil {
		return nil, err
	}
	if norm.ID <= 0 {
		return nil, entity.ErrInvalidVocID
	}
	return u.repo.Update(ctx, norm)
}

func (u *wordUsecase) Get(ctx context.Context, id int64) (*entity.Word, error) {
	if id <= 0 {
		return nil, entity.ErrInvalidVocID
	}
	return u.repo.GetByID(ctx, id)
}

func (u *wordUsecase) Lookup(ctx context.Context, lemma string, language entity.Language) (*entity.Word, error) {
	lemma = strings.TrimSpace(lemma)
	if lemma == "" {
		return nil, entity.ErrInvalidVocText
	}
	if language == entity.LanguageUnspecified {
		language = _defaultLanguage
	}
	v, err := u.repo.Lookup(ctx, lemma, language)
	if err != nil || v == nil {
		return v, err
	}
	if v.WordType == entity.WordTypeLemma {
		forms, ferr := u.repo.ListFormsByLemma(ctx, v.Text, v.Language)
		if ferr == nil {
			v.Forms = forms
		}
	}
	return v, nil
}

func (u *wordUsecase) List(ctx context.Context, query *repository.ListWordQuery) ([]*entity.Word, int64, error) {
	return u.repo.List(ctx, query)
}

func (u *wordUsecase) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return entity.ErrInvalidVocID
	}
	return u.repo.Delete(ctx, id)
}

func normalizeVocForUpsert(in *entity.Word) (*entity.Word, error) {
	if in == nil {
		return nil, errors.New("word payload required")
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil, entity.ErrInvalidVocText
	}
	out := *in
	out.Text = text
	if out.Language == entity.LanguageUnspecified {
		out.Language = _defaultLanguage
	}
	out.WordType = strings.TrimSpace(out.WordType)
	if out.WordType == "" {
		out.WordType = entity.WordTypeLemma
	}
	if out.WordType != entity.WordTypeLemma {
		if out.Lemma == nil || strings.TrimSpace(*out.Lemma) == "" {
			return nil, errors.New("lemma reference required for non-lemma entries")
		}
		lemma := strings.TrimSpace(*out.Lemma)
		out.Lemma = &lemma
	} else {
		out.Lemma = nil
	}

	return &out, nil
}
