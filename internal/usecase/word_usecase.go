package usecase

import (
	"context"
	"errors"
	"fmt"
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
	if v.WordType == "lemma" {
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
		out.WordType = "lemma"
	}
	if out.WordType != "lemma" {
		if out.Lemma == nil || strings.TrimSpace(*out.Lemma) == "" {
			return nil, errors.New("lemma reference required for non-lemma entries")
		}
		lemma := strings.TrimSpace(*out.Lemma)
		out.Lemma = &lemma
	} else {
		out.Lemma = nil
	}
	if len(out.Tags) == 0 {
		out.Tags = nil
	}
	out.Phrases = normalizeStringSlice(out.Phrases)
	out.Sentences = normalizeSentences(out.Sentences)
	out.Relations = normalizeWordRelations(out.Relations)
	out.Phonetics = normalizePhonetics(out.Phonetics)
	return &out, nil
}

func normalizePhonetics(in []entity.WordPhonetic) []entity.WordPhonetic {
	if len(in) == 0 {
		return nil
	}
	out := make([]entity.WordPhonetic, 0, len(in))
	for _, p := range in {
		ipa := strings.TrimSpace(p.IPA)
		if ipa == "" {
			continue
		}
		dialect := strings.TrimSpace(p.Dialect)
		out = append(out, entity.WordPhonetic{IPA: ipa, Dialect: dialect})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeSentences(in []entity.Sentence) []entity.Sentence {
	if len(in) == 0 {
		return nil
	}
	out := make([]entity.Sentence, 0, len(in))
	for _, s := range in {
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		out = append(out, entity.Sentence{
			Text:      text,
			Source:    s.Source,
			SourceRef: strings.TrimSpace(s.SourceRef),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeWordRelations(in []entity.WordRelation) []entity.WordRelation {
	if len(in) == 0 {
		return nil
	}
	out := make([]entity.WordRelation, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, rel := range in {
		word := strings.TrimSpace(rel.Word)
		if word == "" {
			continue
		}
		key := strings.ToLower(word) + "|" + fmt.Sprint(rel.RelationType)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entity.WordRelation{Word: word, RelationType: rel.RelationType})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
